package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"
)

// SongBatchUsecase は楽曲バッチの取得・検証・統合を実行します。
type SongBatchUsecase struct {
	resolver      DatasourceResolver
	newDownloader DownloaderFactory
	importer      SourceImporter
	consolidator  Consolidator
}

type sourceInput struct {
	Type string
	Path string
}

// NewSongBatchUsecase は SongBatchUsecase を生成します。
func NewSongBatchUsecase(
	resolver DatasourceResolver,
	newDownloader DownloaderFactory,
	sourceImporter SourceImporter,
	consolidator Consolidator,
) *SongBatchUsecase {
	return &SongBatchUsecase{
		resolver:      resolver,
		newDownloader: newDownloader,
		importer:      sourceImporter,
		consolidator:  consolidator,
	}
}

// Execute はモードに応じてデータソースを取得し、必須条件を満たせば同期します。
func (u *SongBatchUsecase) Execute(ctx context.Context, req RunRequest) error {
	tempDir, err := os.MkdirTemp("", info.TempDirPrefix)
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	slog.Info("using execution temp dir", "path", tempDir)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			slog.Warn("failed to remove temp dir", "path", tempDir, "error", removeErr)
		}
	}()

	inputs, warningCount, err := u.inputsFromDownload(ctx, req.Mode, targetDatasourceTypes(req.Mode), tempDir)
	if err != nil {
		return err
	}

	sources, names, importWarnings, err := u.importInputs(req.Mode, inputs)
	if err != nil {
		return err
	}
	warningCount += importWarnings

	opts := service.ConsolidationOptions{
		MajorUpdate:            req.Mode == RunModeMajorUpdate,
		FillMissingReleaseDate: req.FillMissingReleaseDate,
	}
	if err := u.consolidator.Consolidate(ctx, sources, names, opts); err != nil {
		return err
	}

	if warningCount > 0 {
		slog.Warn("Data Import Batch Completed with Warnings", "warning_count", warningCount)
	} else {
		slog.Info("Data Import Batch Completed Successfully")
	}
	return nil
}

func (u *SongBatchUsecase) inputsFromDownload(ctx context.Context, mode RunMode, targets []string, tempDir string) ([]sourceInput, int, error) {
	resolved := make(map[string]DatasourceRef, len(targets))
	toDownload := make([]DatasourceRef, 0, len(targets))
	warningCount := 0

	for _, name := range targets {
		ds, err := u.resolver.Resolve(name)
		if err != nil {
			if isRequired(mode, name) {
				return nil, warningCount, fmt.Errorf("required datasource %s failed at resolve stage: %w", name, err)
			}
			logComplementaryFailure(name, "resolve", err.Error())
			warningCount++
			continue
		}
		if ds.Type == "" {
			ds.Type = name
		}
		resolved[name] = ds
		toDownload = append(toDownload, ds)
	}

	results, err := u.newDownloader(tempDir).DownloadAll(ctx, toDownload)
	if err != nil {
		return nil, warningCount, fmt.Errorf("datasource download failed: %w", err)
	}
	resultsByType := make(map[string]DownloadResult, len(results))
	for _, result := range results {
		resultsByType[result.Type] = result
	}

	inputs := make([]sourceInput, 0, len(targets))
	for _, name := range targets {
		ds, ok := resolved[name]
		if !ok {
			continue
		}
		result, ok := resultsByType[ds.Type]
		if ok && result.Success {
			inputs = append(inputs, sourceInput{Type: ds.Type, Path: result.Path})
			continue
		}

		reason := "download result was not returned"
		if ok && result.Error != "" {
			reason = result.Error
		}
		if isRequired(mode, name) {
			return nil, warningCount, fmt.Errorf("required datasource %s failed at download stage: %s", name, reason)
		}
		logComplementaryFailure(name, "download", reason)
		warningCount++
	}

	return inputs, warningCount, nil
}

func (u *SongBatchUsecase) importInputs(mode RunMode, inputs []sourceInput) (service.ConsolidationSources, []string, int, error) {
	var sources service.ConsolidationSources
	names := make([]string, 0, len(inputs))
	warningCount := 0

	for _, in := range inputs {
		result, err := u.importer.Import(in.Type, in.Path)
		if err == nil && (result == nil || result.Data == nil) {
			err = fmt.Errorf("no data")
		}
		if err != nil {
			stage := importFailureStage(err)
			if isRequired(mode, in.Type) {
				return service.ConsolidationSources{}, nil, warningCount, fmt.Errorf("required datasource %s failed at %s stage: %w", in.Type, stage, err)
			}
			logComplementaryFailure(in.Type, stage, err.Error())
			warningCount++
			continue
		}
		if err := assignSource(&sources, in.Type, result.Data); err != nil {
			if isRequired(mode, in.Type) {
				return service.ConsolidationSources{}, nil, warningCount, fmt.Errorf("required datasource %s failed at validation stage: %w", in.Type, err)
			}
			logComplementaryFailure(in.Type, "validation", err.Error())
			warningCount++
			continue
		}
		names = append(names, in.Type)
	}

	for _, required := range requiredDatasourceTypes(mode) {
		if !slices.Contains(names, required) {
			return service.ConsolidationSources{}, nil, warningCount, fmt.Errorf("required datasource %s is missing after validation", required)
		}
	}

	return sources, names, warningCount, nil
}

func importFailureStage(err error) string {
	if errors.Is(err, importer.ErrValidation) {
		return "validation"
	}
	return "parse"
}

func logComplementaryFailure(source, stage, reason string) {
	slog.Warn("excluding complementary datasource", "source", source, "stage", stage, "reason", reason)
}
