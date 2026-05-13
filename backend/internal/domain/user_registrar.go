package domain

import (
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// UserRegistrar はユーザー登録のドメインサービス。
// BR-U-01（email 一意性）の事前チェックと User の生成を domain 層に閉じ込め、
// ユースケース層からは Register 経由でのみ User を作れるようにする。
// レース時の最終的な一意性は DB の UNIQUE 制約で保証される前提。
type UserRegistrar struct {
	repo UserRepository
}

func NewUserRegistrar(repo UserRepository) *UserRegistrar {
	return &UserRegistrar{repo: repo}
}

// Register は email 一意性を検証して新規ユーザーを生成する。
// 永続化（UserRepository.Save）は呼び出し側の責務。
func (r *UserRegistrar) Register(email Email, displayName string) (*User, error) {
	exists, err := r.repo.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.NewValidationError("validation.email.already_registered", nil)
	}
	return newUser(email, displayName)
}
