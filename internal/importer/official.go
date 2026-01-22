package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// OfficialImporter は公式データソースによって生成されたデータをロードします
type OfficialImporter struct{}

// NewOfficialImporter はOfficialImporterのインスタンスを返します
func NewOfficialImporter() *OfficialImporter {
	return &OfficialImporter{}
}

// Import は公式のJSONファイルを読み込み、ImportResultを生成します
func (oi *OfficialImporter) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting official data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("Official datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceOfficial, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read official data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var officialData OfficialData
	if err := json.Unmarshal(data, &officialData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal official JSON: %w", err)
	}

	// すべてのフィールドの前後の空白を除去
	for i := range officialData {
		officialData[i].ID = strings.TrimSpace(officialData[i].ID)
		officialData[i].Catname = strings.TrimSpace(officialData[i].Catname)
		officialData[i].Newflag = strings.TrimSpace(officialData[i].Newflag)
		officialData[i].Title = strings.TrimSpace(officialData[i].Title)
		officialData[i].Reading = strings.TrimSpace(officialData[i].Reading)
		officialData[i].Artist = strings.TrimSpace(officialData[i].Artist)
		officialData[i].LevBas = strings.TrimSpace(officialData[i].LevBas)
		officialData[i].LevAdv = strings.TrimSpace(officialData[i].LevAdv)
		officialData[i].LevExp = strings.TrimSpace(officialData[i].LevExp)
		officialData[i].LevMas = strings.TrimSpace(officialData[i].LevMas)
		officialData[i].LevUlt = strings.TrimSpace(officialData[i].LevUlt)
		officialData[i].WeKanji = strings.TrimSpace(officialData[i].WeKanji)
		officialData[i].WeStar = strings.TrimSpace(officialData[i].WeStar)
		officialData[i].Image = strings.TrimSpace(officialData[i].Image)
	}

	// WORLD'S END楽曲と通常楽曲をカウント
	var normalCount, worldsEndCount int
	for _, song := range officialData {
		if song.WeKanji != "" || song.WeStar != "" {
			worldsEndCount++
		} else {
			normalCount++
		}
	}

	slog.Info("Successfully loaded official music data",
		"total", len(officialData),
		"normal", normalCount,
		"worldsend", worldsEndCount)

	return &ImportResult{
		Type: DataSourceOfficial,
		Data: &officialData,
	}, nil
}
