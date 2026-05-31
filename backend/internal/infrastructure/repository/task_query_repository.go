package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/usecase/query"
)

var _ query.TaskQueryRepository = (*taskQueryRepository)(nil)

type taskQueryRepository struct {
	db *gorm.DB
}

func NewTaskQueryRepository(db *gorm.DB) query.TaskQueryRepository {
	return &taskQueryRepository{db: db}
}

// taskQueryRow は tasks テーブルの 1 行に対応するスキャン先。
// gorm の Scan 用に公開フィールドを使う（domain モデルは不要）。
type taskQueryRow struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
}

// assigneeNameRow は task_assignees JOIN users の 1 行。
type assigneeNameRow struct {
	TaskID      string
	DisplayName string
}

// ListTasks は tasks テーブルをフィルタして取得し、担当ユーザー名を 2 クエリで付与して返す（N+1 回避）。
func (r *taskQueryRepository) ListTasks(params query.ListTasksParams) ([]query.TaskListItemDTO, error) {
	q := r.db.Table("tasks").
		Select("id, title, status, priority, due_date")

	if len(params.Statuses) > 0 {
		q = q.Where("status IN ?", params.Statuses)
	}
	if len(params.Priorities) > 0 {
		q = q.Where("priority IN ?", params.Priorities)
	}

	var rows []taskQueryRow
	if err := q.Order("created_at DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []query.TaskListItemDTO{}, nil
	}

	taskIDs := make([]string, len(rows))
	for i, row := range rows {
		taskIDs[i] = row.ID
	}

	namesByTask, err := r.fetchAssigneeNames(taskIDs)
	if err != nil {
		return nil, err
	}

	dtos := make([]query.TaskListItemDTO, len(rows))
	for i, row := range rows {
		names := namesByTask[row.ID]
		if names == nil {
			names = []string{}
		}
		dtos[i] = query.TaskListItemDTO{
			ID:            row.ID,
			Title:         row.Title,
			Status:        row.Status,
			Priority:      row.Priority,
			DueDate:       row.DueDate,
			AssigneeNames: names,
		}
	}
	return dtos, nil
}

// GetTaskByID は主キーで 1 件取得し、担当ユーザー名を付与して返す。
// 未存在時は domain.ErrTaskNotFound を返す。
func (r *taskQueryRepository) GetTaskByID(id string) (*query.TaskDetailDTO, error) {
	var row taskQueryRow
	if err := r.db.Table("tasks").
		Select("id, title, description, status, priority, due_date").
		Where("id = ?", id).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}

	namesByTask, err := r.fetchAssigneeNames([]string{id})
	if err != nil {
		return nil, err
	}
	names := namesByTask[id]
	if names == nil {
		names = []string{}
	}

	return &query.TaskDetailDTO{
		ID:            row.ID,
		Title:         row.Title,
		Description:   row.Description,
		Status:        row.Status,
		Priority:      row.Priority,
		DueDate:       row.DueDate,
		AssigneeNames: names,
	}, nil
}

// fetchAssigneeNames は task_id のスライスに対して担当ユーザー名を一括取得し、
// task_id → []display_name のマップを返す（N+1 回避）。
func (r *taskQueryRepository) fetchAssigneeNames(taskIDs []string) (map[string][]string, error) {
	var rows []assigneeNameRow
	if err := r.db.Table("task_assignees").
		Select("task_assignees.task_id, users.display_name").
		Joins("JOIN users ON users.id = task_assignees.user_id").
		Where("task_assignees.task_id IN ?", taskIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string][]string, len(taskIDs))
	for _, row := range rows {
		result[row.TaskID] = append(result[row.TaskID], row.DisplayName)
	}
	return result, nil
}
