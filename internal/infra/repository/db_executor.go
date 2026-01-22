package repository

import (
	"context"
	"database/sql"

	domainrepo "chunisupport-song-batch/internal/domain/repository"

	"github.com/jmoiron/sqlx"
)

// DBorTx は sqlx.DB と sqlx.Tx の両方を満たす汎用インターフェースです。
// domain/repository.ExtendedDBExecutor を満たすため、Context 版のメソッドも含みます。
type DBorTx interface {
	// Context なしのメソッド（レガシー互換用）
	Get(dest any, query string, args ...any) error
	Select(dest any, query string, args ...any) error
	Exec(query string, args ...any) (sql.Result, error)
	NamedExec(query string, arg any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	Queryx(query string, args ...any) (*sqlx.Rows, error)
	QueryRowx(query string, args ...any) *sqlx.Row

	// Context 版メソッド（domain/repository.DBExecutor を満たす）
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// 拡張メソッド（domain/repository.ExtendedDBExecutor を満たす）
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	Rebind(query string) string
}

var _ DBorTx = (*sqlx.DB)(nil)
var _ DBorTx = (*sqlx.Tx)(nil)

// DBorTx が domain/repository.ExtendedDBExecutor を満たすことを保証
var _ domainrepo.ExtendedDBExecutor = (DBorTx)(nil)
