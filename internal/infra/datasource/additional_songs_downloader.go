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
	"strings"
	"time"
)

// AdditionalSongsDownloader はGoogleスプレッドシートから追加楽曲データをダウンロードします
type AdditionalSongsDownloader struct {
	httpClient *http.Client
	outputDir  string
	apiKey     string
	sheetID    string
	baseURL    string
}

// additionalSongsSheetData はJSONに保存する追加楽曲データ構造
type additionalSongsSheetData struct {
	Songs    []additionalSongRow    `json:"songs"`
	Charts   []additionalChartRow   `json:"charts"`
	WECharts []additionalWEChartRow `json:"we_charts"`
}

// additionalSongRow は additional_songs シートの1行
type additionalSongRow struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Artist  string   `json:"artist"`
	Genre   string   `json:"genre"`
	Release string   `json:"release"`
	BPM     *int     `json:"bpm"`
	BasNt   *int     `json:"basnt"`
	Bas     *float64 `json:"bas"`
	BasUK   bool     `json:"basuk"`
	AdvNt   *int     `json:"advnt"`
	Adv     *float64 `json:"adv"`
	AdvUK   bool     `json:"advuk"`
	ExpNt   *int     `json:"expnt"`
	Exp     *float64 `json:"exp"`
	ExpUK   bool     `json:"expuk"`
	MasNt   *int     `json:"masnt"`
	Mas     *float64 `json:"mas"`
	MasUK   bool     `json:"masuk"`
	UltNt   *int     `json:"ultnt"`
	Ult     *float64 `json:"ult"`
	UltUK   bool     `json:"ultuk"`
	Img     string   `json:"img"`
}

// additionalChartRow は additional_charts シートの1行
type additionalChartRow struct {
	ID    string   `json:"id"`
	Diff  string   `json:"diff"`
	Const *float64 `json:"const"`
	CsUK  bool     `json:"csuk"`
	Notes *int     `json:"notes"`
}

// additionalWEChartRow は additional_songs_charts_we シートの1行
// カラム: id, title, artist, genre, release, we_kanji, we_star, notes, img
type additionalWEChartRow struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Genre   string `json:"genre"`
	Release string `json:"release"`
	WEKanji string `json:"we_kanji"`
	WEStar  *int   `json:"we_star"`
	Notes   *int   `json:"notes"`
	Img     string `json:"img"`
}

// NewAdditionalSongsDownloader は新しいAdditionalSongsDownloaderのインスタンスを生成します
func NewAdditionalSongsDownloader(outputDir, apiKey, sheetID, baseURL string) *AdditionalSongsDownloader {
	return &AdditionalSongsDownloader{
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
func (d *AdditionalSongsDownloader) Download() error {
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	slog.Info("Fetching additional_songs data from Google Sheets", "sheetID", d.sheetID)

	// additional_songs、additional_charts、additional_songs_charts_we の3シートを取得
	sheetNames := []string{"additional_songs", "additional_charts", "additional_songs_charts_we"}

	// データを一括取得
	allData, err := d.batchGetSheetData(sheetNames)
	if err != nil {
		return fmt.Errorf("failed to batch get sheet data: %w", err)
	}

	// データを解析
	data, err := d.parseSheetData(allData)
	if err != nil {
		return fmt.Errorf("failed to parse sheet data: %w", err)
	}

	slog.Info("Parsed additional songs data", "songs", len(data.Songs), "charts", len(data.Charts), "we_charts", len(data.WECharts))

	// JSONファイルとして保存
	filename := "additional_songs.json"
	filePath := filepath.Join(d.outputDir, filename)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	slog.Info("Successfully saved additional_songs data", "path", filePath, "size", len(jsonData))

	return nil
}

// batchGetSheetData は複数のシートのデータを一括取得します
func (d *AdditionalSongsDownloader) batchGetSheetData(sheetNames []string) (*batchGetResponse, error) {
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

// parseSheetData はスプレッドシートのデータを解析してadditionalSongsSheetDataに変換します
func (d *AdditionalSongsDownloader) parseSheetData(data *batchGetResponse) (*additionalSongsSheetData, error) {
	result := &additionalSongsSheetData{
		Songs:    []additionalSongRow{},
		Charts:   []additionalChartRow{},
		WECharts: []additionalWEChartRow{},
	}

	for _, valueRange := range data.ValueRanges {
		// rangeからシート名を抽出 (例: "additional_songs!A1:Z1000" -> "additional_songs")
		sheetName := extractSheetName(valueRange.Range)

		switch sheetName {
		case "additional_songs":
			songs, err := d.parseSongsSheet(valueRange.Values)
			if err != nil {
				return nil, fmt.Errorf("failed to parse additional_songs: %w", err)
			}
			result.Songs = songs
		case "additional_charts":
			charts, err := d.parseChartsSheet(valueRange.Values)
			if err != nil {
				return nil, fmt.Errorf("failed to parse additional_charts: %w", err)
			}
			result.Charts = charts
		case "additional_songs_charts_we":
			weCharts, err := d.parseWEChartsSheet(valueRange.Values)
			if err != nil {
				return nil, fmt.Errorf("failed to parse additional_songs_charts_we: %w", err)
			}
			result.WECharts = weCharts
		}
	}

	return result, nil
}

// parseSongsSheet は additional_songs シートのデータを解析します
// カラム: id, title, artist, genre, release, bpm, basnt, bas, basuk, advnt, adv, advuk,
//
//	expnt, exp, expuk, masnt, mas, masuk, ultnt, ult, ultuk, img
func (d *AdditionalSongsDownloader) parseSongsSheet(values [][]string) ([]additionalSongRow, error) {
	if len(values) < 2 {
		// ヘッダー行のみまたは空
		return []additionalSongRow{}, nil
	}

	var songs []additionalSongRow
	// 最初の行はヘッダーなのでスキップ
	for rowIdx, row := range values[1:] {
		if len(row) == 0 {
			continue
		}

		song := additionalSongRow{}

		// 必須フィールドのバリデーション
		song.ID = getString(row, 0)
		song.Title = getString(row, 1)
		song.Artist = getString(row, 2)
		song.Genre = getString(row, 3)
		song.Release = getString(row, 4)

		// 必須フィールドチェック
		if song.ID == "" || song.Title == "" || song.Artist == "" || song.Genre == "" || song.Release == "" {
			continue
		}

		// オプションフィールド
		song.BPM = getIntPtr(row, 5)

		// BASIC
		song.BasNt = getIntPtr(row, 6)
		song.Bas = getFloatPtr(row, 7)
		song.BasUK = getBool(row, 8)

		// ADVANCED
		song.AdvNt = getIntPtr(row, 9)
		song.Adv = getFloatPtr(row, 10)
		song.AdvUK = getBool(row, 11)

		// EXPERT
		song.ExpNt = getIntPtr(row, 12)
		song.Exp = getFloatPtr(row, 13)
		song.ExpUK = getBool(row, 14)

		// MASTER
		song.MasNt = getIntPtr(row, 15)
		song.Mas = getFloatPtr(row, 16)
		song.MasUK = getBool(row, 17)

		// ULTIMA
		song.UltNt = getIntPtr(row, 18)
		song.Ult = getFloatPtr(row, 19)
		song.UltUK = getBool(row, 20)

		// Image
		song.Img = getString(row, 21)

		// 各難易度の定数チェック（少なくともBASIC〜MASTERの4つは必須）
		if song.Bas == nil || song.Adv == nil || song.Exp == nil || song.Mas == nil {
			slog.Warn("Skipping additional_songs row with missing difficulty constants",
				"row", rowIdx+2, "id", song.ID, "title", song.Title)
			continue
		}

		songs = append(songs, song)
	}

	return songs, nil
}

// parseChartsSheet は additional_charts シートのデータを解析します
// カラム: id, diff, const, csuk, notes
func (d *AdditionalSongsDownloader) parseChartsSheet(values [][]string) ([]additionalChartRow, error) {
	if len(values) < 2 {
		// ヘッダー行のみまたは空
		return []additionalChartRow{}, nil
	}

	var charts []additionalChartRow
	// 最初の行はヘッダーなのでスキップ
	for _, row := range values[1:] {
		if len(row) == 0 {
			continue
		}

		chart := additionalChartRow{}

		// 必須フィールド
		chart.ID = getString(row, 0)
		chart.Diff = getString(row, 1)
		chart.Const = getFloatPtr(row, 2)
		chart.CsUK = getBool(row, 3)
		chart.Notes = getIntPtr(row, 4)

		// 必須フィールドチェック
		if chart.ID == "" || chart.Diff == "" || chart.Const == nil {
			continue
		}

		charts = append(charts, chart)
	}

	return charts, nil
}

// parseWEChartsSheet は additional_songs_charts_we シートのデータを解析します
// カラム: id, title, artist, genre, release, we_kanji, we_star, notes, img
func (d *AdditionalSongsDownloader) parseWEChartsSheet(values [][]string) ([]additionalWEChartRow, error) {
	if len(values) < 2 {
		// ヘッダー行のみまたは空
		return []additionalWEChartRow{}, nil
	}

	var weCharts []additionalWEChartRow
	// 最初の行はヘッダーなのでスキップ
	for rowIdx, row := range values[1:] {
		if len(row) == 0 {
			continue
		}

		weChart := additionalWEChartRow{}

		// 必須フィールド
		weChart.ID = getString(row, 0)
		weChart.Title = getString(row, 1)
		weChart.Artist = getString(row, 2)
		weChart.Genre = getString(row, 3)
		weChart.Release = getString(row, 4)
		weChart.WEKanji = getString(row, 5)
		weChart.WEStar = getIntPtr(row, 6)
		weChart.Notes = getIntPtr(row, 7)
		weChart.Img = getString(row, 8)

		// 必須フィールドチェック
		if weChart.ID == "" || weChart.Title == "" || weChart.Artist == "" || weChart.Genre == "" || weChart.Release == "" {
			slog.Warn("Skipping additional_songs_charts_we row with missing required fields",
				"row", rowIdx+2, "id", weChart.ID, "title", weChart.Title)
			continue
		}

		weCharts = append(weCharts, weChart)
	}

	return weCharts, nil
}

// extractSheetName はrange文字列からシート名を抽出します
func extractSheetName(rangeStr string) string {
	// "additional_songs!A1:Z1000" -> "additional_songs"
	// "additional_songs" -> "additional_songs"
	if idx := strings.Index(rangeStr, "!"); idx != -1 {
		return rangeStr[:idx]
	}
	return rangeStr
}

// getString は行から指定インデックスの文字列を取得します
func getString(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// getIntPtr は行から指定インデックスの整数ポインタを取得します
func getIntPtr(row []string, idx int) *int {
	s := getString(row, idx)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	if v == 0 {
		return nil // 0はnullとして扱う
	}
	return &v
}

// getFloatPtr は行から指定インデックスの浮動小数点ポインタを取得します
func getFloatPtr(row []string, idx int) *float64 {
	s := getString(row, idx)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	if v == 0 {
		return nil // 0はnullとして扱う
	}
	return &v
}

// getBool は行から指定インデックスのbool値を取得します
func getBool(row []string, idx int) bool {
	s := getString(row, idx)
	return strings.EqualFold(s, "true") || s == "1"
}
