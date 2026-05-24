# .env.local を環境変数の唯一の source of truth として読み込み、子プロセス（go / docker）へ export する。
# 未作成なら .env.example からコピーする想定（-include なので存在しなくても make 自体は失敗しない）。
-include .env.local
export

.DEFAULT_GOAL := help

.PHONY: help up down run test build fmt vet tidy

help: ## 利用可能なターゲット一覧を表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

up: ## ローカル開発用 PostgreSQL を起動する
	docker compose --env-file .env.local up -d

down: ## ローカル開発用 PostgreSQL を停止する
	docker compose --env-file .env.local down

run: ## API サーバーを起動する
	go run ./cmd/api

test: ## すべてのテストを実行する
	go test ./...

build: ## API バイナリをビルドする
	go build -o bin/api ./cmd/api

fmt: ## コードを整形する
	go fmt ./...

vet: ## 静的解析を実行する
	go vet ./...

tidy: ## go.mod / go.sum を整理する
	go mod tidy
