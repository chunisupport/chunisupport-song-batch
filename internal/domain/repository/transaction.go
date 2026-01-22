package repository

import "context"

// TransactionManager はトランザクション処理を抽象化します。
// 実装はインフラ層に配置され、ドメイン層がDB実装に依存しないようにします。
type TransactionManager interface {
	// Transactional は処理 f をトランザクション内で実行します。
	// f がエラーを返した場合はロールバック、正常終了した場合はコミットします。
	Transactional(ctx context.Context, f func(tx ExtendedDBExecutor) error) error
}
