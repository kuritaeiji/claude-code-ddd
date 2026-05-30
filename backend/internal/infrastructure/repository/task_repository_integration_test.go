package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// persistActiveUser は task_assignees の FK（user_id → users.id）を満たすため、
// 担当者となる ACTIVE ユーザーを実 DB に永続化して返す。
func persistActiveUser(t *testing.T, db *gorm.DB, emailValue, displayName string) *domain.User {
	t.Helper()
	user := newActiveUser(t, emailValue, displayName)
	require.NoError(t, NewUserRepository(db).Insert(user))
	return user
}

func newTodoTask(t *testing.T, due *time.Time, assignees ...*domain.User) *domain.Task {
	t.Helper()
	task, err := domain.NewTask("Test task", "desc", string(domain.PriorityMedium), due, assignees)
	require.NoError(t, err)
	return task
}

func TestIntegration_TaskRepository_Insert_WithAssignees(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	bob := persistActiveUser(t, db, "bob@example.com", "Bob")

	due := time.Now().AddDate(0, 0, 7)
	task := newTodoTask(t, &due, alice, bob)
	require.NoError(t, repo.Insert(task))

	got, err := repo.FindByID(task.ID())
	require.NoError(t, err)
	assert.True(t, task.ID().Equals(got.ID()))
	assert.Equal(t, "Test task", got.Title())
	assert.Equal(t, domain.TaskStatusTodo, got.Status())
	assert.Equal(t, domain.PriorityMedium, got.Priority())
	require.NotNil(t, got.DueDate())
	assert.Len(t, got.AssigneeIDs(), 2)

	var count int64
	require.NoError(t, db.Model(&taskAssigneeModel{}).Where("task_id = ?", task.ID().String()).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestIntegration_TaskRepository_Insert_NoAssignees(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	task := newTodoTask(t, nil)
	require.NoError(t, repo.Insert(task))

	got, err := repo.FindByID(task.ID())
	require.NoError(t, err)
	assert.Nil(t, got.DueDate())
	assert.Empty(t, got.AssigneeIDs())
}

func TestIntegration_TaskRepository_FindByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	_, err := repo.FindByID(domain.GenerateTaskID())
	require.ErrorIs(t, err, domain.ErrTaskNotFound)
}

// Update が担当者一覧を全置換することを検証する（T-02 の担当者変更経路）。
func TestIntegration_TaskRepository_Update_ReplacesAssignees(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	bob := persistActiveUser(t, db, "bob@example.com", "Bob")
	carol := persistActiveUser(t, db, "carol@example.com", "Carol")

	task := newTodoTask(t, nil, alice, bob)
	require.NoError(t, repo.Insert(task))

	due := time.Now().AddDate(0, 0, 3)
	require.NoError(t, task.Update("Updated", "new desc", string(domain.TaskStatusDone), string(domain.PriorityHigh), &due, []*domain.User{carol}))
	require.NoError(t, repo.Update(task))

	got, err := repo.FindByID(task.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title())
	assert.Equal(t, domain.TaskStatusDone, got.Status())
	assert.Equal(t, domain.PriorityHigh, got.Priority())
	require.NotNil(t, got.DueDate())

	ids := got.AssigneeIDs()
	require.Len(t, ids, 1)
	assert.True(t, carol.ID().Equals(ids[0]))
}

// Update でタスクを全担当者から外す（RemoveAssignee の結果のような空一覧）経路を検証する。
func TestIntegration_TaskRepository_Update_RemovesAllAssignees(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	task := newTodoTask(t, nil, alice)
	require.NoError(t, repo.Insert(task))

	require.NoError(t, task.Update("Updated", "", string(domain.TaskStatusTodo), string(domain.PriorityLow), nil, nil))
	require.NoError(t, repo.Update(task))

	got, err := repo.FindByID(task.ID())
	require.NoError(t, err)
	assert.Empty(t, got.AssigneeIDs())

	var count int64
	require.NoError(t, db.Model(&taskAssigneeModel{}).Where("task_id = ?", task.ID().String()).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestIntegration_TaskRepository_Delete(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	task := newTodoTask(t, nil, alice)
	require.NoError(t, repo.Insert(task))

	require.NoError(t, repo.Delete(task.ID()))

	_, err := repo.FindByID(task.ID())
	require.ErrorIs(t, err, domain.ErrTaskNotFound)

	var count int64
	require.NoError(t, db.Model(&taskAssigneeModel{}).Where("task_id = ?", task.ID().String()).Count(&count).Error)
	assert.Equal(t, int64(0), count, "delete must also remove task_assignees rows")
}

func TestIntegration_TaskRepository_FindByAssigneeID(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	alice := persistActiveUser(t, db, "alice@example.com", "Alice")
	bob := persistActiveUser(t, db, "bob@example.com", "Bob")

	task1 := newTodoTask(t, nil, alice, bob)
	task2 := newTodoTask(t, nil, alice)
	task3 := newTodoTask(t, nil, bob)
	require.NoError(t, repo.Insert(task1))
	require.NoError(t, repo.Insert(task2))
	require.NoError(t, repo.Insert(task3))

	got, err := repo.FindByAssigneeID(alice.ID())
	require.NoError(t, err)
	require.Len(t, got, 2)

	gotIDs := map[string]bool{}
	for _, task := range got {
		gotIDs[task.ID().String()] = true
	}
	assert.True(t, gotIDs[task1.ID().String()])
	assert.True(t, gotIDs[task2.ID().String()])
	assert.False(t, gotIDs[task3.ID().String()])
}

func TestIntegration_TaskRepository_FindByAssigneeID_None(t *testing.T) {
	db := newTestDB(t)
	repo := NewTaskRepository(db)

	got, err := repo.FindByAssigneeID(domain.GenerateUserID())
	require.NoError(t, err)
	assert.Empty(t, got)
}
