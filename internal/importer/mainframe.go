package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// MainframeImporter はmainframeデータソースによって生成されたデータをロードします
type MainframeImporter struct{}

// NewMainframeImporter はMainframeImporterのインスタンスを返します
func NewMainframeImporter() *MainframeImporter {
	return &MainframeImporter{}
}

// Import はmainframeのJSONファイルを読み込み、ImportResultを生成します
func (mi *MainframeImporter) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting mainframe data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("Mainframe datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceMainframe, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mainframe data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var mainframeData MainframeData
	if err := json.Unmarshal(data, &mainframeData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mainframe JSON: %w", err)
	}

	// すべてのフィールドの前後の空白を除去
	for i := range mainframeData {
		mainframeData[i].Title = strings.TrimSpace(mainframeData[i].Title)
		mainframeData[i].Diff = strings.TrimSpace(mainframeData[i].Diff)
		mainframeData[i].Genre = strings.TrimSpace(mainframeData[i].Genre)
	}

	slog.Info("Successfully loaded mainframe chart data", "count", len(mainframeData))

	return &ImportResult{
		Type: DataSourceMainframe,
		Data: &mainframeData,
	}, nil
}
