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

func newExistingTask(id domain.TaskID) *domain.Task {
	return domain.ReconstructTask(
		id.String(), "old title", "old description",
		domain.TaskStatusTodo, domain.PriorityLow, nil, nil,
	)
}

func TestUpdateTask_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	tasks.On("Update", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	dto, err := cmd.Execute(command.UpdateTaskParams{
		ID:       id.String(),
		Title:    "new title",
		Status:   "DONE",
		Priority: "CRITICAL",
	})
	require.NoError(t, err)

	assert.Equal(t, id.String(), dto.ID)
	assert.Equal(t, "new title", dto.Title)
	assert.Equal(t, "DONE", dto.Status)
	assert.Equal(t, "CRITICAL", dto.Priority)
	tasks.AssertExpectations(t)
	users.AssertNotCalled(t, "FindByIDs")
}

func TestUpdateTask_InvalidID(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{ID: "not-a-uuid", Title: "x", Status: "TODO", Priority: "LOW"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "FindByID")
}

func TestUpdateTask_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(nil, domain.ErrTaskNotFound).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{ID: id.String(), Title: "x", Status: "TODO", Priority: "LOW"})

	var nfe *apperrors.NotFoundError
	require.ErrorAs(t, err, &nfe)
	tasks.AssertNotCalled(t, "Update")
}

func TestUpdateTask_InvalidStatus(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{ID: id.String(), Title: "x", Status: "ARCHIVED", Priority: "LOW"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Update")
}

func TestUpdateTask_InactiveAssignee(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	userID := domain.GenerateUserID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	users.On("FindByIDs", mock.AnythingOfType("[]domain.UserID")).
		Return([]*domain.User{newInactiveUser(t, userID, "inactive@example.com")}, nil).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{
		ID: id.String(), Title: "x", Status: "TODO", Priority: "LOW",
		AssigneeIDs: []string{userID.String()},
	})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Update")
}

func TestUpdateTask_AssigneeNotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	userID := domain.GenerateUserID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	users.On("FindByIDs", mock.AnythingOfType("[]domain.UserID")).Return([]*domain.User{}, nil).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{
		ID: id.String(), Title: "x", Status: "TODO", Priority: "LOW",
		AssigneeIDs: []string{userID.String()},
	})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Update")
}

func TestUpdateTask_InvalidAssigneeID(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{
		ID: id.String(), Title: "x", Status: "TODO", Priority: "LOW",
		AssigneeIDs: []string{"not-a-uuid"},
	})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	users.AssertNotCalled(t, "FindByIDs")
	tasks.AssertNotCalled(t, "Update")
}

func TestUpdateTask_PropagatesUpdateError(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	repoErr := errors.New("db down")
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	tasks.On("Update", mock.AnythingOfType("*domain.Task")).Return(repoErr).Once()

	cmd := command.NewUpdateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.UpdateTaskParams{ID: id.String(), Title: "x", Status: "TODO", Priority: "LOW"})

	require.ErrorIs(t, err, repoErr)
}
