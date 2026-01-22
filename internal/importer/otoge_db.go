package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// OtogeDbImporter はotoge-dbデータソースによって生成されたデータをロードします
type OtogeDbImporter struct{}

// NewOtogeDbImporter はOtogeDbImporterのインスタンスを返します
func NewOtogeDbImporter() *OtogeDbImporter {
	return &OtogeDbImporter{}
}

// Import はotoge-dbのJSONファイルを読み込み、ImportResultを生成します
func (oi *OtogeDbImporter) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting otoge-db data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("Otoge-db datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceOtogeDb, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read otoge-db data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var otogeDbData OtogeDbData
	if err := json.Unmarshal(data, &otogeDbData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otoge-db JSON: %w", err)
	}

	// すべてのフィールドの前後の空白を除去
	for i := range otogeDbData {
		otogeDbData[i].ID = strings.TrimSpace(otogeDbData[i].ID)
		otogeDbData[i].Title = strings.TrimSpace(otogeDbData[i].Title)
		otogeDbData[i].DateAdded = strings.TrimSpace(otogeDbData[i].DateAdded)
		otogeDbData[i].Version = strings.TrimSpace(otogeDbData[i].Version)
		otogeDbData[i].Catname = strings.TrimSpace(otogeDbData[i].Catname)
		otogeDbData[i].Artist = strings.TrimSpace(otogeDbData[i].Artist)
	}

	slog.Info("Successfully loaded otoge-db song data", "count", len(otogeDbData))

	return &ImportResult{
		Type: DataSourceOtogeDb,
		Data: &otogeDbData,
	}, nil
}
