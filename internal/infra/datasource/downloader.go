package datasource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Datasource はデータソースの定義を表します
type Datasource struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Params any    `json:"params"`
}

// Downloader はデータソースからファイルをダウンロードします
type Downloader struct {
	httpClient *http.Client
	outputDir  string
}

// NewDownloader は新しいDownloaderのインスタンスを生成します
func NewDownloader(outputDir string) *Downloader {
	return &Downloader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		outputDir: outputDir,
	}
}

// DownloadAll はすべてのデータソースをダウンロードします
func (d *Downloader) DownloadAll(datasources []Datasource) error {
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var g errgroup.Group
	g.SetLimit(8)
	var (
		mu             sync.Mutex
		downloadErrors []string
		successCount   int
		totalDownloads int
	)

	for _, ds := range datasources {
		totalDownloads++

		ds := ds // loop variable capture
		g.Go(func() error {
			slog.Info("Downloading datasource", "type", ds.Type, "url", ds.URL)

			var err error
			switch ds.Type {
			case "mainframe":
				err = d.downloadMainframe(ds)
			case "additional_songs":
				err = d.downloadAdditionalSongs(ds)
			default:
				err = d.downloadDatasource(ds)
			}

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				slog.Error("Failed to download datasource", "type", ds.Type, "error", err)
				downloadErrors = append(downloadErrors, fmt.Sprintf("%s: %v", ds.Type, err))
			} else {
				slog.Info("Successfully downloaded datasource", "type", ds.Type)
				successCount++
			}
			// errgroupが即座に終了しないよう、常にnilを返す
			return nil
		})
	}

	// すべてのgoroutineが終了するのを待つ
	// g.Go内の関数は常にnilを返すため、ここのエラーは常にnil
	_ = g.Wait()

	if successCount == 0 && len(downloadErrors) > 0 {
		return fmt.Errorf("all datasource downloads failed: %v", downloadErrors)
	}

	if len(downloadErrors) > 0 {
		slog.Warn("Some datasource downloads failed", "failed", downloadErrors, "succeeded", successCount, "total", totalDownloads)
	}

	return nil
}

// downloadDatasource は単一のデータソースをダウンロードします
func (d *Downloader) downloadDatasource(ds Datasource) error {
	finalURL, err := d.buildURL(ds.URL, ds.Params)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "chunisupport-api/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// fix_songs_charts データソースの場合、APIレスポンスのsuccessフラグをチェック
	if ds.Type == "fix_songs_charts" {
		var apiResponse struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(data, &apiResponse); err != nil {
			return fmt.Errorf("failed to parse fix_songs_charts response: %w", err)
		}
		if !apiResponse.Success {
			slog.Warn("fix_songs_charts API returned error, skipping file save", "type", ds.Type)
			return fmt.Errorf("fix_songs_charts API returned success=false")
		}
	}

	// JSONをminifyして不要な空白を削除
	originalSize := len(data)
	var minified bytes.Buffer
	if err := json.Compact(&minified, data); err != nil {
		slog.Warn("Failed to minify JSON, using original data", "type", ds.Type, "error", err)
	} else {
		data = minified.Bytes()
		slog.Debug("JSON minified", "type", ds.Type, "original", originalSize, "minified", len(data), "saved", originalSize-len(data))
	}

	filename := fmt.Sprintf("%s.json", ds.Type)
	filePath := filepath.Join(d.outputDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	slog.Debug("File saved", "path", filePath, "size", len(data))

	return nil
}

// buildURL はベースURLとパラメータから完全なURLを構築します
func (d *Downloader) buildURL(baseURL string, params any) (string, error) {
	if params == nil {
		return baseURL, nil
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if paramSlice, ok := params.([]any); ok {
		query := parsedURL.Query()

		for _, paramItem := range paramSlice {
			if paramMap, ok := paramItem.(map[string]any); ok {
				key, keyOK := paramMap["key"].(string)
				value, valueOK := paramMap["value"].(string)

				if keyOK && valueOK {
					query.Set(key, value)
				}
			}
		}

		parsedURL.RawQuery = query.Encode()
	}

	return parsedURL.String(), nil
}

// downloadMainframe はmainframeデータソース(Googleスプレッドシート)から専用の方法でダウンロードします
func (d *Downloader) downloadMainframe(ds Datasource) error {
	// paramsからapiKeyとsheetIDを取得
	paramsMap, ok := ds.Params.(map[string]string)
	if !ok {
		return fmt.Errorf("mainframe params must be map[string]string, got %T", ds.Params)
	}

	apiKey, ok := paramsMap["apiKey"]
	if !ok || apiKey == "" {
		return fmt.Errorf("apiKey not found in mainframe params")
	}

	sheetID, ok := paramsMap["sheetID"]
	if !ok || sheetID == "" {
		return fmt.Errorf("sheetID not found in mainframe params")
	}

	baseURL, ok := paramsMap["baseURL"]
	if !ok || baseURL == "" {
		return fmt.Errorf("baseURL not found in mainframe params")
	}

	// MainframeDownloaderを使用してダウンロード
	mainframeDownloader := NewMainframeDownloader(d.outputDir, apiKey, sheetID, baseURL)
	return mainframeDownloader.Download()
}

// downloadAdditionalSongs はadditional_songsデータソース(Googleスプレッドシート)から専用の方法でダウンロードします
func (d *Downloader) downloadAdditionalSongs(ds Datasource) error {
	// paramsからapiKeyとsheetIDを取得
	paramsMap, ok := ds.Params.(map[string]string)
	if !ok {
		return fmt.Errorf("additional_songs params must be map[string]string, got %T", ds.Params)
	}

	apiKey, ok := paramsMap["apiKey"]
	if !ok || apiKey == "" {
		return fmt.Errorf("apiKey not found in additional_songs params")
	}

	sheetID, ok := paramsMap["sheetID"]
	if !ok || sheetID == "" {
		return fmt.Errorf("sheetID not found in additional_songs params")
	}

	baseURL, ok := paramsMap["baseURL"]
	if !ok || baseURL == "" {
		return fmt.Errorf("baseURL not found in additional_songs params")
	}

	// AdditionalSongsDownloaderを使用してダウンロード
	additionalSongsDownloader := NewAdditionalSongsDownloader(d.outputDir, apiKey, sheetID, baseURL)
	return additionalSongsDownloader.Download()
}
