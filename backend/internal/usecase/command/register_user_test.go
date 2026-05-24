package command_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
)

func newRegisterUserCommand(repo *mocks.UserRepository) *command.RegisterUserCommand {
	return command.NewRegisterUserCommand(domain.NewUserRegistrar(repo))
}

// Usecase 層は Registrar への委譲と DTO 変換のみを担うため、ここでは入力検証や
// 一意性違反などのエッジケースは網羅しない（それらは UserRegistrar のテストで担保）。
// このパッケージのテストは「成功時に DTO が正しく組み立てられる」「Registrar のエラーが
// そのまま伝播する」の 2 経路に絞る。

func TestRegisterUser_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repo.On("ExistsByEmail", email).Return(false, nil).Once()
	repo.On("Insert", mock.AnythingOfType("*domain.User")).Return(nil).Once()

	cmd := newRegisterUserCommand(repo)
	dto, err := cmd.Execute(command.RegisterUserParams{Email: "user@example.com", DisplayName: "Alice"})
	require.NoError(t, err)

	assert.Equal(t, "user@example.com", dto.Email)
	assert.Equal(t, "Alice", dto.DisplayName)
	assert.Equal(t, string(domain.UserStatusActive), dto.Status)
	_, parseErr := uuid.Parse(dto.ID)
	assert.NoError(t, parseErr, "DTO.ID must be a valid UUID")
	repo.AssertExpectations(t)
}

func TestRegisterUser_PropagatesRegistrarError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	repoErr := errors.New("db down")
	repo.On("ExistsByEmail", email).Return(false, repoErr).Once()

	cmd := newRegisterUserCommand(repo)
	_, err = cmd.Execute(command.RegisterUserParams{Email: "user@example.com", DisplayName: "Alice"})

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}
