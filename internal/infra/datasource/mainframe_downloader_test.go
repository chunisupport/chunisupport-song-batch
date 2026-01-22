package datasource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMainframeDownloader_parseSheetData(t *testing.T) {
	// テスト用のダミーデータ
	mockData := &batchGetResponse{
		ValueRanges: []struct {
			Range  string     `json:"range"`
			Values [][]string `json:"values"`
		}{
			{
				Range: "Sheet1",
				Values: [][]string{
					{"曲名1", "BAS", "POPS", "", "10.5"},
					{"曲名2", "ADV", "niconico", "", "12.3"},
					{"曲名3", "EXP", "ORIGINAL", "", "13.7"},
				},
			},
			{
				Range: "Sheet2",
				Values: [][]string{
					{"曲名4", "MAS", "東方Project", "", "14.2"},
					{"曲名5", "ULT", "VARIETY", "", "15.0"},
				},
			},
		},
	}

	d := NewMainframeDownloader("test_output", "dummy_key", "dummy_id", "https://sheets.googleapis.com/v4/spreadsheets")
	charts := d.parseSheetData(mockData)

	if len(charts) != 5 {
		t.Errorf("Expected 5 charts, got %d", len(charts))
	}

	// 重複排除のテスト
	mockDataWithDuplicates := &batchGetResponse{
		ValueRanges: []struct {
			Range  string     `json:"range"`
			Values [][]string `json:"values"`
		}{
			{
				Range: "Sheet1",
				Values: [][]string{
					{"曲名1", "BAS", "POPS", "", "10.5"},
					{"曲名1", "BAS", "POPS", "", "10.5"}, // 重複
				},
			},
		},
	}

	chartsWithDuplicates := d.parseSheetData(mockDataWithDuplicates)
	if len(chartsWithDuplicates) != 1 {
		t.Errorf("Expected 1 chart after deduplication, got %d", len(chartsWithDuplicates))
	}
}

func TestMainframeDownloader_isDifficultyShort(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"BAS", true},
		{"ADV", true},
		{"EXP", true},
		{"MAS", true},
		{"ULT", true},
		{"BASIC", false},
		{"ADVANCED", false},
		{"OTHER", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isDifficultyShort(tc.input)
		if result != tc.expected {
			t.Errorf("isDifficultyShort(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestMainframeDownloader_Download_Integration(t *testing.T) {
	// 統合テストは環境変数が必要なためスキップ可能に
	apiKey := os.Getenv("CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY")
	sheetID := os.Getenv("CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID")

	if apiKey == "" || sheetID == "" {
		t.Skip("Skipping integration test: CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY or CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID not set")
	}

	// 一時ディレクトリを作成
	tempDir := t.TempDir()

	downloader := NewMainframeDownloader(tempDir, apiKey, sheetID, "https://sheets.googleapis.com/v4/spreadsheets")

	err := downloader.Download()
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// ファイルが作成されたか確認
	filePath := filepath.Join(tempDir, "mainframe.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist", filePath)
	}

	// JSONが正しくパースできるか確認
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var charts []MainframeChartData
	if err := json.Unmarshal(data, &charts); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(charts) == 0 {
		t.Error("Expected at least one chart in the output")
	}

	t.Logf("Successfully downloaded and parsed %d charts", len(charts))
}
