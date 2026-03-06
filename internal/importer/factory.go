package importer

import (
	"fmt"
)

// ImporterFactory はインポーターのファクトリです
type ImporterFactory struct{}

// NewImporterFactory は新しいImporterFactoryのインスタンスを生成します
func NewImporterFactory() *ImporterFactory {
	return &ImporterFactory{}
}

// CreateImporter はデータソースの種類に応じたインポーターを生成します
func (f *ImporterFactory) CreateImporter(dataSourceType DataSourceType) (Importer, error) {
	switch dataSourceType {
	case DataSourceOfficial:
		return NewOfficialImporter(), nil
	case DataSourceNatua:
		return NewNatuaImporter(), nil
	case DataSourceMainframe:
		return NewMainframeImporter(), nil
	case DataSourceOtogeDb:
		return NewOtogeDbImporter(), nil
	case DataSourceAdditionalSongs:
		return NewAdditionalSongsImporter(), nil
	case DataSourceSt1027:
		return NewSt1027Importer(), nil
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dataSourceType)
	}
}

// GetSupportedDataSources はサポートされているデータソースの種類のリストを返します
func (f *ImporterFactory) GetSupportedDataSources() []DataSourceType {
	return []DataSourceType{
		DataSourceOfficial,
		DataSourceAdditionalSongs,
		DataSourceSt1027,
		DataSourceNatua,
		DataSourceMainframe,
		DataSourceOtogeDb,
	}
}
