package usecase

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func officialData() *importer.OfficialData {
	data := importer.OfficialData{}
	return &data
}

func additionalData() *importer.AdditionalSongsData {
	return &importer.AdditionalSongsData{}
}

func mainframeData() *importer.MainframeData {
	data := importer.MainframeData{}
	return &data
}

func st1027Data() *importer.St1027Data {
	return &importer.St1027Data{}
}

func otogeData() *importer.OtogeDbData {
	data := importer.OtogeDbData{}
	return &data
}

func successfulImport(sourceType string) *importer.ImportResult {
	switch importer.DataSourceType(sourceType) {
	case importer.DataSourceOfficial:
		return &importer.ImportResult{Type: importer.DataSourceOfficial, Data: officialData()}
	case importer.DataSourceAdditionalSongs:
		return &importer.ImportResult{Type: importer.DataSourceAdditionalSongs, Data: additionalData()}
	case importer.DataSourceMainframe:
		return &importer.ImportResult{Type: importer.DataSourceMainframe, Data: mainframeData()}
	case importer.DataSourceSt1027:
		return &importer.ImportResult{Type: importer.DataSourceSt1027, Data: st1027Data()}
	case importer.DataSourceOtogeDb:
		return &importer.ImportResult{Type: importer.DataSourceOtogeDb, Data: otogeData()}
	default:
		return &importer.ImportResult{Type: importer.DataSourceType(sourceType), Data: nil}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestCopyFileReplacesDestinationWithoutLeavingTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.json")
	dst := filepath.Join(dir, "cache.json")
	writeFile(t, src, `{"version":"new"}`)
	writeFile(t, dst, `{"version":"old"}`)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(data) != `{"version":"new"}` {
		t.Fatalf("destination = %s", data)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".cache.json-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
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

func newUsecase(
	t *testing.T,
	resolver *stubResolver,
	downloadFn func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error),
	importFn func(sourceType, filePath string) (*importer.ImportResult, error),
	consolidator *stubConsolidator,
) (*SongBatchUsecase, *string) {
	t.Helper()
	cacheDir := t.TempDir()
	var tempDir string
	uc := NewSongBatchUsecase(
		resolver,
		func(outputDir string) Downloader {
			tempDir = outputDir
			return &stubDownloader{outputDir: outputDir, fn: downloadFn}
		},
		&stubImporter{fn: importFn},
		consolidator,
		cacheDir,
	)
	return uc, &tempDir
}

func TestRunRequest_LockConflictIsError(t *testing.T) {
	t.Parallel()

	if NewRunRequest(false, false, false).LockConflictIsError() {
		t.Fatal("normal run should skip on lock conflict")
	}
	if NewRunRequest(false, false, true).LockConflictIsError() {
		t.Fatal("fill-missing-release-date should skip on lock conflict")
	}
	if !NewRunRequest(true, false, false).LockConflictIsError() {
		t.Fatal("major update should error on lock conflict")
	}
	if !NewRunRequest(false, true, false).LockConflictIsError() {
		t.Fatal("skip-download should error on lock conflict")
	}
}

func TestExecute_DoesNotReuseStaleOfficialOnDownloadFailure(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			if ds.Type == "official" {
				results = append(results, DownloadResult{Type: ds.Type, Success: false, Error: "http 500"})
				continue
			}
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "official.json"), `[{"id":"old"}]`)

	err := uc.Execute(context.Background(), NewRunRequest(false, false, false))
	if err == nil {
		t.Fatal("expected required official failure")
	}
	if consolidator.called {
		t.Fatal("consolidator must not run when official download fails")
	}
}

func TestExecute_UsesLastKnownGoodForComplementary(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	var importedPaths []string
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			if ds.Type == "st1027" || ds.Type == "otoge_db" {
				results = append(results, DownloadResult{Type: ds.Type, Success: false, Error: "timeout"})
				continue
			}
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		importedPaths = append(importedPaths, sourceType+":"+filePath)
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "st1027.json"), `{"songs":[]}`)
	writeFile(t, filepath.Join(uc.cacheDir, "otoge_db.json"), `[]`)

	if err := uc.Execute(context.Background(), NewRunRequest(false, false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !consolidator.called {
		t.Fatal("expected consolidator to run")
	}
	if consolidator.sources.St1027 == nil || consolidator.sources.OtogeDb == nil {
		t.Fatal("expected complementary sources from last-known-good")
	}
	wantNames := []string{"official", "additional_songs", "st1027", "mainframe", "otoge_db"}
	if len(consolidator.names) != len(wantNames) {
		t.Fatalf("names=%v", consolidator.names)
	}
	for i, name := range wantNames {
		if consolidator.names[i] != name {
			t.Fatalf("consolidation order: got %v, want %v", consolidator.names, wantNames)
		}
	}
	foundCacheSt1027 := false
	foundCacheOfficial := false
	for _, p := range importedPaths {
		switch p {
		case "st1027:" + filepath.Join(uc.cacheDir, "st1027.json"):
			foundCacheSt1027 = true
		case "official:" + filepath.Join(uc.cacheDir, "official.json"):
			foundCacheOfficial = true
		}
	}
	if !foundCacheSt1027 {
		t.Fatalf("expected st1027 to be imported from cache, got %v", importedPaths)
	}
	if foundCacheOfficial {
		t.Fatal("official must not be imported from cache when download succeeded")
	}
}

func TestExecute_RemovesTempDirAndDoesNotWriteCacheOnRequiredFailure(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, tempDir := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			if ds.Type == "mainframe" {
				results = append(results, DownloadResult{Type: ds.Type, Success: false, Error: "sheet missing"})
				continue
			}
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{"ok":true}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)

	err := uc.Execute(context.Background(), NewRunRequest(false, false, false))
	if err == nil {
		t.Fatal("expected mainframe failure")
	}
	if consolidator.called {
		t.Fatal("consolidator must not run")
	}
	if *tempDir == "" {
		t.Fatal("expected temp dir to be created")
	}
	if _, err := os.Stat(*tempDir); !os.IsNotExist(err) {
		t.Fatalf("temp dir should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(uc.cacheDir, "official.json")); err == nil {
		t.Fatal("failed run must not promote cache")
	}
}

func TestExecute_PromotesSuccessfulDownloadsAndRemovesTempDir(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, tempDir := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{"from":"temp"}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)

	if err := uc.Execute(context.Background(), NewRunRequest(false, false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if *tempDir == "" {
		t.Fatal("expected temp dir")
	}
	if _, err := os.Stat(*tempDir); !os.IsNotExist(err) {
		t.Fatalf("temp dir should be removed, stat err=%v", err)
	}
	promoted := filepath.Join(uc.cacheDir, "official.json")
	data, err := os.ReadFile(promoted)
	if err != nil {
		t.Fatalf("expected promoted cache: %v", err)
	}
	if string(data) != `{"from":"temp"}` {
		t.Fatalf("unexpected cache content: %s", data)
	}
}

func TestExecute_MajorUpdateResolvesOnlyRequiredSources(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		if len(datasources) != 2 {
			t.Fatalf("expected 2 datasources, got %d", len(datasources))
		}
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			if ds.Type != "official" && ds.Type != "additional_songs" {
				t.Fatalf("unexpected datasource in major update: %s", ds.Type)
			}
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)

	if err := uc.Execute(context.Background(), NewRunRequest(true, false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, name := range resolver.called {
		if name != "official" && name != "additional_songs" {
			t.Fatalf("major update must not resolve %s", name)
		}
	}
	if consolidator.sources.Mainframe != nil || consolidator.sources.St1027 != nil {
		t.Fatal("major update must not import extra sources")
	}
	if !consolidator.opts.MajorUpdate {
		t.Fatal("expected major update option")
	}
}

func TestExecute_MajorUpdateOfficialFailureDoesNotSync(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			if ds.Type == "official" {
				results = append(results, DownloadResult{Type: ds.Type, Success: false, Error: "timeout"})
				continue
			}
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
				t.Fatalf("write: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "official.json"), `[]`)

	if err := uc.Execute(context.Background(), NewRunRequest(true, false, false)); err == nil {
		t.Fatal("expected failure")
	}
	if consolidator.called {
		t.Fatal("must not sync")
	}
}

func TestExecute_ResolveFailureUsesLastKnownGoodWithoutReordering(t *testing.T) {
	resolver := &stubResolver{
		sources: allResolved(),
		errs:    map[string]error{"st1027": errors.New("missing env")},
	}
	delete(resolver.sources, "st1027")
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		for _, ds := range datasources {
			if ds.Type == "st1027" {
				t.Fatal("st1027 should not be downloaded when unresolved")
			}
		}
		results := make([]DownloadResult, 0, len(datasources))
		for _, ds := range datasources {
			path := filepath.Join(outputDir, ds.Type+".json")
			if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
				t.Fatalf("write download: %v", err)
			}
			results = append(results, DownloadResult{Type: ds.Type, Success: true, Path: path})
		}
		return results, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "st1027.json"), `{"songs":[]}`)

	if err := uc.Execute(context.Background(), NewRunRequest(false, false, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(consolidator.names) < 3 || consolidator.names[0] != "official" || consolidator.names[2] != "st1027" {
		t.Fatalf("expected official before st1027, got %v", consolidator.names)
	}
}

func TestExecute_SkipDownloadUsesCacheOnly(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	downloaderCalled := false
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		downloaderCalled = true
		return nil, errors.New("should not download")
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	for _, name := range []string{"official", "additional_songs", "mainframe"} {
		writeFile(t, filepath.Join(uc.cacheDir, name+".json"), `{}`)
	}

	if err := uc.Execute(context.Background(), NewRunRequest(false, true, true)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !consolidator.opts.FillMissingReleaseDate {
		t.Fatal("expected fill-missing-release-date option")
	}
	if downloaderCalled {
		t.Fatal("skip-download must not download")
	}
	if len(resolver.called) != 0 {
		t.Fatalf("skip-download must not resolve datasources, called=%v", resolver.called)
	}
	if !consolidator.called {
		t.Fatal("expected consolidator")
	}
}

func TestExecute_SkipDownloadMajorUpdateDoesNotRequireMainframe(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		t.Fatal("should not download")
		return nil, nil
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "official.json"), `{}`)
	writeFile(t, filepath.Join(uc.cacheDir, "additional_songs.json"), `{}`)

	if err := uc.Execute(context.Background(), NewRunRequest(true, true, false)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if consolidator.sources.Mainframe != nil {
		t.Fatal("major update skip-download must not require mainframe")
	}
}

func TestExecute_SkipDownloadMissingRequiredFails(t *testing.T) {
	resolver := &stubResolver{sources: allResolved()}
	consolidator := &stubConsolidator{}
	uc, _ := newUsecase(t, resolver, func(outputDir string, datasources []DatasourceRef) ([]DownloadResult, error) {
		return nil, errors.New("should not download")
	}, func(sourceType, filePath string) (*importer.ImportResult, error) {
		return successfulImport(sourceType), nil
	}, consolidator)
	writeFile(t, filepath.Join(uc.cacheDir, "official.json"), `{}`)

	if err := uc.Execute(context.Background(), NewRunRequest(false, true, false)); err == nil {
		t.Fatal("expected missing additional_songs/mainframe to fail")
	}
	if consolidator.called {
		t.Fatal("must not sync")
	}
}

func TestTargetAndRequiredDatasourceTypes(t *testing.T) {
	t.Parallel()

	majorTargets := targetDatasourceTypes(RunModeMajorUpdate)
	if len(majorTargets) != 2 {
		t.Fatalf("major targets: %v", majorTargets)
	}
	majorRequired := requiredDatasourceTypes(RunModeMajorUpdate)
	if len(majorRequired) != 2 {
		t.Fatalf("major required: %v", majorRequired)
	}
	normalRequired := requiredDatasourceTypes(RunModeNormal)
	if len(normalRequired) != 3 {
		t.Fatalf("normal required: %v", normalRequired)
	}
	if !allowsLastKnownGood("st1027") || !allowsLastKnownGood("otoge_db") {
		t.Fatal("complementary sources should allow last-known-good")
	}
	if allowsLastKnownGood("official") || allowsLastKnownGood("mainframe") {
		t.Fatal("required sources must not allow last-known-good")
	}
}
