package db

import (
	"fmt"

	"github.com/chunisupport/chunisupport-song-batch/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Connect はデータベースに接続します
func Connect(dbConfig config.DbConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		dbConfig.DbUser,
		dbConfig.DbPass,
		dbConfig.DbHost,
		dbConfig.DbPort,
		dbConfig.DbName,
	)

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("Failed to open MySQL database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("Failed to ping MySQL database: %w", err)
	}

	return db, nil
}
