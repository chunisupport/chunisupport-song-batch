package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// St1027Importer はst1027データソースによって生成されたデータをロードします
type St1027Importer struct{}

// NewSt1027Importer はSt1027Importerのインスタンスを返します
func NewSt1027Importer() *St1027Importer {
	return &St1027Importer{}
}

// Import はst1027のJSONファイルを読み込み、ImportResultを生成します
func (si *St1027Importer) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting st1027 data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("St1027 datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceSt1027, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read st1027 data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var st1027Data St1027Data
	if err := json.Unmarshal(data, &st1027Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal st1027 JSON: %w", err)
	}

	slog.Info("Successfully loaded st1027 music data", "count", len(st1027Data.Songs))

	return &ImportResult{
		Type: DataSourceSt1027,
		Data: &st1027Data,
	}, nil
}
