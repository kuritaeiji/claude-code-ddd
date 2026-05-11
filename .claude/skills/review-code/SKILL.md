---
name: review-code
description: backend/ 配下の Go 実装コードをレビューする。docs/ と照合してビジネスルール・オニオンアーキテクチャの依存方向・DDD設計・CQRS の妥当性を横断チェックし、問題点を報告する。
disable-model-invocation: true
---

# コードレビュー

backend/ 配下の Go 実装コードをレビューしてください。以下の手順を順番に実行してください。

---

## ステップ 1: source of truth ドキュメントの読込

Read ツールで以下を全て読み、ビジネスルール・アーキテクチャ方針・ディレクトリ意図を把握してください。

- docs/production-requirements.md
- docs/domain.md
- docs/usecase.md
- docs/architecture.md
- docs/repository-structure.md
- docs/schema.md（存在しない場合はスキップ）

---

## ステップ 2: 実装コードの読込

Bash の `find backend -type f -name '*.go'` で対象ファイルを列挙し、Read ツールで全ファイルを読んでください。

対象ディレクトリ:

- `backend/cmd/api/`
- `backend/internal/domain/`
- `backend/internal/usecase/command/`
- `backend/internal/usecase/query/`
- `backend/internal/controller/`
- `backend/internal/infrastructure/repository/`
- `backend/internal/infrastructure/event/`
- `backend/pkg/errors/`

---

## ステップ 3: レビュー観点

### 1. ビジネスルール・アーキテクチャ方針の遵守

- `docs/domain.md` のビジネスルール（BR-T-XX, BR-U-XX）が Domain 層に正しく実装されているか
- `docs/usecase.md` のユースケース・APIエンドポイントが過不足なく実装されているか
- `docs/architecture.md` のエラーハンドリング方針が遵守されているか
- バリデーションが Domain 層のみで実施されているか（Usecase / Controller に漏れていないか）

### 2. オニオンアーキテクチャの依存方向

import 文を確認し、以下の違反を検出する:

- Domain が Usecase / Controller / Infrastructure を import していないか
- Infrastructure が Usecase を import していないか（EventHandler は Domain インターフェース経由で接続するのが正）
- Domain が外部ライブラリ（gorm, net/http など）に依存していないか
- `mocks/` パッケージがプロダクションコードから参照されていないか

### 3. DDD 設計の妥当性

- 集約境界は適切か（集約をまたぐ参照は ID のみか）
- 値オブジェクトとエンティティの分類は正しいか（不変・等価性で識別すべきものが値オブジェクトになっているか）
- ドメインサービス（`UserRegistrar` 等）の責務が妥当か（ドメインルールの強制になっているか、単なるリポジトリラッパーになっていないか）
- ドメインイベントの発行タイミング・型・副作用が `docs/domain.md` の定義と一致しているか
- 集約ルートを介さず内部状態を直接書き換える経路（公開フィールド等）が無いか

### 4. CQRS の適用

- コマンド（Write）はドメインモデル経由でビジネスルールを通っているか
- クエリ（Read）はドメインモデルを経由せず、専用 DTO で PostgreSQL から直接読んでいるか
- ドメインイベントの `EventHandler` 実装が Usecase 層に置かれているか
- `EventBus` の DI が `cmd/api/main.go` で行われているか

### 5. テストの妥当性

- Domain 層のビジネスルールに対するユニットテストが存在するか
- モック実装は `backend/internal/domain/mocks/` 配下に testify/mock ベースで配置されているか（テストファイル内に手書き fake が再導入されていないか）
- Infrastructure 層のリポジトリ・EventBus に対するユニットテストが存在するか（該当層が実装済みの場合）
- テストが `testify/assert` `testify/require` を使っているか

### 6. コード品質

- コメントは Why のみで、What コメントが混入していないか
- 未使用の export / 未使用の依存がないか
- 過剰な抽象化や未使用のインターフェースがないか

---

## 出力形式

以下の形式で報告してください。問題がない観点は省略せず「問題なし」と明記してください。

```
## コードレビュー結果

### 重大な問題（修正必須）
- [path/to/file.go:LINE] 問題の説明 → 修正案

### 軽微な問題・改善提案
- [path/to/file.go:LINE] 問題の説明 → 修正案

### 良い点
- 適切に設計・実装されている点を列挙

### サマリー
全体の評価と次のアクション（1〜3文）
```
