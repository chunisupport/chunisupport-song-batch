package db

import (
	"database/sql"
	"testing"
)

func TestInterpretGetLock(t *testing.T) {
	t.Parallel()

	t.Run("acquired", func(t *testing.T) {
		ok, err := interpretGetLock(sql.NullInt64{Int64: 1, Valid: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected lock to be acquired")
		}
	})

	t.Run("not acquired", func(t *testing.T) {
		ok, err := interpretGetLock(sql.NullInt64{Int64: 0, Valid: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected lock not to be acquired")
		}
	})

	t.Run("null", func(t *testing.T) {
		ok, err := interpretGetLock(sql.NullInt64{})
		if err == nil {
			t.Fatal("expected error for NULL")
		}
		if ok {
			t.Fatal("expected lock not to be acquired")
		}
	})
}

func TestInterpretReleaseLock(t *testing.T) {
	t.Parallel()

	if err := interpretReleaseLock(sql.NullInt64{Int64: 1, Valid: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := interpretReleaseLock(sql.NullInt64{Int64: 0, Valid: true}); err == nil {
		t.Fatal("expected error when RELEASE_LOCK returns 0")
	}
	if err := interpretReleaseLock(sql.NullInt64{}); err == nil {
		t.Fatal("expected error when RELEASE_LOCK returns NULL")
	}
}
