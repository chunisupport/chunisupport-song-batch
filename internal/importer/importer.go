package importer

// Importer はデータソースからデータをインポートするためのインターフェースです
type Importer interface {
	Import(filePath string) (*ImportResult, error)
}

// DataSourceType はデータソースの種類を表します
type DataSourceType string

const (
	// DataSourceOfficial は公式のデータソースです
	DataSourceOfficial DataSourceType = "official"
	// DataSourceNatua はnatua.devのデータソースです
	DataSourceNatua DataSourceType = "natua"
	// DataSourceMainframe はMainframeのデータソースです
	DataSourceMainframe DataSourceType = "mainframe"
	// DataSourceOtogeDb はotoge-dbのデータソースです
	DataSourceOtogeDb DataSourceType = "otoge_db"
	// DataSourceAdditionalSongs は追加楽曲のデータソースです
	DataSourceAdditionalSongs DataSourceType = "additional_songs"
)

// ImportResult はインポート処理の結果を表します
type ImportResult struct {
	Type DataSourceType
	Data any
}
