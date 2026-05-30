package command_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// newInactiveUser は担当者の非活性ケース検証用に INACTIVE な User を復元する。
func newInactiveUser(t *testing.T, id domain.UserID, emailValue string) *domain.User {
	t.Helper()
	email, err := domain.NewEmail(emailValue)
	require.NoError(t, err)
	return domain.ReconstructUser(id.String(), email.String(), "Bob", domain.UserStatusInactive)
}

func TestCreateTask_Success_NoAssignees(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	tasks.On("Insert", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	cmd := command.NewCreateTaskCommand(tasks, users)
	dto, err := cmd.Execute(command.CreateTaskParams{
		Title:    "Write report",
		Priority: "HIGH",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, dto.ID)
	assert.Equal(t, "Write report", dto.Title)
	assert.Equal(t, string(domain.TaskStatusTodo), dto.Status)
	assert.Equal(t, "HIGH", dto.Priority)
	assert.Nil(t, dto.DueDate)
	assert.Empty(t, dto.AssigneeIDs)
	tasks.AssertExpectations(t)
	users.AssertNotCalled(t, "FindByIDs")
}

func TestCreateTask_Success_WithAssignees(t *testing.T) {
	t.Parallel()

	id1 := domain.GenerateUserID()
	id2 := domain.GenerateUserID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	users.On("FindByIDs", mock.AnythingOfType("[]domain.UserID")).
		Return([]*domain.User{newActiveUser(t, id1, "a@example.com"), newActiveUser(t, id2, "b@example.com")}, nil).Once()
	tasks.On("Insert", mock.AnythingOfType("*domain.Task")).Return(nil).Once()

	due := time.Now().Add(48 * time.Hour)
	cmd := command.NewCreateTaskCommand(tasks, users)
	dto, err := cmd.Execute(command.CreateTaskParams{
		Title:       "Pair task",
		Description: "with assignees",
		Priority:    "MEDIUM",
		DueDate:     &due,
		AssigneeIDs: []string{id1.String(), id2.String()},
	})
	require.NoError(t, err)

	require.NotNil(t, dto.DueDate)
	assert.Equal(t, due.UTC().Format("2006-01-02"), dto.DueDate.UTC().Format("2006-01-02"))
	assert.ElementsMatch(t, []string{id1.String(), id2.String()}, dto.AssigneeIDs)
	tasks.AssertExpectations(t)
	users.AssertExpectations(t)
}

func TestCreateTask_InvalidPriority(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()

	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "URGENT"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_TitleTooLong(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()

	long := make([]rune, 201)
	for i := range long {
		long[i] = 'a'
	}
	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: string(long), Priority: "LOW"})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_PastDueDate(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()

	past := time.Now().Add(-48 * time.Hour)
	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "LOW", DueDate: &past})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_InactiveAssignee(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	users.On("FindByIDs", mock.AnythingOfType("[]domain.UserID")).
		Return([]*domain.User{newInactiveUser(t, id, "inactive@example.com")}, nil).Once()

	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "LOW", AssigneeIDs: []string{id.String()}})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_AssigneeNotFound(t *testing.T) {
	t.Parallel()

	id1 := domain.GenerateUserID()
	id2 := domain.GenerateUserID()
	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	users.On("FindByIDs", mock.AnythingOfType("[]domain.UserID")).
		Return([]*domain.User{newActiveUser(t, id1, "a@example.com")}, nil).Once()

	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "LOW", AssigneeIDs: []string{id1.String(), id2.String()}})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_InvalidAssigneeID(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()

	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "LOW", AssigneeIDs: []string{"not-a-uuid"}})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	users.AssertNotCalled(t, "FindByIDs")
	tasks.AssertNotCalled(t, "Insert")
}

func TestCreateTask_PropagatesInsertError(t *testing.T) {
	t.Parallel()

	tasks := mocks.NewTaskRepository()
	users := mocks.NewUserRepository()
	repoErr := errors.New("db down")
	tasks.On("Insert", mock.AnythingOfType("*domain.Task")).Return(repoErr).Once()

	cmd := command.NewCreateTaskCommand(tasks, users)
	_, err := cmd.Execute(command.CreateTaskParams{Title: "x", Priority: "LOW"})

	require.ErrorIs(t, err, repoErr)
}
