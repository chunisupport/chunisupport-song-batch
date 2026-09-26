package importer

import "errors"

// ErrValidation によりユースケースは解析失敗と検証失敗を別の段階として記録できます。
var ErrValidation = errors.New("datasource validation failed")

// Importer はデータソースからデータをインポートするためのインターフェースです
type Importer interface {
	Import(filePath string) (*ImportResult, error)
}

// DataSourceType はデータソースの種類を表します
type DataSourceType string

const (
	// DataSourceOfficial は公式のデータソースです
	DataSourceOfficial DataSourceType = "official"
	// DataSourceMainframe はMainframeのデータソースです
	DataSourceMainframe DataSourceType = "mainframe"
	// DataSourceOtogeDb はotoge-dbのデータソースです
	DataSourceOtogeDb DataSourceType = "otoge_db"
	// DataSourceAdditionalSongs は追加楽曲データソースです
	DataSourceAdditionalSongs DataSourceType = "additional_songs"
	// DataSourceSt1027 はst1027のデータソースです
	DataSourceSt1027 DataSourceType = "st1027"
)

// ImportResult はインポート結果を表します
type ImportResult struct {
	Type DataSourceType
	Data any
}
