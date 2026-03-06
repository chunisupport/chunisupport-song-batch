// Package config はアプリケーション設定の読み込みと管理機能を提供します。
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Auth は認証関連の設定を定義します。
type Auth struct {
	JWTSecret             string `json:"jwt_secret"`
	JWTExpirationHour     int    `json:"jwt_expiration_hour"`
	SessionExpirationHour int    `json:"session_expiration_hour"`
}

// Config はアプリケーション全体の設定を表します。
type Config struct {
	AppPort  int      `json:"app_port"`
	LogLevel string   `json:"log_level"`
	PwPepper string   `json:"pw_pepper"`
	Auth     Auth     `json:"auth"`
	Database Database `json:"database"`
}

// DbConfig はデータベース接続パラメータを定義します。
type DbConfig struct {
	DbName string `json:"DB_NAME"`
	DbHost string `json:"DB_HOST"`
	DbPort int    `json:"DB_PORT"`
	DbUser string `json:"DB_USER"`
	DbPass string `json:"DB_PASS"`
}

// Database はデータベース設定をラップします。
type Database struct {
	DbConfig DbConfig `json:"db_config"`
}

func loadDbConfigFromEnv() (DbConfig, error) {
	var config DbConfig
	var missing []string

	get := func(key string) string {
		value, ok := os.LookupEnv(key)
		if !ok || value == "" {
			missing = append(missing, key)
			return ""
		}
		return value
	}

	config.DbName = get("DB_NAME")
	config.DbHost = get("DB_HOST")
	portStr := get("DB_PORT")
	config.DbUser = get("DB_USER")
	config.DbPass = get("DB_PASS")

	if len(missing) > 0 {
		return config, fmt.Errorf("missing required DB environment variables: %s", strings.Join(missing, ", "))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return config, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	config.DbPort = port

	return config, nil
}

// LoadConfigFromEnv は環境変数からアプリケーション設定を読み込みます。
func LoadConfigFromEnv() (Config, error) {
	var config Config

	if pepper, ok := os.LookupEnv("PW_PEPPER"); ok && pepper != "" {
		config.PwPepper = pepper
	}

	if config.PwPepper == "" {
		return config, errors.New("missing PW_PEPPER environment variable")
	}

	dbConfig, err := loadDbConfigFromEnv()
	if err != nil {
		return config, err
	}
	config.Database.DbConfig = dbConfig

	return config, nil
}
