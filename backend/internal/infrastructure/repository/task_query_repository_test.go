package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
)

// persistTask は tasks + task_assignees を実 DB に書き込み、task を返す。
func persistTask(t *testing.T, repo domain.TaskRepository, task *domain.Task) {
	t.Helper()
	require.NoError(t, repo.Insert(task))
}

func TestIntegration_TaskQueryRepository_ListTasks_NoFilter(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	t1 := newTodoTask(t, nil, alice)
	t2 := newTodoTask(t, nil)
	persistTask(t, taskRepo, t1)
	persistTask(t, taskRepo, t2)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{})
	require.NoError(t, err)
	assert.Len(t, dtos, 2)
}

func TestIntegration_TaskQueryRepository_ListTasks_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	done := newTodoTask(t, nil)
	todo := newTodoTask(t, nil)
	persistTask(t, taskRepo, done)
	persistTask(t, taskRepo, todo)

	// done ステータスに更新
	require.NoError(t, done.Update("done task", "", string(domain.TaskStatusDone), string(domain.PriorityLow), nil, nil))
	require.NoError(t, taskRepo.Update(done))

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{Statuses: []string{"DONE"}})
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.Equal(t, done.ID().String(), dtos[0].ID)
	assert.Equal(t, "DONE", dtos[0].Status)
}

func TestIntegration_TaskQueryRepository_ListTasks_FilterByMultipleStatuses(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	todo := newTodoTask(t, nil)
	inProgress := newTodoTask(t, nil)
	done := newTodoTask(t, nil)
	persistTask(t, taskRepo, todo)
	persistTask(t, taskRepo, inProgress)
	persistTask(t, taskRepo, done)

	require.NoError(t, inProgress.Update("ip", "", string(domain.TaskStatusInProgress), string(domain.PriorityLow), nil, nil))
	require.NoError(t, taskRepo.Update(inProgress))
	require.NoError(t, done.Update("done", "", string(domain.TaskStatusDone), string(domain.PriorityLow), nil, nil))
	require.NoError(t, taskRepo.Update(done))

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{Statuses: []string{"TODO", "IN_PROGRESS"}})
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	gotIDs := map[string]bool{}
	for _, dto := range dtos {
		gotIDs[dto.ID] = true
	}
	assert.True(t, gotIDs[todo.ID().String()])
	assert.True(t, gotIDs[inProgress.ID().String()])
	assert.False(t, gotIDs[done.ID().String()])
}

func TestIntegration_TaskQueryRepository_ListTasks_FilterByPriority(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	highTask, err := domain.NewTask("High Task", "", string(domain.PriorityHigh), nil, nil)
	require.NoError(t, err)
	lowTask := newTodoTask(t, nil)
	persistTask(t, taskRepo, highTask)
	persistTask(t, taskRepo, lowTask)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{Priorities: []string{"HIGH"}})
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.Equal(t, highTask.ID().String(), dtos[0].ID)
	assert.Equal(t, "HIGH", dtos[0].Priority)
}

func TestIntegration_TaskQueryRepository_ListTasks_FilterByMultiplePriorities(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	highTask, err := domain.NewTask("High Task", "", string(domain.PriorityHigh), nil, nil)
	require.NoError(t, err)
	criticalTask, err := domain.NewTask("Critical Task", "", string(domain.PriorityCritical), nil, nil)
	require.NoError(t, err)
	lowTask := newTodoTask(t, nil)
	persistTask(t, taskRepo, highTask)
	persistTask(t, taskRepo, criticalTask)
	persistTask(t, taskRepo, lowTask)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{Priorities: []string{"HIGH", "CRITICAL"}})
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	gotIDs := map[string]bool{}
	for _, dto := range dtos {
		gotIDs[dto.ID] = true
	}
	assert.True(t, gotIDs[highTask.ID().String()])
	assert.True(t, gotIDs[criticalTask.ID().String()])
	assert.False(t, gotIDs[lowTask.ID().String()])
}

func TestIntegration_TaskQueryRepository_ListTasks_FilterByBoth(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	// DONE かつ CRITICAL なタスク
	target, err := domain.NewTask("Target", "", string(domain.PriorityCritical), nil, nil)
	require.NoError(t, err)
	require.NoError(t, target.Update("Target", "", string(domain.TaskStatusDone), string(domain.PriorityCritical), nil, nil))
	persistTask(t, taskRepo, target)

	// TODO/CRITICAL なタスク（ステータスが違う）
	other, err := domain.NewTask("Other", "", string(domain.PriorityCritical), nil, nil)
	require.NoError(t, err)
	persistTask(t, taskRepo, other)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{Statuses: []string{"DONE"}, Priorities: []string{"CRITICAL"}})
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.Equal(t, target.ID().String(), dtos[0].ID)
}

func TestIntegration_TaskQueryRepository_ListTasks_WithAssigneeNames(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	bob := persistActiveUser(t, db, "bob@example.com", "Bob")
	task := newTodoTask(t, nil, alice, bob)
	persistTask(t, taskRepo, task)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{})
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.ElementsMatch(t, []string{"Alice", "Bob"}, dtos[0].AssigneeNames)
}

func TestIntegration_TaskQueryRepository_ListTasks_EmptyResult(t *testing.T) {
	db := newTestDB(t)
	queryRepo := NewTaskQueryRepository(db)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{})
	require.NoError(t, err)
	assert.Empty(t, dtos)
}

func TestIntegration_TaskQueryRepository_ListTasks_DueDate(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	due := time.Now().AddDate(0, 0, 7)
	task := newTodoTask(t, &due)
	persistTask(t, taskRepo, task)

	dtos, err := queryRepo.ListTasks(query.ListTasksParams{})
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	require.NotNil(t, dtos[0].DueDate)
	assert.Equal(t, due.UTC().Format("2006-01-02"), dtos[0].DueDate.UTC().Format("2006-01-02"))
}

func TestIntegration_TaskQueryRepository_GetTaskByID_Success(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	task := newTodoTask(t, nil, alice)
	persistTask(t, taskRepo, task)

	dto, err := queryRepo.GetTaskByID(task.ID().String())
	require.NoError(t, err)
	assert.Equal(t, task.ID().String(), dto.ID)
	assert.Equal(t, "Test task", dto.Title)
	assert.Equal(t, "desc", dto.Description)
	assert.Equal(t, "TODO", dto.Status)
	assert.Equal(t, "MEDIUM", dto.Priority)
	assert.Equal(t, []string{"Alice"}, dto.AssigneeNames)
}

func TestIntegration_TaskQueryRepository_GetTaskByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	queryRepo := NewTaskQueryRepository(db)

	_, err := queryRepo.GetTaskByID(domain.GenerateTaskID().String())
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestIntegration_TaskQueryRepository_GetTaskByID_NoAssignees(t *testing.T) {
	db := newTestDB(t)
	taskRepo := NewTaskRepository(db)
	queryRepo := NewTaskQueryRepository(db)

	task := newTodoTask(t, nil)
	persistTask(t, taskRepo, task)

	dto, err := queryRepo.GetTaskByID(task.ID().String())
	require.NoError(t, err)
	assert.Empty(t, dto.AssigneeNames)
}
