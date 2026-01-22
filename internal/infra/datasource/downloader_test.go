package datasource

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloader_DownloadAll_ErrorHandling(t *testing.T) {
	// 1つが失敗し、もう1つが成功するケース
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
		{Type: "success", URL: server.URL + "/success", Active: true},
		{Type: "fail", URL: server.URL + "/fail", Active: true},
		{Type: "inactive", URL: server.URL + "/inactive", Active: false},
	}

	downloader := NewDownloader(outputDir)
	err = downloader.DownloadAll(datasources)

	// 戻り値のエラーはnilであること（一部成功しているため）
	if err != nil {
		t.Errorf("Expected nil error when at least one download succeeds, but got: %v", err)
	}

	// 成功したファイルが存在することを確認
	successPath := filepath.Join(outputDir, "success.json")
	if _, err := os.Stat(successPath); os.IsNotExist(err) {
		t.Errorf("Expected successful download file to exist")
	}

	// 失敗したファイルが存在しないことを確認
	failPath := filepath.Join(outputDir, "fail.json")
	if _, err := os.Stat(failPath); err == nil {
		t.Errorf("Expected failed download file to not exist")
	}
}

func TestDownloader_DownloadAll_AllFail(t *testing.T) {
	// すべてのダウンロードが失敗するケース
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
		{Type: "fail1", URL: server.URL + "/fail1", Active: true},
		{Type: "fail2", URL: server.URL + "/fail2", Active: true},
	}

	downloader := NewDownloader(outputDir)
	err = downloader.DownloadAll(datasources)

	// エラーが返されることを確認
	if err == nil {
		t.Errorf("Expected an error when all downloads fail, but got nil")
	}
}
