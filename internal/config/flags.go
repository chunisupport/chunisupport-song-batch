// Package config はアプリケーション設定の読み込みと管理を提供します。
package config

import "flag"

// BatchFlags はバッチアプリケーションのコマンドラインフラグを定義します。
type BatchFlags struct {
	SkipDownload      bool
	MajorUpdate       bool
	WorkspaceDumpPath string
}

// NewBatchFlags はバッチアプリケーションのコマンドラインフラグを解析して返します。
func NewBatchFlags() BatchFlags {
	var flags BatchFlags
	flag.BoolVar(&flags.SkipDownload, "skip-download", false, "Skip download and use existing JSON files")
	flag.BoolVar(&flags.MajorUpdate, "major-update", false, "Enable major update mode (official data only, special const handling)")
	flag.StringVar(&flags.WorkspaceDumpPath, "workspace-dump", "", "Dump consolidated workspace into a SQLite file")
	flag.Parse()
	return flags
}
