package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

// --- TaskID ---

func TestNewTaskID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid uuid v4", uuid.NewString(), false},
		{"empty string", "", true},
		{"non-uuid", "not-a-uuid", true},
		{"truncated uuid", "550e8400-e29b-41d4-a716", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := domain.NewTaskID(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.value, id.String())
		})
	}
}

func TestGenerateTaskID_IsValid(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	_, err := uuid.Parse(id.String())
	assert.NoError(t, err)
}

func TestTaskID_Equals(t *testing.T) {
	t.Parallel()

	value := uuid.NewString()
	a, err := domain.NewTaskID(value)
	require.NoError(t, err)
	b, err := domain.NewTaskID(value)
	require.NoError(t, err)
	c := domain.GenerateTaskID()

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

// --- TaskStatus ---

func TestNewTaskStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    domain.TaskStatus
		wantErr bool
	}{
		{"todo", "TODO", domain.TaskStatusTodo, false},
		{"in_progress", "IN_PROGRESS", domain.TaskStatusInProgress, false},
		{"done", "DONE", domain.TaskStatusDone, false},
		{"lowercase rejected", "todo", "", true},
		{"unknown", "PENDING", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewTaskStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Priority ---

func TestNewPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    domain.Priority
		wantErr bool
	}{
		{"low", "LOW", domain.PriorityLow, false},
		{"medium", "MEDIUM", domain.PriorityMedium, false},
		{"high", "HIGH", domain.PriorityHigh, false},
		{"critical", "CRITICAL", domain.PriorityCritical, false},
		{"lowercase rejected", "low", "", true},
		{"unknown", "URGENT", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewPriority(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- DueDate ---

func TestNewDueDate(t *testing.T) {
	t.Parallel()

	t.Run("today is accepted", func(t *testing.T) {
		t.Parallel()
		d, err := domain.NewDueDate(time.Now())
		require.NoError(t, err)
		assert.Equal(t, time.Now().Format("2006-01-02"), d.String())
	})

	t.Run("future date is accepted", func(t *testing.T) {
		t.Parallel()
		future := time.Now().AddDate(0, 0, 1)
		d, err := domain.NewDueDate(future)
		require.NoError(t, err)
		assert.Equal(t, future.Format("2006-01-02"), d.String())
	})

	t.Run("past date is rejected (BR-T-03)", func(t *testing.T) {
		t.Parallel()
		past := time.Now().AddDate(0, 0, -1)
		_, err := domain.NewDueDate(past)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})
}

func TestDueDate_Equals(t *testing.T) {
	t.Parallel()

	future := time.Now().AddDate(0, 0, 1)
	a, err := domain.NewDueDate(future)
	require.NoError(t, err)
	b, err := domain.NewDueDate(future)
	require.NoError(t, err)

	other := time.Now().AddDate(0, 0, 7)
	c, err := domain.NewDueDate(other)
	require.NoError(t, err)

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

// --- NewTask ---

func TestNewTask(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	activeUser := domain.ReconstructUser(domain.GenerateUserID().String(), email.String(), "Alice", domain.UserStatusActive)
	inactiveUser := domain.ReconstructUser(domain.GenerateUserID().String(), email.String(), "Bob", domain.UserStatusInactive)

	future := time.Now().AddDate(0, 0, 1)

	t.Run("valid task has TODO status (BR-T-01)", func(t *testing.T) {
		t.Parallel()
		task, err := domain.NewTask("Fix bug", "", "HIGH", &future, []*domain.User{activeUser})
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusTodo, task.Status())
		assert.Equal(t, "Fix bug", task.Title())
		assert.Equal(t, domain.PriorityHigh, task.Priority())
		require.Len(t, task.AssigneeIDs(), 1)
		assert.True(t, task.AssigneeIDs()[0].Equals(activeUser.ID()))
	})

	t.Run("nil dueDate is accepted", func(t *testing.T) {
		t.Parallel()
		task, err := domain.NewTask("No deadline", "", "LOW", nil, nil)
		require.NoError(t, err)
		assert.Nil(t, task.DueDate())
		assert.Empty(t, task.AssigneeIDs())
	})

	t.Run("empty title is rejected (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		_, err := domain.NewTask("", "", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("title over 200 chars is rejected (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		_, err := domain.NewTask(strings.Repeat("a", 201), "", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("title exactly 200 chars is accepted (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		maxTitle := strings.Repeat("a", 200)
		task, err := domain.NewTask(maxTitle, "", "LOW", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, maxTitle, task.Title())
	})

	t.Run("description over 5000 chars is rejected (BR-T-06)", func(t *testing.T) {
		t.Parallel()
		_, err := domain.NewTask("title", strings.Repeat("a", 5001), "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("description exactly 5000 chars is accepted (BR-T-06)", func(t *testing.T) {
		t.Parallel()
		maxDesc := strings.Repeat("a", 5000)
		task, err := domain.NewTask("title", maxDesc, "LOW", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, maxDesc, task.Description())
	})

	t.Run("invalid priority is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := domain.NewTask("title", "", "INVALID", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("past dueDate is rejected (BR-T-03)", func(t *testing.T) {
		t.Parallel()
		past := time.Now().AddDate(0, 0, -1)
		_, err := domain.NewTask("title", "", "LOW", &past, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("inactive assignee is rejected (BR-T-04)", func(t *testing.T) {
		t.Parallel()
		_, err := domain.NewTask("title", "", "LOW", nil, []*domain.User{inactiveUser})
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("multiple active assignees are accepted", func(t *testing.T) {
		t.Parallel()
		email2, err := domain.NewEmail("user2@example.com")
		require.NoError(t, err)
		active2 := domain.ReconstructUser(domain.GenerateUserID().String(), email2.String(), "Carol", domain.UserStatusActive)

		task, err := domain.NewTask("title", "", "MEDIUM", nil, []*domain.User{activeUser, active2})
		require.NoError(t, err)
		assert.Len(t, task.AssigneeIDs(), 2)
	})
}

// --- ReconstructTask ---

func TestReconstructTask(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	// 復元は DB 値を信頼するため、過去日付の dueDate も検証せず受け入れる（BR-T-03 を再検査しない）。
	past := time.Now().AddDate(0, 0, -10)

	task := domain.ReconstructTask(id.String(), "old title", "", domain.TaskStatusDone, domain.PriorityLow, &past, nil)

	assert.True(t, id.Equals(task.ID()))
	assert.Equal(t, domain.TaskStatusDone, task.Status())
	assert.Equal(t, "old title", task.Title())
	assert.Equal(t, domain.PriorityLow, task.Priority())
	require.NotNil(t, task.DueDate())
	assert.Equal(t, past.Format("2006-01-02"), task.DueDate().String())
}

// 復元はリポジトリの DB 値を信頼するため、BR-T-05 違反のタイトルでも検証せず復元可能。
func TestReconstructTask_SkipsValidation(t *testing.T) {
	t.Parallel()

	id := domain.GenerateTaskID()
	longTitle := strings.Repeat("a", 300) // BR-T-05 違反だが復元では許容
	task := domain.ReconstructTask(id.String(), longTitle, "", domain.TaskStatusTodo, domain.PriorityLow, nil, nil)
	assert.Equal(t, longTitle, task.Title())
}

// --- Task.Update ---

func TestTask_Update(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	active := domain.ReconstructUser(domain.GenerateUserID().String(), email.String(), "Alice", domain.UserStatusActive)
	inactive := domain.ReconstructUser(domain.GenerateUserID().String(), email.String(), "Bob", domain.UserStatusInactive)

	newTask := func() *domain.Task {
		return domain.ReconstructTask(domain.GenerateTaskID().String(), "original", "", domain.TaskStatusTodo, domain.PriorityLow, nil, nil)
	}

	t.Run("all fields updated", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		future := time.Now().AddDate(0, 0, 3)
		require.NoError(t, task.Update("new title", "new desc", "DONE", "HIGH", &future, []*domain.User{active}))
		assert.Equal(t, "new title", task.Title())
		assert.Equal(t, "new desc", task.Description())
		assert.Equal(t, domain.TaskStatusDone, task.Status())
		assert.Equal(t, domain.PriorityHigh, task.Priority())
		assert.NotNil(t, task.DueDate())
		require.Len(t, task.AssigneeIDs(), 1)
		assert.True(t, task.AssigneeIDs()[0].Equals(active.ID()))
	})

	t.Run("nil dueDate and empty assignees accepted", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		require.NoError(t, task.Update("title", "", "TODO", "LOW", nil, nil))
		assert.Nil(t, task.DueDate())
		assert.Empty(t, task.AssigneeIDs())
	})

	t.Run("empty title rejected (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update("", "", "TODO", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
		assert.Equal(t, "original", task.Title(), "fields must not change on error")
	})

	t.Run("title over 200 chars rejected (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update(strings.Repeat("x", 201), "", "TODO", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("title exactly 200 chars accepted (BR-T-05)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		maxTitle := strings.Repeat("a", 200)
		require.NoError(t, task.Update(maxTitle, "", "TODO", "LOW", nil, nil))
		assert.Equal(t, maxTitle, task.Title())
	})

	t.Run("description over 5000 chars rejected (BR-T-06)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update("title", strings.Repeat("a", 5001), "TODO", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("description exactly 5000 chars accepted (BR-T-06)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		maxDesc := strings.Repeat("a", 5000)
		require.NoError(t, task.Update("title", maxDesc, "TODO", "LOW", nil, nil))
		assert.Equal(t, maxDesc, task.Description())
	})

	t.Run("invalid status rejected", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update("title", "", "INVALID", "LOW", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("invalid priority rejected", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update("title", "", "TODO", "INVALID", nil, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("past dueDate rejected (BR-T-03)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		past := time.Now().AddDate(0, 0, -1)
		err := task.Update("title", "", "TODO", "LOW", &past, nil)
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("inactive assignee rejected (BR-T-04)", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		err := task.Update("title", "", "TODO", "LOW", nil, []*domain.User{inactive})
		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("status transitions have no constraint (BR-T-02)", func(t *testing.T) {
		t.Parallel()
		transitions := []struct {
			from  domain.TaskStatus
			toStr string
		}{
			{domain.TaskStatusTodo, "DONE"},
			{domain.TaskStatusDone, "TODO"},
			{domain.TaskStatusInProgress, "TODO"},
		}
		for _, tt := range transitions {
			tt := tt
			t.Run(string(tt.from)+"->"+tt.toStr, func(t *testing.T) {
				t.Parallel()
				task := domain.ReconstructTask(domain.GenerateTaskID().String(), "title", "", tt.from, domain.PriorityLow, nil, nil)
				require.NoError(t, task.Update("title", "", tt.toStr, "LOW", nil, nil))
			})
		}
	})

	t.Run("validation failure leaves all fields unchanged", func(t *testing.T) {
		t.Parallel()
		task := newTask()
		_ = task.Update("", "new desc", "DONE", "HIGH", nil, nil)
		assert.Equal(t, "original", task.Title())
		assert.Equal(t, "", task.Description())
		assert.Equal(t, domain.TaskStatusTodo, task.Status())
		assert.Equal(t, domain.PriorityLow, task.Priority())
	})
}

// --- Task.RemoveAssignee ---

func TestTask_RemoveAssignee(t *testing.T) {
	t.Parallel()

	id1 := domain.GenerateUserID()
	id2 := domain.GenerateUserID()

	t.Run("removes matching assignee", func(t *testing.T) {
		t.Parallel()
		task := domain.ReconstructTask(domain.GenerateTaskID().String(), "title", "", domain.TaskStatusTodo, domain.PriorityLow, nil, []string{id1.String(), id2.String()})
		task.RemoveAssignee(id1)
		ids := task.AssigneeIDs()
		require.Len(t, ids, 1)
		assert.True(t, ids[0].Equals(id2))
	})

	t.Run("no-op when assignee not found", func(t *testing.T) {
		t.Parallel()
		task := domain.ReconstructTask(domain.GenerateTaskID().String(), "title", "", domain.TaskStatusTodo, domain.PriorityLow, nil, []string{id1.String()})
		task.RemoveAssignee(id2)
		assert.Len(t, task.AssigneeIDs(), 1)
	})

	t.Run("removes all if multiple same IDs", func(t *testing.T) {
		t.Parallel()
		task := domain.ReconstructTask(domain.GenerateTaskID().String(), "title", "", domain.TaskStatusTodo, domain.PriorityLow, nil, []string{id1.String(), id1.String()})
		task.RemoveAssignee(id1)
		assert.Empty(t, task.AssigneeIDs())
	})
}

// --- AssigneeIDs returns a copy ---

func TestTask_AssigneeIDs_ReturnsCopy(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	task := domain.ReconstructTask(domain.GenerateTaskID().String(), "title", "", domain.TaskStatusTodo, domain.PriorityLow, nil, []string{id.String()})

	ids := task.AssigneeIDs()
	ids[0] = domain.GenerateUserID() // mutate the copy

	assert.True(t, task.AssigneeIDs()[0].Equals(id), "internal slice must not be mutated")
}
