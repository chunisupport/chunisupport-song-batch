package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

// Datasource はデータソースの定義を表します
type Datasource struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	Params any    `json:"params"`
}

// DownloadResult はデータソース単位の取得結果です。
type DownloadResult struct {
	Type      string
	Success   bool
	Path      string
	FetchedAt time.Time
	Bytes     int64
	Error     string
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

// DownloadAll はすべてのデータソースをダウンロードし、ソース単位の結果を返します。
// 一部失敗は全体 error にせず、失敗した要素の Success=false として表します。
func (d *Downloader) DownloadAll(ctx context.Context, datasources []Datasource) ([]DownloadResult, error) {
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	results := make([]DownloadResult, len(datasources))
	var g errgroup.Group
	g.SetLimit(8)

	for i, ds := range datasources {
		g.Go(func() error {
			slog.Info("Downloading datasource", "type", ds.Type, "url", ds.URL)
			results[i] = d.downloadOne(ctx, ds)
			if results[i].Success {
				slog.Info("Successfully downloaded datasource",
					"type", ds.Type,
					"path", results[i].Path,
					"fetched_at", results[i].FetchedAt.Format(time.RFC3339),
					"bytes", results[i].Bytes)
			} else {
				slog.Error("Failed to download datasource", "type", ds.Type, "error", results[i].Error)
			}
			return nil
		})
	}

	_ = g.Wait()
	return results, nil
}

func (d *Downloader) downloadOne(ctx context.Context, ds Datasource) DownloadResult {
	result := DownloadResult{Type: ds.Type}
	err := retryDownload(ctx, ds.Type, func() error {
		switch ds.Type {
		case "mainframe":
			return d.downloadMainframe(ctx, ds)
		case "additional_songs":
			return d.downloadAdditionalSongs(ctx, ds)
		default:
			return d.downloadDatasource(ctx, ds)
		}
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}

	path := filepath.Join(d.outputDir, fmt.Sprintf("%s.json", ds.Type))
	fileInfo, statErr := os.Stat(path)
	if statErr != nil {
		result.Error = fmt.Sprintf("downloaded file not found: %v", statErr)
		return result
	}

	result.Success = true
	result.Path = path
	result.FetchedAt = time.Now()
	result.Bytes = fileInfo.Size()
	return result
}

// downloadDatasource は単一のデータソースをダウンロードします
func (d *Downloader) downloadDatasource(ctx context.Context, ds Datasource) error {
	finalURL, err := d.buildURL(ds.URL, ds.Params)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", finalURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "chunisupport-api/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return wrapRequestError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return httpStatusError{status: resp.StatusCode}
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
func (d *Downloader) downloadMainframe(ctx context.Context, ds Datasource) error {
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

	return withGoogleSheetsGate(ctx, func() error {
		mainframeDownloader := NewMainframeDownloader(d.outputDir, apiKey, sheetID, baseURL)
		return mainframeDownloader.Download(ctx)
	})
}

// downloadAdditionalSongs はadditional_songsデータソース(Googleスプレッドシート)から専用の方法でダウンロードします
func (d *Downloader) downloadAdditionalSongs(ctx context.Context, ds Datasource) error {
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

	return withGoogleSheetsGate(ctx, func() error {
		additionalSongsDownloader := NewAdditionalSongsDownloader(d.outputDir, apiKey, sheetID, baseURL)
		return additionalSongsDownloader.Download(ctx)
	})
}
