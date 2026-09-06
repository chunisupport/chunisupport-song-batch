package usecase

import (
	"fmt"
	"slices"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"
)

func targetDatasourceTypes(mode RunMode) []string {
	if mode == RunModeMajorUpdate {
		return []string{
			string(importer.DataSourceOfficial),
			string(importer.DataSourceAdditionalSongs),
		}
	}
	types := importer.NewImporterFactory().GetSupportedDataSources()
	names := make([]string, len(types))
	for i, sourceType := range types {
		names[i] = string(sourceType)
	}
	return names
}

func requiredDatasourceTypes(mode RunMode) []string {
	required := []string{
		string(importer.DataSourceOfficial),
		string(importer.DataSourceAdditionalSongs),
	}
	if mode != RunModeMajorUpdate {
		required = append(required, string(importer.DataSourceMainframe))
	}
	return required
}

func isRequired(mode RunMode, sourceType string) bool {
	return slices.Contains(requiredDatasourceTypes(mode), sourceType)
}

func allowsLastKnownGood(sourceType string) bool {
	switch importer.DataSourceType(sourceType) {
	case importer.DataSourceSt1027, importer.DataSourceOtogeDb:
		return true
	default:
		return false
	}
}

func assignSource(sources *service.ConsolidationSources, sourceType string, data any) error {
	switch importer.DataSourceType(sourceType) {
	case importer.DataSourceOfficial:
		typed, ok := data.(*importer.OfficialData)
		if !ok {
			return fmt.Errorf("unexpected data type for official datasource: %T", data)
		}
		sources.Official = typed
	case importer.DataSourceAdditionalSongs:
		typed, ok := data.(*importer.AdditionalSongsData)
		if !ok {
			return fmt.Errorf("unexpected data type for additional_songs datasource: %T", data)
		}
		sources.AdditionalSongs = typed
	case importer.DataSourceSt1027:
		typed, ok := data.(*importer.St1027Data)
		if !ok {
			return fmt.Errorf("unexpected data type for st1027 datasource: %T", data)
		}
		sources.St1027 = typed
	case importer.DataSourceMainframe:
		typed, ok := data.(*importer.MainframeData)
		if !ok {
			return fmt.Errorf("unexpected data type for mainframe datasource: %T", data)
		}
		sources.Mainframe = typed
	case importer.DataSourceOtogeDb:
		typed, ok := data.(*importer.OtogeDbData)
		if !ok {
			return fmt.Errorf("unexpected data type for otoge-db datasource: %T", data)
		}
		sources.OtogeDb = typed
	default:
		return fmt.Errorf("unsupported datasource type: %s", sourceType)
	}
	return nil
}
