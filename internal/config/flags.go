// Package config はアプリケーション設定の読み込みと管理を提供します。
package config

import "flag"

// BatchFlags はバッチアプリケーションのコマンドラインフラグを定義します。
type BatchFlags struct {
	SkipDownload           bool
	MajorUpdate            bool
	FillMissingReleaseDate bool // 特定フラグ: データソース・MySQL両方に日付がないbrand new楽曲へ実行日(JST)を補完
}

// NewBatchFlags はバッチアプリケーションのコマンドラインフラグを解析して返します。
func NewBatchFlags() BatchFlags {
	var flags BatchFlags
	flag.BoolVar(&flags.SkipDownload, "skip-download", false, "Skip download and use existing JSON files")
	flag.BoolVar(&flags.MajorUpdate, "major-update", false, "Enable major update mode (official data only, special const handling)")
	flag.BoolVar(&flags.FillMissingReleaseDate, "fill-missing-release-date", false, "Fill missing release date (released_at) with execution date (JST) for brand new songs with no date from any datasource and not existing in MySQL")
	flag.Parse()
	return flags
}
