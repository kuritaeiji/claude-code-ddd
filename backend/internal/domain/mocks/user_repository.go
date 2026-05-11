// Package mocks は Domain 層のインターフェースに対する testify/mock ベースのモック実装を提供する。
// テストコードから import して使うことを目的とし、プロダクションコードからの参照は禁止する。
package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// UserRepository インターフェースの実装漏れをビルド時に検出するためのアサーション。
var _ domain.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	mock.Mock
}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (m *UserRepository) Save(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *UserRepository) FindByID(id domain.UserID) (*domain.User, error) {
	args := m.Called(id)
	user, _ := args.Get(0).(*domain.User)
	return user, args.Error(1)
}

func (m *UserRepository) FindByEmail(email domain.Email) (*domain.User, error) {
	args := m.Called(email)
	user, _ := args.Get(0).(*domain.User)
	return user, args.Error(1)
}

func (m *UserRepository) ExistsByEmail(email domain.Email) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}
