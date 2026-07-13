package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-song-batch/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/workspace/songchart"
)

// ConsolidationOptions は統合処理の挙動を制御します。
type ConsolidationOptions struct {
	MajorUpdate            bool
	FillMissingReleaseDate bool // 特定フラグ有効時、データソース・MySQL両方に日付のない新規楽曲へ実行日(JST)を補完
}

// ConsolidationSources は統合に利用するソースデータを保持します。
type ConsolidationSources struct {
	Official        *importer.OfficialData
	AdditionalSongs *importer.AdditionalSongsData
	St1027          *importer.St1027Data
	Mainframe       *importer.MainframeData
	OtogeDb         *importer.OtogeDbData
}

// ConsolidationService はデータソースの統合処理を管理します。
type ConsolidationService struct {
	db             domainrepo.ExtendedDBExecutor
	difficultyRepo domainrepo.DifficultyRepository
	genreRepo      domainrepo.GenreRepository
	courseRepo     domainrepo.CourseRepository
	pwPepper       string
	datasources    []string
	opts           ConsolidationOptions
	sources        ConsolidationSources
}

// NewConsolidationService は新しいConsolidationServiceのインスタンスを生成します。
func NewConsolidationService(
	db domainrepo.ExtendedDBExecutor,
	difficultyRepo domainrepo.DifficultyRepository,
	genreRepo domainrepo.GenreRepository,
	courseRepo domainrepo.CourseRepository,
	pwPepper string,
	datasources []string,
	opts ConsolidationOptions,
	sources ConsolidationSources,
) *ConsolidationService {
	return &ConsolidationService{
		db:             db,
		difficultyRepo: difficultyRepo,
		genreRepo:      genreRepo,
		courseRepo:     courseRepo,
		pwPepper:       pwPepper,
		datasources:    datasources,
		opts:           opts,
		sources:        sources,
	}
}

// ConsolidateAll は渡されたデータソースをすべて統合します。
func (s *ConsolidationService) ConsolidateAll(ctx context.Context) error {
	workspace, err := s.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return nil
	}
	defer workspace.Close()
	return s.syncWorkspace(ctx, workspace, s.db, true)
}

// ConsolidateBySource は指定されたデータソースのみを統合します。
func (s *ConsolidationService) ConsolidateBySource(ctx context.Context, sourceType string) error {
	var target []string
	for _, sourceName := range s.datasources {
		if sourceName == sourceType {
			target = append(target, sourceName)
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
	return s.syncWorkspace(ctx, workspace, s.db, sourceType == "additional_songs")
}

// BuildWorkspace は対象データソースを SQLite ワークスペースに取り込みます。
func (s *ConsolidationService) BuildWorkspace(ctx context.Context) (*songchart.SongChartWorkspace, error) {
	return s.buildWorkspace(ctx, s.datasources)
}

// SyncWorkspace は準備済みワークスペースを MySQL に同期します。
func (s *ConsolidationService) SyncWorkspace(ctx context.Context, workspace *songchart.SongChartWorkspace, mysql domainrepo.DBExecutor) error {
	return s.syncWorkspace(ctx, workspace, mysql, true)
}

func (s *ConsolidationService) syncWorkspace(ctx context.Context, workspace *songchart.SongChartWorkspace, mysql domainrepo.DBExecutor, syncCourses bool) error {
	if workspace == nil {
		return nil
	}
	syncOpts := songchart.SyncOptions{
		MajorUpdate:            s.opts.MajorUpdate,
		FillMissingReleaseDate: s.opts.FillMissingReleaseDate,
	}

	if err := workspace.SyncToMySQL(ctx, mysql, syncOpts); err != nil {
		return fmt.Errorf("failed to sync workspace to MySQL: %w", err)
	}
	if syncCourses {
		if err := s.syncCourses(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsolidationService) syncCourses(ctx context.Context) error {
	if s.sources.AdditionalSongs == nil || len(s.sources.AdditionalSongs.Courses) == 0 {
		return nil
	}

	courses := make([]entity.Course, 0, len(s.sources.AdditionalSongs.Courses))
	for _, course := range s.sources.AdditionalSongs.Courses {
		courses = append(courses, entity.Course{
			OfficialIdx: strings.TrimSpace(course.ID),
			Name:        strings.TrimSpace(course.Title),
			ClassName:   strings.TrimSpace(course.Class),
		})
	}
	if err := s.courseRepo.SaveAll(ctx, courses); err != nil {
		return fmt.Errorf("failed to sync courses to MySQL: %w", err)
	}
	slog.Info("Synchronized courses to MySQL", "count", len(courses))
	return nil
}

func (s *ConsolidationService) buildWorkspace(ctx context.Context, datasources []string) (*songchart.SongChartWorkspace, error) {
	target := make([]string, 0, len(datasources))
	for _, sourceName := range datasources {
		if s.opts.MajorUpdate {
			if sourceName != "official" && sourceName != "additional_songs" {
				slog.Info("Skipping datasource in major update mode", "type", sourceName)
				continue
			}
		}
		target = append(target, sourceName)
	}

	if len(target) == 0 {
		slog.Warn("No datasources to consolidate")
		return nil, nil
	}

	workspace, err := songchart.NewSongChartWorkspace(ctx, songchart.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create song chart workspace: %w", err)
	}

	for _, sourceName := range target {
		if err := s.consolidateSource(ctx, workspace, sourceName); err != nil {
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
