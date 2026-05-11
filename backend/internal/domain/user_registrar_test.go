package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func TestUserRegistrar_Register_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()

	registrar := domain.NewUserRegistrar(repo)
	u, err := registrar.Register(email, "Alice")
	require.NoError(t, err)
	assert.Equal(t, "Alice", u.DisplayName())
	assert.Equal(t, domain.UserStatusActive, u.Status())
	assert.True(t, email.Equals(u.Email()))
	_, parseErr := uuid.Parse(u.ID().String())
	assert.NoError(t, parseErr, "generated UserID must be a valid UUID")
	assert.Empty(t, u.PullEvents(), "newly registered user must have no pending events")
	repo.AssertExpectations(t)
}

func TestUserRegistrar_Register_RejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(true, nil).Once()

	registrar := domain.NewUserRegistrar(repo)
	_, err = registrar.Register(email, "Bob")
	require.Error(t, err)
	var ve *apperrors.ValidationError
	assert.ErrorAs(t, err, &ve, "duplicate email must return ValidationError")
	repo.AssertExpectations(t)
}

func TestUserRegistrar_Register_DisplayNameBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		displayName string
		wantErr     bool
	}{
		{"empty rejected", "", true},
		{"1 char accepted", "a", false},
		{"50 chars accepted", strings.Repeat("a", 50), false},
		{"51 chars rejected", strings.Repeat("a", 51), true},
		{"50 multibyte chars accepted", strings.Repeat("あ", 50), false},
		{"51 multibyte chars rejected", strings.Repeat("あ", 51), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewUserRepository()
			// 各テストで別の email を使い、一意性チェックを通過させる
			email, err := domain.NewEmail("user-" + uuid.NewString() + "@example.com")
			require.NoError(t, err)
			// displayName が境界外でも UserRegistrar は ExistsByEmail を必ず先に呼ぶため、
			// すべてのケースで false 応答を仕込んでおく。
			repo.On("ExistsByEmail", email).Return(false, nil).Once()

			registrar := domain.NewUserRegistrar(repo)
			_, err = registrar.Register(email, tt.displayName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
