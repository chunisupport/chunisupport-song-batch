package repository

import (
	"context"
	"errors"
	"testing"

	domainrepo "chunisupport-song-batch/internal/domain/repository"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// TestTransactional_Commit はトランザクションが正常にコミットされることを確認
func TestTransactional_Commit(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	// トランザクション内で INSERT
	err = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "test_value")
		return err
	})
	if err != nil {
		t.Fatalf("Transactional returned error: %v", err)
	}

	// 検証: データがコミットされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	var value string
	err = db.Get(&value, "SELECT value FROM test_table WHERE id = 1")
	if err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if value != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", value)
	}
}

// TestTransactional_Rollback はエラー時にロールバックされることを確認
func TestTransactional_Rollback(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	expectedErr := errors.New("intentional error")

	// トランザクション内で INSERT した後にエラーを返す
	err = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "test_value")
		if err != nil {
			return err
		}
		return expectedErr
	})

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to be '%v', got '%v'", expectedErr, err)
	}

	// 検証: データがロールバックされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows (rolled back), got %d", count)
	}
}

// TestTransactional_PanicRecovery はパニック時にロールバックされ、パニックが再送出されることを確認
func TestTransactional_PanicRecovery(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	// パニックがキャッチされることを確認
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic to be re-thrown")
		}
	}()

	// トランザクション内でパニック
	_ = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "test_value")
		if err != nil {
			return err
		}
		panic("intentional panic")
	})

	// パニックで到達しないはず
	t.Error("should not reach here after panic")
}

// TestTransactional_PanicRecovery_DataRolledBack はパニック時にデータがロールバックされることを確認
func TestTransactional_PanicRecovery_DataRolledBack(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	// パニックをキャッチ
	func() {
		defer func() {
			_ = recover() // パニックを吸収
		}()

		_ = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "test_value")
			if err != nil {
				return err
			}
			panic("intentional panic")
		})
	}()

	// 検証: データがロールバックされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows (rolled back after panic), got %d", count)
	}
}

// TestTransactional_MultipleOperations はトランザクション内で複数の操作が正しく処理されることを確認
func TestTransactional_MultipleOperations(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	// トランザクション内で複数の INSERT
	err = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		for i := 1; i <= 5; i++ {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "value_"+string(rune('0'+i)))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transactional returned error: %v", err)
	}

	// 検証: すべてのデータがコミットされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 rows, got %d", count)
	}
}

// TestTransactional_NestedError は複数操作の途中でエラーが発生した場合にすべてロールバックされることを確認
func TestTransactional_NestedError(t *testing.T) {
	ctx := context.Background()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// テーブル作成
	_, err = db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	tm := NewTransactionManager(db)

	// トランザクション内で 3 件 INSERT した後にエラー
	err = tm.Transactional(ctx, func(tx domainrepo.ExtendedDBExecutor) error {
		for i := 1; i <= 3; i++ {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_table (value) VALUES (?)", "value_"+string(rune('0'+i)))
			if err != nil {
				return err
			}
		}
		return errors.New("error after inserts")
	})

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// 検証: すべてのデータがロールバックされている
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_table")
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows (all rolled back), got %d", count)
	}
}

// TestNewTransactionManager は TransactionManager が正しく生成されることを確認
func TestNewTransactionManager(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	tm := NewTransactionManager(db)

	if tm == nil {
		t.Fatal("expected non-nil TransactionManager")
	}
}
