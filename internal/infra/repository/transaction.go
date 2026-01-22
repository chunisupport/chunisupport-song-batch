package repository

import (
	"context"
	"fmt"
	"log/slog"

	domainrepo "chunisupport-song-batch/internal/domain/repository"

	"github.com/jmoiron/sqlx"
)

// transactionManagerImpl は TransactionManager インターフェースの実装です。
type transactionManagerImpl struct {
	db *sqlx.DB
}

// NewTransactionManager は TransactionManager を生成します。
func NewTransactionManager(db *sqlx.DB) domainrepo.TransactionManager {
	return &transactionManagerImpl{db: db}
}

// Transactional は処理 f をトランザクション内で実行します。
func (tm *transactionManagerImpl) Transactional(ctx context.Context, f func(tx domainrepo.ExtendedDBExecutor) error) (err error) {
	slog.Debug("Beginning transaction")
	tx, err := tm.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			slog.Debug("Rolling back transaction due to panic")
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("Failed to rollback transaction after panic", "error", rbErr)
			}
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			slog.Debug("Rolling back transaction due to error", "error", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("Failed to rollback transaction", "error", rbErr)
			} // err is non-nil; don't change it
		} else {
			slog.Debug("Committing transaction")
			err = tx.Commit() // if Commit returns error, update err
			if err == nil {
				slog.Debug("Transaction committed successfully")
			}
		}
	}()

	err = f(tx)
	return err
}
