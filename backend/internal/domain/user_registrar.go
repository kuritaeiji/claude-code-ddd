package domain

import (
	"errors"

	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// UserRegistrar はユーザー登録のドメインサービス。
// Email 値オブジェクトの構築（形式検証）・BR-U-01（email 一意性）の事前チェック・newUser（BR-U-02: displayName 検証）
// ・永続化・UNIQUE 制約違反の業務ルール違反への翻訳までを一連の責務として担う。
// ユーザー登録に関するすべてのドメインルールがこのサービス経由でしか発火しないため、
// 外部からユーザーを生成する正規パスは Register のみとなる（newUser は同パッケージ内で非公開）。
type UserRegistrar struct {
	repo UserRepository
}

func NewUserRegistrar(repo UserRepository) *UserRegistrar {
	return &UserRegistrar{repo: repo}
}

// Register は文字列の email / displayName を受け取り、検証・一意性確認・永続化までを一括で行う。
// 失敗時は以下のいずれかを返す:
//   - email 形式不正 → ValidationError（NewEmail 由来）
//   - displayName 長さ違反 → ValidationError（newUser 由来）
//   - email 重複（事前チェック / レース時の Insert 後判定の両方）→ ValidationError "validation.email.already_registered"
//   - リポジトリ I/O エラー → そのまま伝播
func (r *UserRegistrar) Register(emailValue, displayName string) (*User, error) {
	email, err := NewEmail(emailValue)
	if err != nil {
		return nil, err
	}

	exists, err := r.repo.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.NewValidationError("validation.email.already_registered", nil)
	}

	user, err := newUser(email, displayName)
	if err != nil {
		return nil, err
	}

	if err := r.repo.Insert(user); err != nil {
		if errors.Is(err, ErrEmailDuplicate) {
			return nil, apperrors.NewValidationError("validation.email.already_registered", nil)
		}
		return nil, err
	}
	return user, nil
}
