package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

func TestTaskToModel(t *testing.T) {
	t.Parallel()

	due := time.Date(2999, 12, 31, 0, 0, 0, 0, time.UTC)
	id := domain.GenerateTaskID()
	uid := domain.GenerateUserID()
	task := domain.ReconstructTask(
		id.String(),
		"Write report",
		"some description",
		domain.TaskStatusInProgress,
		domain.PriorityHigh,
		&due,
		[]string{uid.String()},
	)

	m := taskToModel(task)

	assert.Equal(t, id.String(), m.ID)
	assert.Equal(t, "Write report", m.Title)
	assert.Equal(t, "some description", m.Description)
	assert.Equal(t, string(domain.TaskStatusInProgress), m.Status)
	assert.Equal(t, string(domain.PriorityHigh), m.Priority)
	require.NotNil(t, m.DueDate)
	assert.True(t, due.Equal(*m.DueDate))

	assignees := assigneeModels(task)
	require.Len(t, assignees, 1)
	assert.Equal(t, id.String(), assignees[0].TaskID)
	assert.Equal(t, uid.String(), assignees[0].UserID)
}

func TestTaskToModel_NilDueDate(t *testing.T) {
	t.Parallel()

	task := domain.ReconstructTask(
		domain.GenerateTaskID().String(),
		"No due date",
		"",
		domain.TaskStatusTodo,
		domain.PriorityLow,
		nil,
		nil,
	)

	m := taskToModel(task)

	assert.Nil(t, m.DueDate)
	assert.Empty(t, assigneeModels(task))
}

func TestTaskToDomain(t *testing.T) {
	t.Parallel()

	idStr := uuid.NewString()
	uid1 := uuid.NewString()
	uid2 := uuid.NewString()
	due := time.Date(2999, 1, 2, 0, 0, 0, 0, time.UTC)
	m := &taskModel{
		ID:          idStr,
		Title:       "Restored",
		Description: "desc",
		Status:      string(domain.TaskStatusDone),
		Priority:    string(domain.PriorityCritical),
		DueDate:     &due,
	}
	assignees := []taskAssigneeModel{
		{TaskID: idStr, UserID: uid1},
		{TaskID: idStr, UserID: uid2},
	}

	task := taskToDomain(m, assignees)

	assert.Equal(t, idStr, task.ID().String())
	assert.Equal(t, "Restored", task.Title())
	assert.Equal(t, "desc", task.Description())
	assert.Equal(t, domain.TaskStatusDone, task.Status())
	assert.Equal(t, domain.PriorityCritical, task.Priority())
	require.NotNil(t, task.DueDate())
	assert.Equal(t, "2999-01-02", task.DueDate().String())

	ids := task.AssigneeIDs()
	require.Len(t, ids, 2)
	assert.Equal(t, uid1, ids[0].String())
	assert.Equal(t, uid2, ids[1].String())
}

func TestTaskToDomain_NilDueDate_NoAssignees(t *testing.T) {
	t.Parallel()

	m := &taskModel{
		ID:       uuid.NewString(),
		Title:    "Minimal",
		Status:   string(domain.TaskStatusTodo),
		Priority: string(domain.PriorityMedium),
	}

	task := taskToDomain(m, nil)

	assert.Nil(t, task.DueDate())
	assert.Empty(t, task.AssigneeIDs())
}

func TestTaskModel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tasks", taskModel{}.TableName())
	assert.Equal(t, "task_assignees", taskAssigneeModel{}.TableName())
}
