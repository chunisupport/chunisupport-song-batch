package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-song-batch/internal/importer"
	"github.com/chunisupport/chunisupport-song-batch/internal/service"
)

// RunMode はバッチの実行モードです。
type RunMode string

const (
	// RunModeNormal は通常実行です。
	RunModeNormal RunMode = "NORMAL"
	// RunModeMajorUpdate は大型更新です。
	RunModeMajorUpdate RunMode = "MAJOR_UPDATE"
)

// RunRequest はユースケースへの入力です。flag 名や環境変数名は含みません。
type RunRequest struct {
	Mode                   RunMode
	SkipDownload           bool
	FillMissingReleaseDate bool
}

// LockConflictIsError はロック競合時にエラー終了すべきかを返します。
func (r RunRequest) LockConflictIsError() bool {
	return r.SkipDownload || r.Mode == RunModeMajorUpdate
}

// NewRunRequest はフラグ値から実行リクエストを組み立てます。
func NewRunRequest(majorUpdate, skipDownload, fillMissingReleaseDate bool) RunRequest {
	mode := RunModeNormal
	if majorUpdate {
		mode = RunModeMajorUpdate
	}
	return RunRequest{
		Mode:                   mode,
		SkipDownload:           skipDownload,
		FillMissingReleaseDate: fillMissingReleaseDate,
	}
}

// DatasourceRef は解決済みデータソースの取得に必要な情報です。
type DatasourceRef struct {
	Type   string
	URL    string
	Params any
}

// DownloadResult はデータソース単位の取得結果です。
type DownloadResult struct {
	Type      string
	Success   bool
	Path      string
	FetchedAt time.Time
	Bytes     int64
	Error     string
}

// DatasourceResolver はデータソース定義を解決します。
type DatasourceResolver interface {
	Resolve(name string) (DatasourceRef, error)
}

// Downloader は渡されたディレクトリへデータソースを取得します。
type Downloader interface {
	DownloadAll(ctx context.Context, datasources []DatasourceRef) ([]DownloadResult, error)
}

// DownloaderFactory は実行専用ディレクトリ向けの Downloader を生成します。
type DownloaderFactory func(outputDir string) Downloader

// SourceImporter はファイルからデータソースを読み込みます。
type SourceImporter interface {
	Import(sourceType, filePath string) (*importer.ImportResult, error)
}

// Consolidator はインポート済みソースをワークスペース経由で MySQL へ同期します。
type Consolidator interface {
	Consolidate(ctx context.Context, sources service.ConsolidationSources, names []string, opts service.ConsolidationOptions) error
}
