package db

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// ValidateRequiredData は必須データがデータベースに存在するかを検証します
func ValidateRequiredData(db *sqlx.DB) error {
	slog.Info("Starting database validation for required data")
	if err := checkTableHasData(db, "songs"); err != nil {
		return fmt.Errorf("songs table validation failed: %w", err)
	}
	if err := checkTableHasData(db, "charts"); err != nil {
		return fmt.Errorf("charts table validation failed: %w", err)
	}

	slog.Info("Database validation completed successfully - all required data exists")
	return nil
}

// checkTableHasData は指定されたテーブルにデータが存在するかをチェックします
func checkTableHasData(db *sqlx.DB, tableName string) error {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

	slog.Debug("Checking data existence", "table", tableName)

	if err := db.Get(&count, query); err != nil {
		return fmt.Errorf("failed to count records in table %s: %w", tableName, err)
	}

	if count == 0 {
		return fmt.Errorf("table %s has no data - application requires %s data to function properly", tableName, tableName)
	}

	slog.Info("Table validation passed", "table", tableName, "record_count", count)
	return nil
}

// GetTableStats は各テーブルのレコード数を取得します
func GetTableStats(db *sqlx.DB) (map[string]int, error) {
	tables := []string{"songs", "charts", "genres", "difficulties"}
	stats := make(map[string]int)

	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.Get(&count, query); err != nil {
			slog.Warn("Failed to get count for table", "table", table, "error", err)
			stats[table] = -1
			continue
		}
		stats[table] = count
	}

	return stats, nil
}
