package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chunisupport/chunisupport-song-batch/internal/config"
	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// ConsolidationOptions は統合処理の挙動を制御します。
type ConsolidationOptions struct {
	MajorUpdate bool
}

// ConsolidationSources は統合に利用するソースデータを保持します。
type ConsolidationSources struct {
	Official        *importer.OfficialData
	AdditionalSongs *importer.AdditionalSongsData
	St1027          *importer.St1027Data
	Natua           *importer.NatuaData
	Mainframe       *importer.MainframeData
	OtogeDb         *importer.OtogeDbData
}

// ConsolidationService はデータソースの統合処理を管理します。
type ConsolidationService struct {
	db             domainrepo.ExtendedDBExecutor
	difficultyRepo domainrepo.DifficultyRepository
	genreRepo      domainrepo.GenreRepository
	pwPepper       string
	datasources    []config.DatasourceEntry
	opts           ConsolidationOptions
	sources        ConsolidationSources
}

// NewConsolidationService は新しいConsolidationServiceのインスタンスを生成します。
func NewConsolidationService(
	db domainrepo.ExtendedDBExecutor,
	difficultyRepo domainrepo.DifficultyRepository,
	genreRepo domainrepo.GenreRepository,
	pwPepper string,
	datasources []config.DatasourceEntry,
	opts ConsolidationOptions,
	sources ConsolidationSources,
) *ConsolidationService {
	return &ConsolidationService{
		db:             db,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		pwPepper:       pwPepper,
		datasources:    datasources,
		opts:           opts,
		sources:        sources,
	}
}

// ConsolidateAll はすべてのアクティブなデータソースを統合します。
func (s *ConsolidationService) ConsolidateAll(ctx context.Context) error {
	workspace, err := s.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return nil
	}
	defer workspace.Close()
	return s.SyncWorkspace(ctx, workspace, s.db)
}

// ConsolidateBySource は指定されたデータソースのみを統合します。
func (s *ConsolidationService) ConsolidateBySource(ctx context.Context, sourceType string) error {
	var target []config.DatasourceEntry
	for _, ds := range s.datasources {
		if ds.Name == sourceType {
			target = append(target, ds)
			break
		}
	}
	if len(target) == 0 {
		slog.Warn("Requested datasource not found", "source", sourceType)
		return nil
	}
	workspace, err := s.buildWorkspace(ctx, target)
	if err != nil {
		return err
	}
	if workspace == nil {
		return nil
	}
	defer workspace.Close()
	return s.SyncWorkspace(ctx, workspace, s.db)
}

// BuildWorkspace はアクティブなデータソースを SQLite ワークスペースに取り込みます。
func (s *ConsolidationService) BuildWorkspace(ctx context.Context) (*songchart.SongChartWorkspace, error) {
	return s.buildWorkspace(ctx, s.datasources)
}

// SyncWorkspace は準備済みワークスペースを MySQL に同期します。
func (s *ConsolidationService) SyncWorkspace(ctx context.Context, workspace *songchart.SongChartWorkspace, mysql domainrepo.DBExecutor) error {
	if workspace == nil {
		return nil
	}
	syncOpts := songchart.SyncOptions{
		MajorUpdate: s.opts.MajorUpdate,
	}

	if err := workspace.SyncToMySQL(ctx, mysql, syncOpts); err != nil {
		return fmt.Errorf("failed to sync workspace to MySQL: %w", err)
	}
	return nil
}

func (s *ConsolidationService) buildWorkspace(ctx context.Context, datasources []config.DatasourceEntry) (*songchart.SongChartWorkspace, error) {
	active := make([]config.DatasourceEntry, 0, len(datasources))
	for _, ds := range datasources {
		if s.opts.MajorUpdate {
			if ds.Name != "official" && ds.Name != "additional_songs" {
				slog.Info("Skipping datasource in major update mode", "type", ds.Name)
				continue
			}
		}

		if !ds.Active {
			slog.Warn("Skipping inactive datasource for consolidation", "type", ds.Name)
			continue
		}
		active = append(active, ds)
	}

	if len(active) == 0 {
		slog.Warn("No active datasources to consolidate")
		return nil, nil
	}

	workspace, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create song chart workspace: %w", err)
	}

	for _, ds := range active {
		if err := s.consolidateSource(ctx, workspace, ds.Name); err != nil {
			_ = workspace.Close()
			return nil, err
		}
	}

	return workspace, nil
}

func (s *ConsolidationService) consolidateSource(ctx context.Context, workspace *songchart.SongChartWorkspace, name string) error {
	switch name {
	case "official":
		if s.sources.Official == nil {
			slog.Warn("Skipping official consolidation due to missing data")
			return nil
		}
		consolidator := NewOfficialConsolidator(s.db, s.difficultyRepo, s.genreRepo, workspace, s.pwPepper, s.sources.Official)
		return consolidator.Consolidate(ctx)
	case "additional_songs":
		if s.sources.AdditionalSongs == nil {
			slog.Warn("Skipping additional_songs consolidation due to missing data")
			return nil
		}
		consolidator := NewAdditionalSongsConsolidator(s.db, s.difficultyRepo, s.genreRepo, workspace, s.pwPepper, s.sources.AdditionalSongs)
		return consolidator.Consolidate(ctx)
	case "natua":
		if s.sources.Natua == nil {
			slog.Warn("Skipping natua consolidation due to missing data")
			return nil
		}
		consolidator := NewNatuaConsolidator(workspace, s.sources.Natua)
		return consolidator.Consolidate(ctx)
	case "st1027":
		if s.sources.St1027 == nil {
			slog.Warn("Skipping st1027 consolidation due to missing data")
			return nil
		}
		consolidator := NewSt1027Consolidator(workspace, s.sources.St1027)
		return consolidator.Consolidate(ctx)
	case "mainframe":
		if s.sources.Mainframe == nil {
			slog.Warn("Skipping mainframe consolidation due to missing data")
			return nil
		}
		consolidator := NewMainframeConsolidator(workspace, s.sources.Mainframe)
		return consolidator.Consolidate(ctx)
	case "otoge_db":
		if s.sources.OtogeDb == nil {
			slog.Warn("Skipping otoge-db consolidation due to missing data")
			return nil
		}
		consolidator := NewOtogeDbConsolidator(workspace, s.sources.OtogeDb)
		return consolidator.Consolidate(ctx)
	default:
		slog.Warn("Unknown datasource requested for consolidation", "type", name)
		return nil
	}
}
