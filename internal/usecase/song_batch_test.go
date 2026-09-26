package usecase

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"
)

type stubResolver struct {
	sources map[string]DatasourceRef
	errs    map[string]error
	called  []string
}

func (s *stubResolver) Resolve(name string) (DatasourceRef, error) {
	s.called = append(s.called, name)
	if err, ok := s.errs[name]; ok {
		return DatasourceRef{}, err
	}
	if ds, ok := s.sources[name]; ok {
		return ds, nil
	}
	return DatasourceRef{}, errors.New("unresolved")
}

type stubDownloader struct {
	outputDir string
	fn        func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error)
}

func (s *stubDownloader) DownloadAll(_ context.Context, datasources []DatasourceRef) ([]DownloadResult, error) {
	return s.fn(s.outputDir, datasources)
}

type stubImporter struct {
	fn func(sourceType, filePath string) (*importer.ImportResult, error)
}

func (s *stubImporter) Import(sourceType, filePath string) (*importer.ImportResult, error) {
	return s.fn(sourceType, filePath)
}

type stubConsolidator struct {
	called  bool
	sources service.ConsolidationSources
	names   []string
	opts    service.ConsolidationOptions
	err     error
}

func (s *stubConsolidator) Consolidate(_ context.Context, sources service.ConsolidationSources, names []string, opts service.ConsolidationOptions) error {
	s.called = true
	s.sources = sources
	s.names = names
	s.opts = opts
	return s.err
}

func successfulImport(sourceType string) *importer.ImportResult {
	switch importer.DataSourceType(sourceType) {
	case importer.DataSourceOfficial:
		data := importer.OfficialData{}
		return &importer.ImportResult{Type: importer.DataSourceOfficial, Data: &data}
	case importer.DataSourceAdditionalSongs:
		return &importer.ImportResult{Type: importer.DataSourceAdditionalSongs, Data: &importer.AdditionalSongsData{}}
	case importer.DataSourceMainframe:
		data := importer.MainframeData{}
		return &importer.ImportResult{Type: importer.DataSourceMainframe, Data: &data}
	case importer.DataSourceSt1027:
		return &importer.ImportResult{Type: importer.DataSourceSt1027, Data: &importer.St1027Data{}}
	case importer.DataSourceOtogeDb:
		data := importer.OtogeDbData{}
		return &importer.ImportResult{Type: importer.DataSourceOtogeDb, Data: &data}
	default:
		return &importer.ImportResult{Type: importer.DataSourceType(sourceType)}
	}
}

func allResolved() map[string]DatasourceRef {
	names := []string{"official", "additional_songs", "st1027", "mainframe", "otoge_db"}
	sources := make(map[string]DatasourceRef, len(names))
	for _, name := range names {
		sources[name] = DatasourceRef{Type: name, URL: "https://example.invalid/" + name}
	}
	return sources
}

func successfulDownloads(t *testing.T, outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
	t.Helper()
	results := make([]DownloadResult, 0, len(datasources))
	for _, ds := range datasources {
		path := filepath.Join(outputDir, ds.Type+".json")
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("write download: %v", err)
		}
		results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
	}
	return results, nil
}

func newUsecase(
	resolver *stubResolver,
	downloadFn func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error),
	importFn func(sourceType, filePath string) (*importer.ImportResult, error),
	consolidator *stubConsolidator,
	tempDir *string,
) *SongBatchUsecase {
	return NewSongBatchUsecase(
		resolver,
		func(outputDir string) Downloader {
			*tempDir = outputDir
			return &stubDownloader{outputDir: outputDir, fn: downloadFn}
		},
		&stubImporter{fn: importFn},
		consolidator,
	)
}

func TestRunRequestLockConflictIsError(t *testing.T) {
	t.Parallel()

	if NewRunRequest(false, false).LockConflictIsError() {
		t.Fatal("normal run should skip on lock conflict")
	}
	if NewRunRequest(false, true).LockConflictIsError() {
		t.Fatal("fill-missing-release-date should skip on lock conflict")
	}
	if !NewRunRequest(true, false).LockConflictIsError() {
		t.Fatal("major update should error on lock conflict")
	}
}

func TestExecuteRequiredDatasourceFailuresDoNotConsolidate(t *testing.T) {
	tests := []struct {
		name       string
		resolver   *stubResolver
		downloadFn func(string, []DatasourceRef) ([]DownloadResult, error)
		importFn   func(string, string) (*importer.ImportResult, error)
		wantStage  string
	}{
		{
			name:     "resolve",
			resolver: &stubResolver{sources: allResolved(), errs: map[string]error{"official": errors.New("missing env")}},
			downloadFn: func(string, []DatasourceRef) ([]DownloadResult, error) {
				t.Fatal("download must not run")
				return nil, nil
			},
			importFn:  func(string, string) (*importer.ImportResult, error) { return nil, nil },
			wantStage: "resolve",
		},
		{
			name:     "download",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
				results, _ := successfulDownloads(t, outputDir, datasources)
				for i := range results {
					if results[i].Type == "official" {
						results[i] = DownloadResult{Type: "official", Error: "http 500"}
					}
				}
				return results, nil
			},
			importFn:  func(sourceType, _ string) (*importer.ImportResult, error) { return successfulImport(sourceType), nil },
			wantStage: "download",
		},
		{
			name:     "parse",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
				return successfulDownloads(t, outputDir, datasources)
			},
			importFn: func(sourceType, _ string) (*importer.ImportResult, error) {
				if sourceType == "official" {
					return nil, errors.New("invalid JSON")
				}
				return successfulImport(sourceType), nil
			},
			wantStage: "parse",
		},
		{
			name:     "validation",
			resolver: &stubResolver{sources: allResolved()},
			downloadFn: func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
				return successfulDownloads(t, outputDir, datasources)
			},
			importFn: func(sourceType, _ string) (*importer.ImportResult, error) {
				if sourceType == "official" {
					return nil, fmtValidationError("official is empty")
				}
				return successfulImport(sourceType), nil
			},
			wantStage: "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consolidator := &stubConsolidator{}
			var tempDir string
			uc := newUsecase(tt.resolver, tt.downloadFn, tt.importFn, consolidator, &tempDir)

			err := uc.Execute(context.Background(), NewRunRequest(false, false))
			if err == nil || !strings.Contains(err.Error(), "official") || !strings.Contains(err.Error(), tt.wantStage+" stage") {
				t.Fatalf("unexpected error: %v", err)
			}
			if consolidator.called {
				t.Fatal("consolidator must not run")
			}
			if tempDir != "" {
				if _, statErr := os.Stat(tempDir); !os.IsNotExist(statErr) {
					t.Fatalf("temp dir should be removed: %v", statErr)
				}
			}
		})
	}
}

func fmtValidationError(message string) error {
	return errors.Join(importer.ErrValidation, errors.New(message))
}

func TestExecuteExcludesUnavailableComplementarySourcesAndLogsWarningResult(t *testing.T) {
	resolver := &stubResolver{sources: allResolved(), errs: map[string]error{"st1027": errors.New("missing env")}}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
			results, _ := successfulDownloads(t, outputDir, datasources)
			for i := range results {
				if results[i].Type == "otoge_db" {
					results[i] = DownloadResult{Type: "otoge_db", Error: "timeout"}
				}
			}
			return results, nil
		},
		func(sourceType, _ string) (*importer.ImportResult, error) { return successfulImport(sourceType), nil },
		consolidator,
		&tempDir,
	)

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	if err := uc.Execute(context.Background(), NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !consolidator.called {
		t.Fatal("valid sources must still be consolidated")
	}
	wantNames := []string{"official", "additional_songs", "mainframe"}
	if strings.Join(consolidator.names, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("names=%v, want=%v", consolidator.names, wantNames)
	}
	output := logs.String()
	for _, want := range []string{"source=st1027 stage=resolve", "source=otoge_db stage=download", "Completed with Warnings", "warning_count=2"} {
		if !strings.Contains(output, want) {
			t.Fatalf("log does not contain %q:\n%s", want, output)
		}
	}
}

func TestExecuteExcludesComplementarySourceOnValidationFailure(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType, _ string) (*importer.ImportResult, error) {
			if sourceType == "st1027" {
				return nil, fmtValidationError("missing songs")
			}
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	if err := uc.Execute(context.Background(), NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if consolidator.sources.St1027 != nil {
		t.Fatal("invalid complementary source must be excluded")
	}
	if consolidator.sources.OtogeDb == nil {
		t.Fatal("other valid complementary source must be retained")
	}
}

func TestExecuteLogsSuccessWithoutWarnings(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType, _ string) (*importer.ImportResult, error) { return successfulImport(sourceType), nil },
		consolidator,
		&tempDir,
	)

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	if err := uc.Execute(context.Background(), NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	output := logs.String()
	if !strings.Contains(output, "Completed Successfully") || strings.Contains(output, "Completed with Warnings") {
		t.Fatalf("unexpected completion log:\n%s", output)
	}
}

func TestExecuteDoesNotReadExistingDatasourceDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".datasources", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".datasources", "official.json"), []byte(`stale`), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	var importedPaths []string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType, filePath string) (*importer.ImportResult, error) {
			importedPaths = append(importedPaths, filePath)
			return successfulImport(sourceType), nil
		},
		consolidator,
		&tempDir,
	)

	if err := uc.Execute(context.Background(), NewRunRequest(false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, path := range importedPaths {
		if !strings.HasPrefix(path, tempDir+string(os.PathSeparator)) {
			t.Fatalf("imported non-execution file: %s", path)
		}
	}
}

func TestExecuteMajorUpdateUsesOnlyRequiredSources(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var tempDir string
	uc := newUsecase(
		resolver,
		func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
			if len(datasources) != 2 || datasources[0].Type != "official" || datasources[1].Type != "additional_songs" {
				t.Fatalf("unexpected major-update sources: %v", datasources)
			}
			return successfulDownloads(t, outputDir, datasources)
		},
		func(sourceType, _ string) (*importer.ImportResult, error) { return successfulImport(sourceType), nil },
		consolidator,
		&tempDir,
	)

	if err := uc.Execute(context.Background(), NewRunRequest(true, true)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !consolidator.opts.MajorUpdate || !consolidator.opts.FillMissingReleaseDate {
		t.Fatalf("options=%+v", consolidator.opts)
	}
	if len(resolver.called) != 2 {
		t.Fatalf("resolved sources=%v", resolver.called)
	}
}

func TestTargetAndRequiredDatasourceTypes(t *testing.T) {
	t.Parallel()

	if got := len(targetDatasourceTypes(RunModeMajorUpdate)); got != 2 {
		t.Fatalf("major targets=%d", got)
	}
	if got := len(requiredDatasourceTypes(RunModeMajorUpdate)); got != 2 {
		t.Fatalf("major required=%d", got)
	}
	if got := len(requiredDatasourceTypes(RunModeNormal)); got != 3 {
		t.Fatalf("normal required=%d", got)
	}
}
