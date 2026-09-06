package repository

import "context"

// BatchLock はバッチ実行全体の多重起動を防止します。
type BatchLock interface {
	Release(ctx context.Context) error
}

// BatchLockProvider は接続単位のアドバイザリロックを取得します。
type BatchLockProvider interface {
	TryAcquire(ctx context.Context, name string) (BatchLock, bool, error)
}
