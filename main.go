// パッケージ main は、Chunisupport バッチアプリケーションのエントリーポイントを提供します。
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chunisupport/chunisupport-song-batch/internal/config"
	"github.com/chunisupport/chunisupport-song-batch/internal/datasource/registry"
	"github.com/chunisupport/chunisupport-song-batch/internal/info"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/datasource"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/db"
	"github.com/chunisupport/chunisupport-song-batch/internal/infra/repository"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"
	"github.com/chunisupport/chunisupport-song-batch/internal/usecase"

	"github.com/joho/godotenv"
)

func main() {
	os.Exit(run())
}

func run() int {
	if err := godotenv.Load(); err != nil {
		// .envが読み込めない場合は既存の環境変数を使用
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "develop"
	}

	logLevel := slog.LevelDebug
	if env == "production" {
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Data Import Batch Started - " + info.Name + " v" + info.Version)
	slog.Info("Loaded environment variables", "env", env, "log_level", logLevel.String())

	flags := config.NewBatchFlags()
	req := usecase.NewRunRequest(flags.MajorUpdate, flags.SkipDownload, flags.FillMissingReleaseDate)

	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		slog.Error("Failed to load config from environment variables: " + err.Error())
		return 1
	}

	database, err := db.Connect(cfg.Database.DbConfig)
	if err != nil {
		slog.Error("Failed to connect to database: " + err.Error())
		return 1
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		slog.Error("Failed to ping database: " + err.Error())
		return 1
	}

	slog.Info("Connected to the database")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	lock, acquired, err := db.NewAdvisoryLockProvider(database).TryAcquire(ctx, info.SongBatchLockName)
	if err != nil {
		slog.Error("Failed to acquire song-batch lock: " + err.Error())
		return 1
	}
	if !acquired {
		if req.LockConflictIsError() {
			slog.Error("Another song-batch process is running")
			return 1
		}
		slog.Info("Another song-batch process is running; skipping")
		return 0
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := lock.Release(releaseCtx); err != nil {
			slog.Error("Failed to release song-batch lock: " + err.Error())
		}
	}()

	batchUsecase := usecase.NewSongBatchUsecase(
		registryResolver{},
		func(outputDir string) usecase.Downloader {
			return downloaderAdapter{inner: datasource.NewDownloader(outputDir)}
		},
		usecase.NewFactoryImporter(),
		service.NewBatchRunner(
			database,
			repository.NewTransactionManager(database),
			repository.NewDifficultyRepository(database),
			repository.NewGenreRepository(database),
			repository.NewCourseRepository(database),
			cfg.PwPepper,
		),
		info.DatasourceCacheDir,
	)

	if err := batchUsecase.Execute(ctx, req); err != nil {
		slog.Error("Data import failed: " + err.Error())
		return 1
	}

	slog.Info("Data Import Batch Completed Successfully")
	return 0
}

type registryResolver struct{}

func (registryResolver) Resolve(name string) (usecase.DatasourceRef, error) {
	ds, err := registry.Resolve(name)
	if err != nil {
		return usecase.DatasourceRef{}, err
	}
	return usecase.DatasourceRef{Type: ds.Type, URL: ds.URL, Params: ds.Params}, nil
}

type downloaderAdapter struct {
	inner *datasource.Downloader
}

func (a downloaderAdapter) DownloadAll(ctx context.Context, datasources []usecase.DatasourceRef) ([]usecase.DownloadResult, error) {
	converted := make([]datasource.Datasource, len(datasources))
	for i, ds := range datasources {
		converted[i] = datasource.Datasource{Type: ds.Type, URL: ds.URL, Params: ds.Params}
	}
	results, err := a.inner.DownloadAll(ctx, converted)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.DownloadResult, len(results))
	for i, result := range results {
		out[i] = usecase.DownloadResult{
			Type:      result.Type,
			Success:   result.Success,
			Path:      result.Path,
			FetchedAt: result.FetchedAt,
			Bytes:     result.Bytes,
			Error:     result.Error,
		}
	}
	return out, nil
}
