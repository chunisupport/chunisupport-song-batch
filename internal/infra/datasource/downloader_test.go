package datasource

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloader_DownloadAll_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			fmt.Fprint(w, `{"status":"ok"}`)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	datasources := []Datasource{
		{Type: "success", URL: server.URL + "/success"},
		{Type: "fail", URL: server.URL + "/fail"},
	}

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), datasources)
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	byType := map[string]DownloadResult{}
	for _, result := range results {
		byType[result.Type] = result
	}

	success := byType["success"]
	if !success.Success {
		t.Errorf("expected success result to be successful, error=%s", success.Error)
	}
	if success.Path == "" {
		t.Error("expected successful download to have a path")
	}
	if success.Bytes == 0 {
		t.Error("expected successful download to report bytes")
	}
	if success.FetchedAt.IsZero() {
		t.Error("expected successful download to report fetched time")
	}

	fail := byType["fail"]
	if fail.Success {
		t.Error("expected fail result to be unsuccessful")
	}
	if fail.Error == "" {
		t.Error("expected fail result to have an error")
	}

	successPath := filepath.Join(outputDir, "success.json")
	if _, err := os.Stat(successPath); os.IsNotExist(err) {
		t.Errorf("Expected successful download file to exist")
	}

	failPath := filepath.Join(outputDir, "fail.json")
	if _, err := os.Stat(failPath); err == nil {
		t.Errorf("Expected failed download file to not exist")
	}
}

func TestDownloader_DownloadAll_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader := NewDownloader(t.TempDir())
	results, err := downloader.DownloadAll(ctx, []Datasource{
		{Type: "official", URL: "https://example.invalid"},
	})
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("results = %v, want one failed result", results)
	}
	if !strings.Contains(results[0].Error, context.Canceled.Error()) {
		t.Fatalf("result error = %q, want context cancellation", results[0].Error)
	}
}

func TestDownloader_DownloadAll_RetriesTransientThenSucceeds(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = original })

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-retry-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), []Datasource{
		{Type: "official", URL: server.URL},
	})
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("expected retry success, results=%v", results)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestDownloader_DownloadAll_AllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	datasources := []Datasource{
		{Type: "fail1", URL: server.URL + "/fail1"},
		{Type: "fail2", URL: server.URL + "/fail2"},
	}

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), datasources)
	if err != nil {
		t.Fatalf("DownloadAll should not return overall error on per-source failure, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Success {
			t.Errorf("expected %s to fail", result.Type)
		}
		if result.Error == "" {
			t.Errorf("expected %s to have an error", result.Type)
		}
		path := filepath.Join(outputDir, result.Type+".json")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected failed download file not to exist: %s", path)
		}
	}
}
