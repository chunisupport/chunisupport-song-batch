package db

import (
	"context"
	"database/sql"
	"fmt"

	domainrepo "github.com/chunisupport/chunisupport-song-batch/internal/domain/repository"

	"github.com/jmoiron/sqlx"
)

type advisoryLockProvider struct {
	db *sqlx.DB
}

// NewAdvisoryLockProvider は MySQL の接続単位アドバイザリロックを生成します。
func NewAdvisoryLockProvider(database *sqlx.DB) domainrepo.BatchLockProvider {
	return &advisoryLockProvider{db: database}
}

func (p *advisoryLockProvider) TryAcquire(ctx context.Context, name string) (domainrepo.BatchLock, bool, error) {
	conn, err := p.db.Connx(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired sql.NullInt64
	if err := conn.GetContext(ctx, &acquired, `SELECT GET_LOCK(?, 0)`, name); err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	ok, err := interpretGetLock(acquired)
	if err != nil {
		_ = conn.Close()
		return nil, false, err
	}
	if !ok {
		_ = conn.Close()
		return nil, false, nil
	}
	return &mysqlAdvisoryLock{conn: conn, name: name}, true, nil
}

type mysqlAdvisoryLock struct {
	conn *sqlx.Conn
	name string
}

func (l *mysqlAdvisoryLock) Release(ctx context.Context) error {
	defer l.conn.Close()
	var released sql.NullInt64
	if err := l.conn.GetContext(ctx, &released, `SELECT RELEASE_LOCK(?)`, l.name); err != nil {
		return err
	}
	return interpretReleaseLock(released)
}

func interpretGetLock(acquired sql.NullInt64) (bool, error) {
	if !acquired.Valid {
		return false, fmt.Errorf("GET_LOCK returned NULL")
	}
	if acquired.Int64 == 0 {
		return false, nil
	}
	return true, nil
}

func interpretReleaseLock(released sql.NullInt64) error {
	if !released.Valid || released.Int64 != 1 {
		return fmt.Errorf("RELEASE_LOCK failed")
	}
	return nil
}
