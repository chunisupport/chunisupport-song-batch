// パッケージ main は、Chunisupport バッチアプリケーションのエントリーポイントを提供します。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/chunisupport/chunisupport-song-batch/internal/config"
	"github.com/chunisupport/chunisupport-song-batch/internal/datasource/registry"
	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/datasource"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/db"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

// main はバッチアプリケーションのエントリーポイントです。
// ロガー、設定、データベース接続を初期化し、
// 次にデータインポートバッチプロセスを実行します。
func main() {
	// .envファイルを先に読み込んでAPP_ENVを取得
	if err := godotenv.Load(); err != nil {
		// .envが読み込めない場合は既存の環境変数を使用
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "develop"
	}

	// APP_ENVに基づいてログレベルを設定
	logLevel := slog.LevelDebug
	if env == "production" {
		logLevel = slog.LevelInfo
	}

	// 標準のTextHandlerを使用してロガーを初期化
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Data Import Batch Started - " + info.Name + " v" + info.Version)
	slog.Info("Loaded environment variables", "env", env, "log_level", logLevel.String())

	flags := config.NewBatchFlags()

	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		slog.Error("Failed to load config from environment variables: " + err.Error())
		os.Exit(1)
	}

	database, err := db.Connect(cfg.Database.DbConfig)
	if err != nil {
		slog.Error("Failed to connect to database: " + err.Error())
		os.Exit(1)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		slog.Error("Failed to ping database: " + err.Error())
		os.Exit(1)
	}

	slog.Info("Connected to the database")

	if err := executeDataImportBatch(database, cfg.PwPepper, flags); err != nil {
		slog.Error("Data import failed: " + err.Error())
		os.Exit(1)
	}

	slog.Info("Data Import Batch Completed Successfully")
}

// executeDataImportBatch はデータインポートプロセスを調整します。
// さまざまなソースからデータをダウンロードし、データベースにインポートし、
// データを最終テーブルに統合します。
func executeDataImportBatch(db *sqlx.DB, pwPepper string, flags config.BatchFlags) error {
	slog.Info("Executing data import batch...")
	resolvedDatasources, resolveErr := resolveAllDatasources()
	if resolveErr != nil {
		slog.Warn("Some datasources failed to resolve and will be skipped", "error", resolveErr)
	}
	if len(resolvedDatasources) == 0 {
		return fmt.Errorf("no datasources could be resolved")
	}
	if err := validateMajorUpdateDatasources(flags, resolvedDatasources); err != nil {
		return err
	}

	if flags.SkipDownload {
		slog.Info("Skipping download, using existing JSON files")
	} else {
		if err := downloadDatasources(resolvedDatasources); err != nil {
			slog.Error("Some or all datasources failed to download: " + err.Error())
			slog.Warn("Continuing batch process with available datasources...")
		}
	}
	sources, err := importDataByDatasources(resolvedDatasources)
	if err != nil {
		return err
	}
	if err := validateMajorUpdateSources(flags, sources); err != nil {
		return err
	}

	ctx := context.Background()
	if err := consolidateToFinalTables(ctx, db, pwPepper, flags, sources, resolvedDatasources); err != nil {
		return fmt.Errorf("failed to consolidate data: %w", err)
	}

	slog.Info("Data import completed")
	return nil
}

// downloadDatasources は設定されたデータソースからデータファイルをダウンロードします。
func downloadDatasources(datasources []datasource.Datasource) error {
	slog.Info("Starting datasource download process")

	const outputDir = ".datasources"

	downloader := datasource.NewDownloader(outputDir)

	if err := downloader.DownloadAll(datasources); err != nil {
		return err
	}

	slog.Info("Datasource download completed")
	return nil
}

// importDataByDatasources はダウンロードされた JSON ファイルからデータをデータベースにインポートします。
func importDataByDatasources(datasources []datasource.Datasource) (service.ConsolidationSources, error) {
	slog.Info("Starting data import for all datasources")

	var sources service.ConsolidationSources
	const outputDir = ".datasources"
	factory := importer.NewImporterFactory()
	for _, ds := range datasources {
		slog.Info("Processing datasource", "type", ds.Type)
		dsImporter, err := factory.CreateImporter(importer.DataSourceType(ds.Type))
		if err != nil {
			return sources, fmt.Errorf("failed to create importer for type %s: %w", ds.Type, err)
		}
		filePath := fmt.Sprintf("%s/%s.json", outputDir, ds.Type)
		result, err := dsImporter.Import(filePath)
		if err != nil {
			return sources, fmt.Errorf("failed to load data for type %s from %s: %w", ds.Type, filePath, err)
		}
		if result.Data == nil {
			slog.Warn("No data loaded for datasource", "type", ds.Type)
			continue
		}

		switch importer.DataSourceType(ds.Type) {
		case importer.DataSourceOfficial:
			if data, ok := result.Data.(*importer.OfficialData); ok {
				sources.Official = data
			} else {
				return sources, fmt.Errorf("unexpected data type for official datasource: %T", result.Data)
			}
		case importer.DataSourceNatua:
			if data, ok := result.Data.(*importer.NatuaData); ok {
				sources.Natua = data
			} else {
				return sources, fmt.Errorf("unexpected data type for natua datasource: %T", result.Data)
			}
		case importer.DataSourceSt1027:
			if data, ok := result.Data.(*importer.St1027Data); ok {
				sources.St1027 = data
			} else {
				return sources, fmt.Errorf("unexpected data type for st1027 datasource: %T", result.Data)
			}
		case importer.DataSourceMainframe:
			if data, ok := result.Data.(*importer.MainframeData); ok {
				sources.Mainframe = data
			} else {
				return sources, fmt.Errorf("unexpected data type for mainframe datasource: %T", result.Data)
			}
		case importer.DataSourceOtogeDb:
			if data, ok := result.Data.(*importer.OtogeDbData); ok {
				sources.OtogeDb = data
			} else {
				return sources, fmt.Errorf("unexpected data type for otoge-db datasource: %T", result.Data)
			}
		case importer.DataSourceAdditionalSongs:
			if data, ok := result.Data.(*importer.AdditionalSongsData); ok {
				sources.AdditionalSongs = data
			} else {
				return sources, fmt.Errorf("unexpected data type for additional_songs datasource: %T", result.Data)
			}
		default:
			slog.Warn("Loaded data for unsupported datasource", "type", ds.Type)
		}

		slog.Info("Successfully imported data", "type", ds.Type)
	}

	slog.Info("Data import for all datasources completed")
	return sources, nil
}

// resolveAllDatasources はサポートされている全データソースを解決します。
func resolveAllDatasources() ([]datasource.Datasource, error) {
	var (
		resolved []datasource.Datasource
		errs     []error
	)
	factory := importer.NewImporterFactory()
	supported := factory.GetSupportedDataSources()

	for _, sourceType := range supported {
		name := string(sourceType)
		definition, err := registry.Resolve(name)
		if err != nil {
			wrapped := fmt.Errorf("failed to resolve datasource %s: %w", name, err)
			slog.Warn("Skipping datasource due to resolution failure", "name", name, "error", err)
			errs = append(errs, wrapped)
			continue
		}

		resolved = append(resolved, definition)
	}

	if len(errs) > 0 {
		return resolved, errors.Join(errs...)
	}

	return resolved, nil
}

func validateMajorUpdateDatasources(flags config.BatchFlags, datasources []datasource.Datasource) error {
	if !flags.MajorUpdate {
		return nil
	}

	required := map[string]bool{
		"official":         false,
		"additional_songs": false,
	}

	for _, ds := range datasources {
		if _, ok := required[ds.Type]; ok {
			required[ds.Type] = true
		}
	}

	missing := make([]string, 0, len(required))
	for name, exists := range required {
		if !exists {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("major update requires datasources to be resolved: %v", missing)
	}

	return nil
}

func datasourceNames(datasources []datasource.Datasource) []string {
	names := make([]string, 0, len(datasources))
	for _, ds := range datasources {
		names = append(names, ds.Type)
	}
	return names
}

func validateMajorUpdateSources(flags config.BatchFlags, sources service.ConsolidationSources) error {
	if !flags.MajorUpdate {
		return nil
	}

	if sources.Official == nil {
		return fmt.Errorf("major update requires official datasource data")
	}
	if sources.AdditionalSongs == nil {
		return fmt.Errorf("major update requires additional_songs datasource data")
	}

	return nil
}

// consolidateToFinalTables はインポートされたデータを最終テーブルに統合します。
func consolidateToFinalTables(ctx context.Context, db *sqlx.DB, pwPepper string, flags config.BatchFlags, sources service.ConsolidationSources, datasources []datasource.Datasource) error {
	slog.Info("Starting data consolidation to final tables")

	// リポジトリのインスタンス生成
	difficultyRepo := repository.NewDifficultyRepository(db)
	genreRepo := repository.NewGenreRepository(db)

	opts := service.ConsolidationOptions{
		MajorUpdate: flags.MajorUpdate,
	}
	consolidationService := service.NewConsolidationService(db, difficultyRepo, genreRepo, pwPepper, datasourceNames(datasources), opts, sources)

	workspace, err := consolidationService.BuildWorkspace(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return nil
	}
	defer workspace.Close()

	tm := repository.NewTransactionManager(db)
	return tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		return consolidationService.SyncWorkspace(ctx, workspace, tx)
	})
}
