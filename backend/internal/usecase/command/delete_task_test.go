package command_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func TestDeleteTask_Success(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	tasks.On("Delete", id).Return(nil).Once()

	cmd := command.NewDeleteTaskCommand(tasks)
	err := cmd.Execute(command.DeleteTaskParams{ID: id.String()})

	require.NoError(t, err)
	tasks.AssertExpectations(t)
}

func TestDeleteTask_InvalidID(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()

	cmd := command.NewDeleteTaskCommand(tasks)
	err := cmd.Execute(command.DeleteTaskParams{ID: "not-a-uuid"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "FindByID")
	tasks.AssertNotCalled(t, "Delete")
}

func TestDeleteTask_NotFound(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	tasks.On("FindByID", id).Return(nil, domain.ErrTaskNotFound).Once()

	cmd := command.NewDeleteTaskCommand(tasks)
	err := cmd.Execute(command.DeleteTaskParams{ID: id.String()})

	var nfe *apperrors.NotFoundError
	require.ErrorAs(t, err, &nfe)
	tasks.AssertNotCalled(t, "Delete")
}

func TestDeleteTask_PropagatesDeleteError(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	tasks := mocks.NewTaskRepository()
	repoErr := errors.New("db down")
	tasks.On("FindByID", id).Return(newExistingTask(id), nil).Once()
	tasks.On("Delete", id).Return(repoErr).Once()

	cmd := command.NewDeleteTaskCommand(tasks)
	err := cmd.Execute(command.DeleteTaskParams{ID: id.String()})

	require.ErrorIs(t, err, repoErr)
}
