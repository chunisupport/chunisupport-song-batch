package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// AdditionalSongsImporter は追加楽曲データソースのデータをロードします
type AdditionalSongsImporter struct{}

// NewAdditionalSongsImporter はAdditionalSongsImporterのインスタンスを返します
func NewAdditionalSongsImporter() *AdditionalSongsImporter {
	return &AdditionalSongsImporter{}
}

// Import は追加楽曲のJSONファイルを読み込み、ImportResultを生成します
func (ai *AdditionalSongsImporter) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting additional_songs data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("Additional songs datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceAdditionalSongs, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read additional_songs data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var songsData AdditionalSongsData
	if err := json.Unmarshal(data, &songsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal additional_songs JSON: %w", err)
	}

	slog.Info("Additional songs data import completed",
		"songs", len(songsData.Songs),
		"charts", len(songsData.Charts),
		"we_charts", len(songsData.WECharts))

	return &ImportResult{
		Type: DataSourceAdditionalSongs,
		Data: &songsData,
	}, nil
}
