// Package config はアプリが必要とする環境変数の読み込みを 1 箇所に集約する。
// 値の source of truth は .env.local（make 経由で export される）。
package config

import (
	"fmt"
	"os"
)

type Config struct {
	DB         DBConfig
	Port       string
	LocalesDir string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.Name,
	)
}

// Load は必須環境変数を構造体に読み込む。デフォルト値は持たず、未設定があれば
// どの変数が欠けているかを列挙して fail-fast する（暗黙のデフォルトで誤接続しないため）。
func Load() (*Config, error) {
	var missing []string
	get := func(key string) string {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
		missing = append(missing, key)
		return ""
	}

	cfg := &Config{
		DB: DBConfig{
			Host:     get("POSTGRES_HOST"),
			Port:     get("POSTGRES_PORT"),
			User:     get("POSTGRES_USER"),
			Password: get("POSTGRES_PASSWORD"),
			Name:     get("POSTGRES_DB"),
		},
		Port:       get("PORT"),
		LocalesDir: get("LOCALES_DIR"),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars %v (set them in .env.local and run via `make run`/`make test`)", missing)
	}
	return cfg, nil
}
