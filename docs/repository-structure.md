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
│   │   └── event.go             # ドメインイベント定義・EventBus IF・EventHandler IF
│   ├── usecase/
│   │   ├── command/             # CQRSコマンド（書き込み）・EventHandler実装
│   │   └── query/               # CQRSクエリ（読み込み）
│   ├── controller/
│   │   └── ...                  # net/http ハンドラー
│   └── infrastructure/
│       ├── repository/          # gormリポジトリ実装
│       └── event/               # インメモリイベントバス実装
├── pkg/
│   └── errors/                  # ドメインエラー種別定義（ValidationError・NotFoundError等）
├── docker-compose.yml           # ローカル開発用PostgreSQL起動
├── .env.example                 # 環境変数テンプレート（コミット対象・ダミー値）
└── .env.local                   # 実際の認証情報（.gitignore対象・コミット禁止）
```

---

## ディレクトリの意図

| ディレクトリ                                  | 層               | 意図                                                                                                                                                                               |
| --------------------------------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backend/cmd/api/`                            | —                | アプリケーションのエントリーポイント。依存オブジェクトの組み立て（DI）とサーバー起動のみを行う。イベントハンドラの登録もここで実施する                                             |
| `backend/internal/domain/`                    | Domain層         | ビジネスの核。エンティティ・値オブジェクト・リポジトリインターフェース・ドメインサービス・ドメインイベントを定義する。外部ライブラリへの依存を持たない                             |
| `backend/internal/domain/task.go`             | Domain層         | タスク集約ルート・TaskStatus・Priority・DueDate・TaskID 値オブジェクト・TaskRepository インターフェースを集約して定義する                                                          |
| `backend/internal/domain/user.go`             | Domain層         | ユーザー集約ルート・Email・UserStatus・UserID 値オブジェクト・UserRepository インターフェースを集約して定義する                                                                    |
| `backend/internal/domain/event.go`            | Domain層         | ドメインイベント型（UserDeactivated 等）・EventBus インターフェース・EventHandler インターフェースを定義する                                                                       |
| `backend/internal/usecase/command/`           | Usecase層        | 書き込み系ユースケース（T-01〜T-03、U-01〜U-02）を実装する。ドメインモデルを通じてビジネスルールを適用し、リポジトリに永続化する。ドメインイベントの EventHandler 実装もここに置く |
| `backend/internal/usecase/query/`             | Usecase層        | 読み込み系クエリ（Q-01〜Q-02）を実装する。ドメインモデルを経由せず、DB から直接クエリ専用 DTO に詰める（JOIN 最適化のため）                                                        |
| `backend/internal/controller/`                | Controller層     | net/http ハンドラー。リクエストを xxxRequest に bind し、xxxParams に変換して Usecase を呼び出す。Usecase から受け取った xxxDTO を xxxResponse に変換してレスポンスを返す          |
| `backend/internal/infrastructure/repository/` | Infrastructure層 | gorm を使った TaskRepository・UserRepository の実装。Domain 層で定義されたリポジトリインターフェースを満たす                                                                       |
| `backend/internal/infrastructure/event/`      | Infrastructure層 | インメモリ EventBus の実装。Domain 層で定義された EventBus インターフェースを満たす。登録された EventHandler を順に呼び出す                                                        |
| `backend/pkg/errors/`                         | —                | ドメインエラー種別（ValidationError・NotFoundError・DomainRuleError 等）を定義する。internal 外から参照できるよう pkg に配置する                                                   |
