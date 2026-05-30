package command_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func newActiveUser(t *testing.T, id domain.UserID, emailValue string) *domain.User {
	t.Helper()
	email, err := domain.NewEmail(emailValue)
	require.NoError(t, err)
	return domain.ReconstructUser(id.String(), email.String(), "Alice", domain.UserStatusActive)
}

func TestDeactivateUser_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	repo := mocks.NewUserRepository()
	bus := mocks.NewEventBus()
	repo.On("FindByID", id).Return(newActiveUser(t, id, "user@example.com"), nil).Once()
	repo.On("Update", mock.AnythingOfType("*domain.User")).Return(nil).Once()
	bus.On("Publish", mock.AnythingOfType("domain.UserDeactivated")).Return(nil).Once()

	cmd := command.NewDeactivateUserCommand(repo, bus)
	dto, err := cmd.Execute(command.DeactivateUserParams{ID: id.String()})
	require.NoError(t, err)

	assert.Equal(t, id.String(), dto.ID)
	assert.Equal(t, "user@example.com", dto.Email)
	assert.Equal(t, string(domain.UserStatusInactive), dto.Status)
	repo.AssertExpectations(t)
	bus.AssertExpectations(t)
}

func TestDeactivateUser_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewUserRepository()
	bus := mocks.NewEventBus()

	cmd := command.NewDeactivateUserCommand(repo, bus)
	_, err := cmd.Execute(command.DeactivateUserParams{ID: "not-a-uuid"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	repo.AssertNotCalled(t, "FindByID")
}

func TestDeactivateUser_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	repo := mocks.NewUserRepository()
	bus := mocks.NewEventBus()
	repo.On("FindByID", id).Return(nil, domain.ErrUserNotFound).Once()

	cmd := command.NewDeactivateUserCommand(repo, bus)
	_, err := cmd.Execute(command.DeactivateUserParams{ID: id.String()})

	var nfe *apperrors.NotFoundError
	require.ErrorAs(t, err, &nfe)
	repo.AssertNotCalled(t, "Update")
	bus.AssertNotCalled(t, "Publish")
}

func TestDeactivateUser_AlreadyInactive(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	inactive := domain.ReconstructUser(id.String(), email.String(), "Alice", domain.UserStatusInactive)

	repo := mocks.NewUserRepository()
	bus := mocks.NewEventBus()
	repo.On("FindByID", id).Return(inactive, nil).Once()

	cmd := command.NewDeactivateUserCommand(repo, bus)
	_, err = cmd.Execute(command.DeactivateUserParams{ID: id.String()})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	repo.AssertNotCalled(t, "Update")
	bus.AssertNotCalled(t, "Publish")
}

func TestDeactivateUser_PropagatesUpdateError(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	repo := mocks.NewUserRepository()
	bus := mocks.NewEventBus()
	repoErr := errors.New("db down")
	repo.On("FindByID", id).Return(newActiveUser(t, id, "user@example.com"), nil).Once()
	repo.On("Update", mock.AnythingOfType("*domain.User")).Return(repoErr).Once()

	cmd := command.NewDeactivateUserCommand(repo, bus)
	_, err := cmd.Execute(command.DeactivateUserParams{ID: id.String()})

	require.ErrorIs(t, err, repoErr)
	bus.AssertNotCalled(t, "Publish")
}
