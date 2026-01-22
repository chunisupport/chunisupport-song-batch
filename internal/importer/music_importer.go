package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// MusicImporter は楽曲ベースのインポーター向けの共有機能を提供します。
type MusicImporter struct{}

// NewMusicImporter はMusicImporterのインスタンスを準備します。
func NewMusicImporter() *MusicImporter {
	return &MusicImporter{}
}

// LoadFromFile はデータソースのJSONファイルをメモリに読み込みます。
func (mi *MusicImporter) LoadFromFile(filePath string) (MusicCollection, error) {
	slog.Info("Loading music data from file", "file", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var musicCollection MusicCollection
	if err := json.Unmarshal(data, &musicCollection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	slog.Info("Successfully loaded music data", "count", len(musicCollection))
	return musicCollection, nil
}

// ImportToDB はMusicCollectionをダウンストリームのリポジトリに保存します。
func (mi *MusicImporter) ImportToDB(musicData MusicCollection) error {
	slog.Info("Starting database import", "records", len(musicData))

	for i, music := range musicData {
		if i < 3 {
			slog.Info("Processing music record",
				"id", music.ID,
				"title", music.Title,
				"artist", music.Artist,
				"genre", music.Catname)
		}
	}

	slog.Info("Database import completed (dummy implementation)")
	return nil
}
