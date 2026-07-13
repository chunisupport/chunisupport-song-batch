package datasource

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// MainframeDownloader はGoogleスプレッドシートからmainframeデータをダウンロードします
type MainframeDownloader struct {
	httpClient *http.Client
	outputDir  string
	apiKey     string
	sheetID    string
	baseURL    string
}

// MainframeChartData はmainframeデータソースの単一の譜面エントリを表します
type MainframeChartData struct {
	Title string  `json:"title"`
	Diff  string  `json:"diff"`
	Genre string  `json:"genre"`
	Const float64 `json:"const"`
}

type sheetsResponse struct {
	Sheets []struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
	} `json:"sheets"`
}

type batchGetResponse struct {
	ValueRanges []struct {
		Range  string     `json:"range"`
		Values [][]string `json:"values"`
	} `json:"valueRanges"`
}

var chunithmDifficultiesShort = []string{"BAS", "ADV", "EXP", "MAS", "ULT"}

// NewMainframeDownloader は新しいMainframeDownloaderのインスタンスを生成します
func NewMainframeDownloader(outputDir, apiKey, sheetID, baseURL string) *MainframeDownloader {
	return &MainframeDownloader{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		outputDir: outputDir,
		apiKey:    apiKey,
		sheetID:   sheetID,
		baseURL:   baseURL,
	}
}

// Download はGoogleスプレッドシートからデータをダウンロードし、JSONファイルとして保存します
func (d *MainframeDownloader) Download() error {
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	slog.Info("Fetching mainframe data from Google Sheets", "sheetID", d.sheetID)

	// Step 1: シート一覧を取得
	sheetNames, err := d.getSheetNames()
	if err != nil {
		return fmt.Errorf("failed to get sheet names: %w", err)
	}

	slog.Info("Retrieved sheet names", "count", len(sheetNames))

	// Step 2: すべてのシートのデータを一括取得
	allData, err := d.batchGetSheetData(sheetNames)
	if err != nil {
		return fmt.Errorf("failed to batch get sheet data: %w", err)
	}

	// Step 3: データを解析
	charts := d.parseSheetData(allData)

	slog.Info("Parsed mainframe chart data", "count", len(charts))

	// Step 4: JSONファイルとして保存（minify版）
	filename := "mainframe.json"
	filePath := filepath.Join(d.outputDir, filename)

	jsonData, err := json.Marshal(charts)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	slog.Info("Successfully saved mainframe data", "path", filePath, "size", len(jsonData))

	return nil
}

// getSheetNames はスプレッドシートからすべてのシート名を取得します
func (d *MainframeDownloader) getSheetNames() ([]string, error) {
	baseURL := fmt.Sprintf("%s/%s", d.baseURL, d.sheetID)
	reqURL := fmt.Sprintf("%s?key=%s", baseURL, d.apiKey)

	resp, err := d.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var sheetsResp sheetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sheetsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	sheetNames := make([]string, 0, len(sheetsResp.Sheets))
	for _, sheet := range sheetsResp.Sheets {
		sheetNames = append(sheetNames, sheet.Properties.Title)
	}

	return sheetNames, nil
}

// batchGetSheetData は複数のシートのデータを一括取得します
func (d *MainframeDownloader) batchGetSheetData(sheetNames []string) (*batchGetResponse, error) {
	baseURL := fmt.Sprintf("%s/%s/values:batchGet", d.baseURL, d.sheetID)

	// URLパラメータを構築
	params := url.Values{}
	params.Set("key", d.apiKey)
	for _, name := range sheetNames {
		params.Add("ranges", name)
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := d.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var batchResp batchGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &batchResp, nil
}

// parseSheetData はスプレッドシートのデータを解析してMainframeChartDataのスライスに変換します
func (d *MainframeDownloader) parseSheetData(data *batchGetResponse) []MainframeChartData {
	resultMap := make(map[string]MainframeChartData)

	for _, valueRange := range data.ValueRanges {
		for _, row := range valueRange.Values {
			for colIdx, cell := range row {
				if !isDifficultyShort(cell) {
					continue
				}

				// タイトル: colIdx - 1
				if colIdx-1 < 0 || colIdx-1 >= len(row) {
					continue
				}
				title := row[colIdx-1]
				if title == "" {
					continue
				}

				// ジャンル: colIdx + 1
				if colIdx+1 >= len(row) {
					continue
				}
				genre := row[colIdx+1]

				// 定数: colIdx + 3
				if colIdx+3 >= len(row) {
					continue
				}
				constStr := row[colIdx+3]
				constValue, err := strconv.ParseFloat(constStr, 64)
				if err != nil {
					// メモ：「parsing \"\": invalid syntax"」←まだ定数が入力されておらず空欄なだけ
					slog.Warn("Failed to parse constant value, skipping this chart entry",
						"title", title,
						"difficulty", cell,
						"genre", genre,
						"constStr", constStr,
						"error", err)
					continue
				}

				// 重複を排除するためにユニークキーを作成
				key := fmt.Sprintf("%s|%s|%s|%.2f", title, cell, genre, constValue)
				resultMap[key] = MainframeChartData{
					Title: title,
					Diff:  cell,
					Genre: genre,
					Const: constValue,
				}
			}
		}
	}

	// mapをスライスに変換
	result := make([]MainframeChartData, 0, len(resultMap))
	for _, chart := range resultMap {
		result = append(result, chart)
	}

	return result
}

// isDifficultyShort は文字列が難易度の略称かどうかを判定します
func isDifficultyShort(s string) bool {
	for _, diff := range chunithmDifficultiesShort {
		if s == diff {
			return true
		}
	}
	return false
}
