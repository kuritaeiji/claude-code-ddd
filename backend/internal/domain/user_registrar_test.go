package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(nil).Once()

	registrar := domain.NewUserRegistrar(repo)
	u, err := registrar.Register("user@example.com", "Alice")
	require.NoError(t, err)
	assert.Equal(t, "Alice", u.DisplayName())
	assert.Equal(t, domain.UserStatusActive, u.Status())
	assert.True(t, email.Equals(u.Email()))
	_, parseErr := uuid.Parse(u.ID().String())
	assert.NoError(t, parseErr, "generated UserID must be a valid UUID")
	assert.Empty(t, u.PullEvents(), "newly registered user must have no pending events")
	repo.AssertExpectations(t)
}

func TestUserRegistrar_Register_RejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	registrar := domain.NewUserRegistrar(repo)

	_, err := registrar.Register("not-an-email", "Alice")

	require.Error(t, err)
	var ve *apperrors.ValidationError
	assert.ErrorAs(t, err, &ve, "invalid email must return ValidationError")
	repo.AssertNotCalled(t, "ExistsByEmail")
	repo.AssertNotCalled(t, "Insert")
}

func TestUserRegistrar_Register_RejectsDuplicateEmail_Precheck(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(true, nil).Once()

	registrar := domain.NewUserRegistrar(repo)
	_, err = registrar.Register("user@example.com", "Bob")
	require.Error(t, err)
	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve, "duplicate email must return ValidationError")
	assert.Equal(t, "validation.email.already_registered", ve.MessageID)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "Insert")
}

// レース時の最終防衛線：事前チェックを通過したが Insert で UNIQUE 制約に当たったケース。
// Repository が ErrEmailDuplicate を返し、Registrar が ValidationError に翻訳する経路を検証する。
func TestUserRegistrar_Register_TranslatesInsertDuplicate(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(domain.ErrEmailDuplicate).Once()

	registrar := domain.NewUserRegistrar(repo)
	_, err = registrar.Register("user@example.com", "Carol")

	require.Error(t, err)
	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "validation.email.already_registered", ve.MessageID)
}

func TestUserRegistrar_Register_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	insertErr := errors.New("db down")
	repo.On("ExistsByEmail", email).Return(false, nil).Once()
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(insertErr).Once()

	registrar := domain.NewUserRegistrar(repo)
	_, err = registrar.Register("user@example.com", "Dave")

	require.Error(t, err)
	assert.ErrorIs(t, err, insertErr)
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
			emailStr := "user-" + uuid.NewString() + "@example.com"
			email, err := domain.NewEmail(emailStr)
			require.NoError(t, err)
			// displayName が境界外でも UserRegistrar は ExistsByEmail を先に呼ぶため、
			// すべてのケースで false 応答を仕込む。Insert は displayName 通過時のみ呼ばれる。
			repo.On("ExistsByEmail", email).Return(false, nil).Once()
			if !tt.wantErr {
				repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(nil).Once()
			}

			registrar := domain.NewUserRegistrar(repo)
			_, err = registrar.Register(emailStr, tt.displayName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
