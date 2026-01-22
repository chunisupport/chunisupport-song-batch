package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// NatuaImporter はNatuaデータソースによって生成されたデータをロードします
type NatuaImporter struct{}

// NewNatuaImporter はNatuaImporterのインスタンスを返します
func NewNatuaImporter() *NatuaImporter {
	return &NatuaImporter{}
}

// Import はNatuaのJSONファイルを読み込み、ImportResultを生成します
func (ni *NatuaImporter) Import(filePath string) (*ImportResult, error) {
	slog.Info("Starting natua data import", "file", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Warn("Natua datasource file not found, skipping import", "path", filePath)
		return &ImportResult{Type: DataSourceNatua, Data: nil}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read natua data file %s: %w", filePath, err)
	}

	// BOMを除去
	data = removeBOM(data)

	var natuaData NatuaData
	if err := json.Unmarshal(data, &natuaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal natua JSON: %w", err)
	}

	// すべての文字列フィールドの前後の空白を除去
	for i := range natuaData.Songs {
		song := &natuaData.Songs[i]
		song.Meta.OfficialID = strings.TrimSpace(song.Meta.OfficialID)
		song.Meta.Name = strings.TrimSpace(song.Meta.Name)
		if song.Meta.Reading != nil {
			trimmed := strings.TrimSpace(*song.Meta.Reading)
			song.Meta.Reading = &trimmed
		}
		song.Meta.Artist = strings.TrimSpace(song.Meta.Artist)
		song.Meta.Genre = strings.TrimSpace(song.Meta.Genre)
		song.Meta.Release = strings.TrimSpace(song.Meta.Release)
		song.Meta.ReleaseVersion = strings.TrimSpace(song.Meta.ReleaseVersion)
		song.Meta.ImageURL = strings.TrimSpace(song.Meta.ImageURL)
		song.Meta.FumenID = strings.TrimSpace(song.Meta.FumenID)
		if song.Meta.WeStar != nil {
			trimmed := strings.TrimSpace(*song.Meta.WeStar)
			song.Meta.WeStar = &trimmed
		}
		if song.Meta.WeKanji != nil {
			trimmed := strings.TrimSpace(*song.Meta.WeKanji)
			song.Meta.WeKanji = &trimmed
		}
		// 各難易度のnotesdesignerもトリミング
		for _, chart := range []*NatuaChart{&song.Basic, &song.Advanced, &song.Expert, &song.Master, &song.Ultima} {
			if chart.Notesdesigner != nil {
				trimmed := strings.TrimSpace(*chart.Notesdesigner)
				chart.Notesdesigner = &trimmed
			}
		}
	}

	slog.Info("Successfully loaded natua music data", "count", len(natuaData.Songs))

	return &ImportResult{
		Type: DataSourceNatua,
		Data: &natuaData,
	}, nil
}
