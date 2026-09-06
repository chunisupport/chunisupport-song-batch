package usecase

import (
	"fmt"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
)

type factoryImporter struct {
	factory *importer.ImporterFactory
}

// NewFactoryImporter は ImporterFactory を使う SourceImporter を返します。
func NewFactoryImporter() SourceImporter {
	return &factoryImporter{factory: importer.NewImporterFactory()}
}

func (f *factoryImporter) Import(sourceType, filePath string) (*importer.ImportResult, error) {
	dsImporter, err := f.factory.CreateImporter(importer.DataSourceType(sourceType))
	if err != nil {
		return nil, fmt.Errorf("failed to create importer for type %s: %w", sourceType, err)
	}
	return dsImporter.Import(filePath)
}
