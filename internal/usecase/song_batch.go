package usecase

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

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
	cacheDir      string
}

type sourceInput struct {
	Type     string
	Path     string
	Promote  bool
	CacheHit bool
}

// NewSongBatchUsecase は SongBatchUsecase を生成します。
func NewSongBatchUsecase(
	resolver DatasourceResolver,
	newDownloader DownloaderFactory,
	sourceImporter SourceImporter,
	consolidator Consolidator,
	cacheDir string,
) *SongBatchUsecase {
	if cacheDir == "" {
		cacheDir = info.DatasourceCacheDir
	}
	return &SongBatchUsecase{
		resolver:      resolver,
		newDownloader: newDownloader,
		importer:      sourceImporter,
		consolidator:  consolidator,
		cacheDir:      cacheDir,
	}
}

// Execute はモードに応じてデータソースを取得し、必須条件を満たせば同期します。
func (u *SongBatchUsecase) Execute(ctx context.Context, req RunRequest) error {
	targets := targetDatasourceTypes(req.Mode)

	var (
		inputs  []sourceInput
		tempDir string
		err     error
	)
	if req.SkipDownload {
		inputs, err = u.inputsFromCache(req.Mode, targets)
	} else {
		dir, mkdirErr := os.MkdirTemp("", info.TempDirPrefix)
		if mkdirErr != nil {
			return fmt.Errorf("failed to create temp dir: %w", mkdirErr)
		}
		tempDir = dir
		slog.Info("using execution temp dir", "path", tempDir)
		defer func() {
			if removeErr := os.RemoveAll(tempDir); removeErr != nil {
				slog.Warn("failed to remove temp dir", "path", tempDir, "error", removeErr)
			}
		}()
		inputs, err = u.inputsFromDownload(ctx, req.Mode, targets, tempDir)
	}
	if err != nil {
		return err
	}

	sources, names, promote, err := u.importInputs(req.Mode, inputs)
	if err != nil {
		return err
	}

	opts := service.ConsolidationOptions{
		MajorUpdate:            req.Mode == RunModeMajorUpdate,
		FillMissingReleaseDate: req.FillMissingReleaseDate,
	}
	if err := u.consolidator.Consolidate(ctx, sources, names, opts); err != nil {
		return err
	}

	u.promote(promote)
	return nil
}

func (u *SongBatchUsecase) inputsFromCache(mode RunMode, targets []string) ([]sourceInput, error) {
	inputs := make([]sourceInput, 0, len(targets))
	for _, name := range targets {
		path := u.cacheFile(name)
		if !fileExists(path) {
			if isRequired(mode, name) {
				return nil, fmt.Errorf("required datasource %s file not found: %s", name, path)
			}
			slog.Warn("skipping complementary datasource; cache file not found", "type", name)
			continue
		}
		inputs = append(inputs, sourceInput{Type: name, Path: path, CacheHit: true})
	}
	return inputs, nil
}

func (u *SongBatchUsecase) inputsFromDownload(ctx context.Context, mode RunMode, targets []string, tempDir string) ([]sourceInput, error) {
	resolved := make(map[string]DatasourceRef, len(targets))
	toDownload := make([]DatasourceRef, 0, len(targets))

	for _, name := range targets {
		ds, err := u.resolver.Resolve(name)
		if err != nil {
			if isRequired(mode, name) {
				return nil, fmt.Errorf("required datasource %s could not be resolved: %w", name, err)
			}
			continue
		}
		if ds.Type == "" {
			ds.Type = name
		}
		resolved[name] = ds
		toDownload = append(toDownload, ds)
	}

	resultsByType := map[string]DownloadResult{}
	if len(toDownload) > 0 {
		results, err := u.newDownloader(tempDir).DownloadAll(ctx, toDownload)
		if err != nil {
			return nil, fmt.Errorf("datasource download failed: %w", err)
		}
		for _, result := range results {
			resultsByType[result.Type] = result
		}
	}

	inputs := make([]sourceInput, 0, len(targets))
	for _, name := range targets {
		ds, resolvedOK := resolved[name]
		if resolvedOK {
			result, ok := resultsByType[ds.Type]
			if ok && result.Success {
				inputs = append(inputs, sourceInput{Type: ds.Type, Path: result.Path, Promote: true})
				continue
			}
			if isRequired(mode, name) {
				errMsg := "download failed"
				if ok && result.Error != "" {
					errMsg = result.Error
				}
				return nil, fmt.Errorf("required datasource %s download failed: %s", name, errMsg)
			}
			if in, ok := u.lastKnownGoodInput(name, "download failed"); ok {
				inputs = append(inputs, in)
			}
			continue
		}
		if in, ok := u.lastKnownGoodInput(name, "resolve failed"); ok {
			inputs = append(inputs, in)
		}
	}

	return inputs, nil
}

func (u *SongBatchUsecase) importInputs(mode RunMode, inputs []sourceInput) (service.ConsolidationSources, []string, []sourceInput, error) {
	var sources service.ConsolidationSources
	names := make([]string, 0, len(inputs))
	promote := make([]sourceInput, 0, len(inputs))

	for _, in := range inputs {
		imported, promoteThis, err := u.importOne(in)
		if err != nil {
			if isRequired(mode, in.Type) {
				return service.ConsolidationSources{}, nil, nil, err
			}
			slog.Warn("skipping complementary datasource due to import failure", "type", in.Type, "error", err)
			continue
		}
		if err := assignSource(&sources, in.Type, imported.Data); err != nil {
			if isRequired(mode, in.Type) {
				return service.ConsolidationSources{}, nil, nil, err
			}
			slog.Warn("skipping complementary datasource due to unexpected type", "type", in.Type, "error", err)
			continue
		}
		names = append(names, in.Type)
		if promoteThis {
			promote = append(promote, in)
		}
	}

	for _, required := range requiredDatasourceTypes(mode) {
		if !slices.Contains(names, required) {
			return service.ConsolidationSources{}, nil, nil, fmt.Errorf("required datasource %s is missing after import", required)
		}
	}

	return sources, names, promote, nil
}

func (u *SongBatchUsecase) importOne(in sourceInput) (*importer.ImportResult, bool, error) {
	result, err := u.importer.Import(in.Type, in.Path)
	if isImported(result, err) {
		return result, in.Promote, nil
	}

	if allowsLastKnownGood(in.Type) && !in.CacheHit {
		cachePath := u.cacheFile(in.Type)
		if fileExists(cachePath) && cachePath != in.Path {
			u.logLastKnownGood(in.Type, cachePath, "import failed")
			cached, cacheErr := u.importer.Import(in.Type, cachePath)
			if isImported(cached, cacheErr) {
				return cached, false, nil
			}
		}
	}

	if err != nil {
		return nil, false, fmt.Errorf("datasource %s import failed: %w", in.Type, err)
	}
	return nil, false, fmt.Errorf("datasource %s import failed: no data", in.Type)
}

func isImported(result *importer.ImportResult, err error) bool {
	return err == nil && result != nil && result.Data != nil
}

func (u *SongBatchUsecase) lastKnownGoodInput(name, reason string) (sourceInput, bool) {
	if !allowsLastKnownGood(name) {
		return sourceInput{}, false
	}
	path := u.cacheFile(name)
	if !fileExists(path) {
		slog.Warn("complementary datasource unavailable and no last-known-good cache", "type", name)
		return sourceInput{}, false
	}
	u.logLastKnownGood(name, path, reason)
	return sourceInput{Type: name, Path: path, CacheHit: true}, true
}

func (u *SongBatchUsecase) logLastKnownGood(name, path, reason string) {
	attrs := []any{"type", name, "path", path, "reason", reason}
	if fi, err := os.Stat(path); err == nil {
		attrs = append(attrs, "modified_at", fi.ModTime().Format(time.RFC3339))
	}
	slog.Warn("using last-known-good datasource", attrs...)
}

func (u *SongBatchUsecase) promote(inputs []sourceInput) {
	if len(inputs) == 0 {
		return
	}
	if err := os.MkdirAll(u.cacheDir, 0755); err != nil {
		slog.Warn("failed to create cache dir", "path", u.cacheDir, "error", err)
		return
	}
	for _, in := range inputs {
		dst := u.cacheFile(in.Type)
		if err := copyFile(in.Path, dst); err != nil {
			slog.Warn("failed to promote datasource cache", "type", in.Type, "error", err)
		}
	}
}

func (u *SongBatchUsecase) cacheFile(sourceType string) string {
	return filepath.Join(u.cacheDir, sourceType+".json")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+"-*")
	if err != nil {
		return err
	}
	tempPath := out.Name()
	defer func() {
		_ = out.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Chmod(0644); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, dst)
}
