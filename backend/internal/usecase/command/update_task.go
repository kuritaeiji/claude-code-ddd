package command

import (
	"errors"
	"time"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

type UpdateTaskParams struct {
	ID          string
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
	AssigneeIDs []string
}

// UpdateTaskCommand は T-02（タスク更新）のオーケストレーション。
// ID 検証 → 取得（未存在は 404）→ 担当者解決 → domain.Task.Update による一括更新 → 永続化 → DTO 変換を順に行う。
// ビジネスルール（BR-T-02〜06）は domain.Task.Update に閉じており、本コマンドは検査しない。
type UpdateTaskCommand struct {
	tasks domain.TaskRepository
	users domain.UserRepository
}

func NewUpdateTaskCommand(tasks domain.TaskRepository, users domain.UserRepository) *UpdateTaskCommand {
	return &UpdateTaskCommand{tasks: tasks, users: users}
}

func (c *UpdateTaskCommand) Execute(params UpdateTaskParams) (TaskDTO, error) {
	id, err := domain.NewTaskID(params.ID)
	if err != nil {
		return TaskDTO{}, err
	}

	task, err := c.tasks.FindByID(id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return TaskDTO{}, apperrors.NewNotFoundError("not_found.task", nil)
		}
		return TaskDTO{}, err
	}

	assignees, err := resolveAssignees(c.users, params.AssigneeIDs)
	if err != nil {
		return TaskDTO{}, err
	}

	if err := task.Update(params.Title, params.Description, params.Status, params.Priority, params.DueDate, assignees); err != nil {
		return TaskDTO{}, err
	}

	if err := c.tasks.Update(task); err != nil {
		return TaskDTO{}, err
	}

	return newTaskDTO(task), nil
}
