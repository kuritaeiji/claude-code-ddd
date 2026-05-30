package command

import (
	"time"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

type CreateTaskParams struct {
	Title       string
	Description string
	Priority    string
	DueDate     *time.Time
	AssigneeIDs []string
}

// TaskDTO はタスクコマンドの結果を Controller に返す共通 DTO。
// DueDate は期日未設定を nil で表す。日付→文字列のワイヤーフォーマット変換は Controller 層の責務とし、
// ここでは time.Time のまま返す（入力側の parseDueDate と対称）。
type TaskDTO struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
	AssigneeIDs []string
}

// newTaskDTO は *domain.Task を TaskDTO に変換する。Create/Update コマンドで共用する。
func newTaskDTO(t *domain.Task) TaskDTO {
	var dueDate *time.Time
	if d := t.DueDate(); d != nil {
		v := d.Value()
		dueDate = &v
	}
	ids := t.AssigneeIDs()
	assigneeIDs := make([]string, len(ids))
	for i, id := range ids {
		assigneeIDs[i] = id.String()
	}
	return TaskDTO{
		ID:          t.ID().String(),
		Title:       t.Title(),
		Description: t.Description(),
		Status:      string(t.Status()),
		Priority:    string(t.Priority()),
		DueDate:     dueDate,
		AssigneeIDs: assigneeIDs,
	}
}

// CreateTaskCommand は T-01（タスク作成）のオーケストレーション。
// 担当者 ID の解決（resolveAssignees）→ domain.NewTask による検証付き生成 → 永続化 → DTO 変換のみを担う。
// ビジネスルール（BR-T-01,03〜06）は domain.NewTask に閉じており、本コマンドは検査しない。
type CreateTaskCommand struct {
	tasks domain.TaskRepository
	users domain.UserRepository
}

func NewCreateTaskCommand(tasks domain.TaskRepository, users domain.UserRepository) *CreateTaskCommand {
	return &CreateTaskCommand{tasks: tasks, users: users}
}

func (c *CreateTaskCommand) Execute(params CreateTaskParams) (TaskDTO, error) {
	assignees, err := resolveAssignees(c.users, params.AssigneeIDs)
	if err != nil {
		return TaskDTO{}, err
	}

	task, err := domain.NewTask(params.Title, params.Description, params.Priority, params.DueDate, assignees)
	if err != nil {
		return TaskDTO{}, err
	}

	if err := c.tasks.Insert(task); err != nil {
		return TaskDTO{}, err
	}

	return newTaskDTO(task), nil
}
