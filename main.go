// パッケージ main は、Chunisupport バッチアプリケーションのエントリーポイントを提供します。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"chunisupport-song-batch/internal/config"
	"chunisupport-song-batch/internal/datasource/registry"
	domainrepo "chunisupport-song-batch/internal/domain/repository"
	"chunisupport-song-batch/internal/importer"
	"chunisupport-song-batch/internal/info"
	"chunisupport-song-batch/internal/infra/datasource"
	"chunisupport-song-batch/internal/infra/db"
	"chunisupport-song-batch/internal/infra/repository"
	"chunisupport-song-batch/internal/service"

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

	cfg, err := config.LoadConfig(env)
	if err != nil {
		slog.Error("Failed to load config: " + err.Error())
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

	if err := executeDataImportBatch(database, cfg, flags); err != nil {
		slog.Error("Data import failed: " + err.Error())
		os.Exit(1)
	}

	slog.Info("Data Import Batch Completed Successfully")
}

// executeDataImportBatch はデータインポートプロセスを調整します。
// さまざまなソースからデータをダウンロードし、データベースにインポートし、
// データを最終テーブルに統合します。
func executeDataImportBatch(db *sqlx.DB, cfg config.Config, flags config.BatchFlags) error {
	slog.Info("Executing data import batch...")
	resolvedDatasources, resolveErr := resolveDatasources(cfg.Datasources)
	if resolveErr != nil {
		slog.Warn("Some datasources failed to resolve and will be skipped", "error", resolveErr)
	}
	if len(resolvedDatasources) == 0 {
		return fmt.Errorf("no datasources could be resolved")
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

	ctx := context.Background()
	if err := consolidateToFinalTables(ctx, db, cfg, flags, sources); err != nil {
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
		if !ds.Active {
			slog.Warn("Skipping inactive datasource", "type", ds.Type)
			continue
		}

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

// resolveDatasources は設定されたデータソースを解決します
func resolveDatasources(datasources []config.DatasourceEntry) ([]datasource.Datasource, error) {
	var (
		resolved []datasource.Datasource
		errs     []error
	)

	for _, ds := range datasources {
		definition, err := registry.Resolve(ds.Name, ds.Active)
		if err != nil {
			wrapped := fmt.Errorf("failed to resolve datasource %s: %w", ds.Name, err)
			slog.Warn("Skipping datasource due to resolution failure", "name", ds.Name, "error", err)
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

// consolidateToFinalTables はインポートされたデータを最終テーブルに統合します。
func consolidateToFinalTables(ctx context.Context, db *sqlx.DB, cfg config.Config, flags config.BatchFlags, sources service.ConsolidationSources) error {
	slog.Info("Starting data consolidation to final tables")

	// リポジトリのインスタンス生成
	difficultyRepo := repository.NewDifficultyRepository(db)
	genreRepo := repository.NewGenreRepository(db)

	opts := service.ConsolidationOptions{
		MajorUpdate: flags.MajorUpdate,
	}
	consolidationService := service.NewConsolidationService(db, difficultyRepo, genreRepo, cfg.PwPepper, cfg.Datasources, opts, sources)

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
