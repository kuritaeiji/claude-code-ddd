# ディレクトリ構成

```
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── task.go              # タスク集約（エンティティ・値オブジェクト・リポジトリIF）
│   │   ├── user.go              # ユーザー集約（エンティティ・値オブジェクト・リポジトリIF）
│   │   ├── event.go             # ドメインイベント定義・EventBus IF・EventHandler IF
│   │   └── mocks/               # testify/mock ベースのモック実装（テストからのみ参照）
│   ├── usecase/
│   │   ├── command/             # CQRSコマンド（書き込み）・EventHandler実装
│   │   └── query/               # CQRSクエリ（読み込み）
│   ├── controller/
│   │   └── ...                  # net/http ハンドラー
│   └── infrastructure/
│       ├── repository/          # gormリポジトリ実装
│       ├── event/               # インメモリイベントバス実装
│       ├── i18n/                # go-i18n Bundle・Translator 実装（英語固定 / Accept-Language ベース）
│       └── config/              # 環境変数の読み込みを集約（Config 構造体・DSN 組み立て）
├── locales/                     # 翻訳ファイル（YAML）
│   ├── en.yaml                  # 英語（デフォルト言語）
│   └── ja.yaml                  # 日本語
├── pkg/
│   └── errors/                  # ドメインエラー種別定義 + Translator インターフェース
├── Makefile                     # .env.local をロードして go / docker をラップする開発タスク
├── docker-compose.yml           # ローカル開発用PostgreSQL起動
├── .env.example                 # 全環境変数のテンプレート（コミット対象・ダミー値）
└── .env.local                   # 全環境変数の実値（.gitignore対象・コミット禁止）
```

---

## ディレクトリの意図

| ディレクトリ                                  | 層                   | 意図                                                                                                                                                                                                                                                                                                                      |
| --------------------------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backend/cmd/api/`                            | —                    | アプリケーションのエントリーポイント。依存オブジェクトの組み立て（DI）とサーバー起動のみを行う。イベントハンドラの登録もここで実施する                                                                                                                                                                                    |
| `backend/internal/domain/`                    | Domain層             | ビジネスの核。エンティティ・値オブジェクト・リポジトリインターフェース・ドメインサービス・ドメインイベントを定義する。外部ライブラリへの依存を持たない                                                                                                                                                                    |
| `backend/internal/domain/task.go`             | Domain層             | タスク集約ルート・TaskStatus・Priority・DueDate・TaskID 値オブジェクト・TaskRepository インターフェースを集約して定義する                                                                                                                                                                                                 |
| `backend/internal/domain/user.go`             | Domain層             | ユーザー集約ルート・Email・UserStatus・UserID 値オブジェクト・UserRepository インターフェースを集約して定義する                                                                                                                                                                                                           |
| `backend/internal/domain/event.go`            | Domain層             | ドメインイベント型（UserDeactivated 等）・EventBus インターフェース・EventHandler インターフェースを定義する                                                                                                                                                                                                              |
| `backend/internal/domain/mocks/`              | Domain層（テスト用） | Domain 層で定義したインターフェース（`UserRepository`・`TaskRepository`・`EventBus` 等）の testify/mock ベースのモック実装を集約する。テストコードからのみ import し、プロダクションコードからは参照しない。新しい Domain インターフェースを追加したら、必要に応じてここに対応するモックを追加する                        |
| `backend/internal/usecase/command/`           | Usecase層            | 書き込み系ユースケース（T-01〜T-03、U-01〜U-02）を実装する。ドメインモデルを通じてビジネスルールを適用し、リポジトリに永続化する。ドメインイベントの EventHandler 実装もここに置く                                                                                                                                        |
| `backend/internal/usecase/query/`             | Usecase層            | 読み込み系クエリ（Q-01〜Q-02）を実装する。ドメインモデルを経由せず、DB から直接クエリ専用 DTO に詰める（JOIN 最適化のため）                                                                                                                                                                                               |
| `backend/internal/controller/`                | Controller層         | net/http ハンドラー。リクエストを xxxRequest に bind し、xxxParams に変換して Usecase を呼び出す。Usecase から受け取った xxxDTO を xxxResponse に変換してレスポンスを返す                                                                                                                                                 |
| `backend/internal/infrastructure/repository/` | Infrastructure層     | gorm を使った TaskRepository・UserRepository の実装。Domain 層で定義されたリポジトリインターフェースを満たす                                                                                                                                                                                                              |
| `backend/internal/infrastructure/event/`      | Infrastructure層     | インメモリ EventBus の実装。Domain 層で定義された EventBus インターフェースを満たす。登録された EventHandler を順に呼び出す                                                                                                                                                                                               |
| `backend/internal/infrastructure/i18n/`       | Infrastructure層     | `github.com/nicksnyder/go-i18n/v2/i18n` の `*i18n.Bundle` 構築と `errors.Translator` インターフェース実装を集約する。`pkg/errors` に注入する英語固定 Translator と、Controller 層のミドルウェアが `Accept-Language` から都度作るリクエスト用 Translator の 2 種類を提供する                                               |
| `backend/internal/infrastructure/config/`     | Infrastructure層     | 環境変数（`POSTGRES_*`・`PORT`・`LOCALES_DIR`）の読み込みを単一の窓口に集約する。`map` ではなく named フィールドの `Config`／`DBConfig` 構造体にロードし、`DBConfig.DSN()` で gorm 用 DSN を組み立てる。必須変数が欠けたら列挙して fail-fast する。`cmd/api/main.go` と integration テストの両方が `config.Load()` を使う |
| `backend/locales/`                            | —                    | go-i18n 用の翻訳ファイル（`en.yaml`・`ja.yaml`）。ドメインエラーの `MessageID` をネストキーで階層化して記述する。アプリ起動時に Infrastructure 層の i18n パッケージが読み込む                                                                                                                                             |
| `backend/pkg/errors/`                         | —                    | ドメインエラー種別（ValidationError・NotFoundError・DomainRuleError 等）と、`Error()` から参照される `Translator` インターフェース・パッケージレベルのデフォルト Translator 注入関数（`SetDefaultTranslator`）を定義する。internal 外から参照できるよう pkg に配置する。go-i18n には依存しない                            |
| `backend/Makefile`                            | —                    | `.env.local` を読み込んで環境変数を export し、`go run`・`go test`・`docker compose` をラップする。`make help` でターゲット一覧を表示。環境変数の source of truth を `.env.local` に一元化し、Go コードにデフォルト値を持たせない方針を支える                                                                             |
