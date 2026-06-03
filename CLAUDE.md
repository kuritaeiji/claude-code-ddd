## プロジェクト概要

**タスク管理バックエンド API**（学習目的）

DDD・オニオンアーキテクチャ・CQRS を Go で実践するための学習プロジェクト。
認証なし、インメモリイベントバス、ローカル開発のみを前提とする。

### 技術スタック

| 要素   | 採用技術   |
| ------ | ---------- |
| 言語   | Go         |
| HTTP   | net/http   |
| ORM    | gorm       |
| DB     | PostgreSQL |
| テスト | testify    |

### アーキテクチャの要点

- **オニオンアーキテクチャ**: 依存方向は Domain ← Usecase ← Controller、Domain ← Infrastructure のみ許可
- **CQRS**: クエリ（Read）はドメインモデルを経由せず、専用 DTO で DB から直接読み取る
- **ドメインイベント**: インメモリ EventBus。EventHandler は Usecase 層に実装し、main.go で DI する
- **バリデーション**: すべてドメイン層で実施する
- **環境変数**: 値は `.env.local` / `.env.example` を唯一の source of truth とし、Go コードにデフォルト値を持たせない（未設定なら fail-fast）。`Makefile` が `.env.local` をロードして `go` / `docker compose` をラップする

### 開発ワークフロー

機能を実装するときは `/implement <機能名>` スキルを使う。
スキルが以下を自動実行する：ドキュメント理解 → パターン調査 → 計画作成（`.steering/`）→ 実装ループ → レビュー → テスト

#### make コマンド一覧（backend/ で実行）

| コマンド     | 用途                                         |
| ------------ | -------------------------------------------- |
| `make up`    | ローカル開発用 PostgreSQL を起動             |
| `make down`  | ローカル開発用 PostgreSQL を停止             |
| `make run`   | API サーバーを起動                           |
| `make test`  | すべてのテストを実行                         |
| `make build` | API バイナリをビルド（`./cmd/api` のみ）     |
| `make vet`   | 全パッケージの静的解析・コンパイルエラー確認 |
| `make fmt`   | コードを整形                                 |
| `make tidy`  | go.mod / go.sum を整理                       |

**`go build`・`go test`・`go vet` などの `go` コマンドを直接実行しない。必ず `make` 経由で実行すること。**
環境変数の注入は Makefile が `.env.local` をロードして行うため、直接実行すると環境変数が欠けて fail-fast する。

#### テスト実行

`infrastructure/repository` の統合テストは常に実 PostgreSQL に接続する。**`make test` の前には必ず `make up` を実行すること。**

```
make up && make test
```

---

## ディレクトリ構造

```
.
├── docs/                        # 永続的ドキュメント（設計の source of truth）
│   ├── production-requirements.md  # プロダクト概要・スコープ・開発フェーズ
│   ├── domain.md                   # ドメインモデル・ビジネスルール・ドメインイベント
│   ├── usecase.md                  # ユースケース一覧・APIエンドポイント
│   ├── infrastructure.md           # AWSインフラ構成・ECS/RDS・CI/CD・Terraform
│   ├── schema.md                   # ER図・テーブル定義・ドメインモデルとのマッピング
│   └── repository-structure.md     # ディレクトリ構成とその意図
├── backend/
│   ├── cmd/api/main.go             # エントリーポイント・DI・サーバー起動
│   ├── internal/
│   │   ├── domain/                 # Domain層: エンティティ・値オブジェクト・リポジトリIF・ドメインイベント
│   │   │   ├── task.go
│   │   │   ├── user.go
│   │   │   └── event.go            # EventBus IF・EventHandler IF・DomainEvent IF（個別イベント型は各集約に定義）
│   │   ├── usecase/
│   │   │   ├── command/            # CQRSコマンド（書き込み）・EventHandler実装
│   │   │   └── query/              # CQRSクエリ（読み込み）・DB直接読み取り
│   │   ├── controller/             # net/http ハンドラー（xxxRequest → xxxParams → xxxDTO → xxxResponse）
│   │   └── infrastructure/
│   │       ├── repository/         # gorm リポジトリ実装
│   │       ├── event/              # インメモリ EventBus 実装
│   │       └── config/             # 環境変数の読み込み集約（Config 構造体・DSN 組み立て）
│   └── pkg/errors/                 # ドメインエラー種別定義
├── .steering/                   # 実装計画（/implement スキルが自動生成）
│   └── YYYYMMDD-<機能名>/
│       ├── requirements.md         # 要件・ビジネスルール
│       ├── approach.md             # 実装方針・作成ファイル一覧
│       └── tasks.md                # タスクリスト（[ ]/[x]）
└── .claude/
    └── skills/
        ├── implement/SKILL.md      # /implement スキル（実装ワークフロー自動実行）
        └── review-docs/SKILL.md    # /review-docs スキル（ドキュメントレビュー）
```
