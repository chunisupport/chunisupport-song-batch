// Package config はアプリケーション設定の読み込みと管理機能を提供します。
package config

import (
	"encoding/json"
	"errors"
	"os"

	"chunisupport-song-batch/internal/info"
)

// Auth は認証関連の設定を定義します。
type Auth struct {
	JWTSecret             string `json:"jwt_secret"`
	JWTExpirationHour     int    `json:"jwt_expiration_hour"`
	SessionExpirationHour int    `json:"session_expiration_hour"`
}

// Config はアプリケーション全体の設定を表します。
type Config struct {
	AppPort     int               `json:"app_port"`
	LogLevel    string            `json:"log_level"`
	PwPepper    string            `json:"pw_pepper"`
	Auth        Auth              `json:"auth"`
	Database    Database          `json:"database"`
	Datasources []DatasourceEntry `json:"datasources"`
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

// DatasourceEntry はバッチインポート用のデータソースを定義します。
type DatasourceEntry struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// LoadConfig は指定された環境に基づいてJSONファイルからアプリケーション設定を読み込みます
func LoadConfig(env string) (Config, error) {
	var config Config

	configFile, err := os.Open(info.ConfigDir + env + ".settings.json")
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		return config, err
	}

	if pepper, ok := os.LookupEnv("PW_PEPPER"); ok && pepper != "" {
		config.PwPepper = pepper
	}

	if config.PwPepper == "" {
		return config, errors.New("missing PW_PEPPER environment variable")
	}

	return config, nil
}
