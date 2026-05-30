package command_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/domain/mocks"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/command"
)

// otherEvent は UserDeactivated 以外の DomainEvent（想定外型）を表すテスト用スタブ。
type otherEvent struct{}

func (otherEvent) EventName() string     { return "other.event" }
func (otherEvent) OccurredAt() time.Time { return time.Now() }

func assigneeIDStrings(ids []domain.UserID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func newTaskWithAssignees(assigneeIDs ...domain.UserID) *domain.Task {
	ids := make([]string, len(assigneeIDs))
	for i, id := range assigneeIDs {
		ids[i] = id.String()
	}
	return domain.ReconstructTask(
		domain.GenerateTaskID().String(),
		"Task",
		"",
		domain.TaskStatusTodo,
		domain.PriorityLow,
		nil,
		ids,
	)
}

func TestRemoveAssigneeOnUserDeactivated_EventName(t *testing.T) {
	t.Parallel()

	h := command.NewRemoveAssigneeOnUserDeactivated(mocks.NewTaskRepository())
	assert.Equal(t, domain.UserDeactivated{}.EventName(), h.EventName())
}

func TestRemoveAssigneeOnUserDeactivated_Success(t *testing.T) {
	t.Parallel()

	target := domain.GenerateUserID()
	other := domain.GenerateUserID()
	task1 := newTaskWithAssignees(target, other)
	task2 := newTaskWithAssignees(target)

	repo := mocks.NewTaskRepository()
	repo.On("FindByAssigneeID", target).Return([]*domain.Task{task1, task2}, nil).Once()
	repo.On("Update", task1).Return(nil).Once()
	repo.On("Update", task2).Return(nil).Once()

	h := command.NewRemoveAssigneeOnUserDeactivated(repo)
	require.NoError(t, h.Handle(domain.NewUserDeactivated(target)))

	assert.NotContains(t, assigneeIDStrings(task1.AssigneeIDs()), target.String())
	assert.Contains(t, assigneeIDStrings(task1.AssigneeIDs()), other.String())
	assert.Empty(t, task2.AssigneeIDs())
	repo.AssertExpectations(t)
}

func TestRemoveAssigneeOnUserDeactivated_NoTasks(t *testing.T) {
	t.Parallel()

	target := domain.GenerateUserID()
	repo := mocks.NewTaskRepository()
	repo.On("FindByAssigneeID", target).Return(nil, nil).Once()

	h := command.NewRemoveAssigneeOnUserDeactivated(repo)
	require.NoError(t, h.Handle(domain.NewUserDeactivated(target)))

	repo.AssertNotCalled(t, "Update")
}

func TestRemoveAssigneeOnUserDeactivated_PropagatesFindError(t *testing.T) {
	t.Parallel()

	target := domain.GenerateUserID()
	repoErr := errors.New("db down")
	repo := mocks.NewTaskRepository()
	repo.On("FindByAssigneeID", target).Return(nil, repoErr).Once()

	h := command.NewRemoveAssigneeOnUserDeactivated(repo)
	require.ErrorIs(t, h.Handle(domain.NewUserDeactivated(target)), repoErr)

	repo.AssertNotCalled(t, "Update")
}

func TestRemoveAssigneeOnUserDeactivated_PropagatesUpdateError(t *testing.T) {
	t.Parallel()

	target := domain.GenerateUserID()
	task1 := newTaskWithAssignees(target)
	task2 := newTaskWithAssignees(target)
	repoErr := errors.New("db down")

	repo := mocks.NewTaskRepository()
	repo.On("FindByAssigneeID", target).Return([]*domain.Task{task1, task2}, nil).Once()
	repo.On("Update", task1).Return(repoErr).Once()

	h := command.NewRemoveAssigneeOnUserDeactivated(repo)
	require.ErrorIs(t, h.Handle(domain.NewUserDeactivated(target)), repoErr)

	// 先頭タスクで中断するため 2 件目の Update は呼ばれない。
	repo.AssertNumberOfCalls(t, "Update", 1)
}

func TestRemoveAssigneeOnUserDeactivated_UnexpectedEventType(t *testing.T) {
	t.Parallel()

	repo := mocks.NewTaskRepository()
	h := command.NewRemoveAssigneeOnUserDeactivated(repo)

	require.Error(t, h.Handle(otherEvent{}))
	repo.AssertNotCalled(t, "FindByAssigneeID")
}
