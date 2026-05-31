package query_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
	querymocks "github.com/kuritaeiji/claude-code-ddd/internal/usecase/query/mocks"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func sampleTaskListItems() []query.TaskListItemDTO {
	due := time.Now().Add(48 * time.Hour)
	return []query.TaskListItemDTO{
		{ID: "id-1", Title: "Task A", Status: "TODO", Priority: "HIGH", DueDate: &due, AssigneeNames: []string{"Alice"}},
		{ID: "id-2", Title: "Task B", Status: "TODO", Priority: "LOW", AssigneeNames: []string{}},
	}
}

func TestListTasksQuery_NoFilter(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{}
	repo.On("ListTasks", params).Return(sampleTaskListItems(), nil).Once()

	q := query.NewListTasksQuery(repo)
	dtos, err := q.Execute(params)

	require.NoError(t, err)
	assert.Len(t, dtos, 2)
	repo.AssertExpectations(t)
}

func TestListTasksQuery_FilterBySingleStatus(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Statuses: []string{"TODO"}}
	repo.On("ListTasks", params).Return(sampleTaskListItems(), nil).Once()

	q := query.NewListTasksQuery(repo)
	dtos, err := q.Execute(params)

	require.NoError(t, err)
	assert.Len(t, dtos, 2)
	repo.AssertExpectations(t)
}

func TestListTasksQuery_FilterByMultipleStatuses(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Statuses: []string{"TODO", "IN_PROGRESS"}}
	repo.On("ListTasks", params).Return(sampleTaskListItems(), nil).Once()

	q := query.NewListTasksQuery(repo)
	dtos, err := q.Execute(params)

	require.NoError(t, err)
	assert.Len(t, dtos, 2)
	repo.AssertExpectations(t)
}

func TestListTasksQuery_FilterByMultiplePriorities(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Priorities: []string{"HIGH", "CRITICAL"}}
	repo.On("ListTasks", params).Return(sampleTaskListItems()[:1], nil).Once()

	q := query.NewListTasksQuery(repo)
	dtos, err := q.Execute(params)

	require.NoError(t, err)
	assert.Len(t, dtos, 1)
	repo.AssertExpectations(t)
}

func TestListTasksQuery_FilterByBoth(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{Statuses: []string{"DONE"}, Priorities: []string{"CRITICAL"}}
	repo.On("ListTasks", params).Return([]query.TaskListItemDTO{}, nil).Once()

	q := query.NewListTasksQuery(repo)
	dtos, err := q.Execute(params)

	require.NoError(t, err)
	assert.Empty(t, dtos)
	repo.AssertExpectations(t)
}

func TestListTasksQuery_InvalidStatusInList(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	q := query.NewListTasksQuery(repo)
	_, err := q.Execute(query.ListTasksParams{Statuses: []string{"TODO", "INVALID"}})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	repo.AssertNotCalled(t, "ListTasks")
}

func TestListTasksQuery_InvalidPriorityInList(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	q := query.NewListTasksQuery(repo)
	_, err := q.Execute(query.ListTasksParams{Priorities: []string{"HIGH", "URGENT"}})

	var ve *apperrors.ValidationError
	require.ErrorAs(t, err, &ve)
	repo.AssertNotCalled(t, "ListTasks")
}

func TestListTasksQuery_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	repo := querymocks.NewTaskQueryRepository()
	params := query.ListTasksParams{}
	repoErr := errors.New("db down")
	repo.On("ListTasks", params).Return(nil, repoErr).Once()

	q := query.NewListTasksQuery(repo)
	_, err := q.Execute(params)

	require.ErrorIs(t, err, repoErr)
}
