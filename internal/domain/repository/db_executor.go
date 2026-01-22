package repository

import (
	"context"
	"database/sql"
)

// DBExecutor はデータベース操作の抽象インターフェースです。
// Clean Architecture の原則に従い、ドメイン層で定義し、インフラ層で実装します。
// すべてのメソッドは Context を受け取り、キャンセルやタイムアウトを伝播できます。
type DBExecutor interface {
	// ExecContext は INSERT/UPDATE/DELETE などのクエリを実行します。
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	// QueryContext は複数行を返すクエリを実行します。
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ExtendedDBExecutor は DBExecutor を拡張し、ORMライブラリ固有の機能を提供します。
// sqlx の SelectContext 等を抽象化し、service層での型キャストを不要にします。
type ExtendedDBExecutor interface {
	DBExecutor
	// SelectContext は複数行をスライスにスキャンします。
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	// GetContext は単一行をスキャンします。
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	// Rebind はプレースホルダをドライバ固有の形式に変換します。
	Rebind(query string) string
}
