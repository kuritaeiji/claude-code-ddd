package command

import (
	"errors"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

type DeleteTaskParams struct {
	ID string
}

// DeleteTaskCommand は T-03（タスク削除）のオーケストレーション。
// ID 検証 → 存在確認（未存在は 404）→ 削除を行う。
// 存在確認を挟むのは、未存在の削除を 204 ではなく 404 として明示するため（DeactivateUser と同じ方針）。
type DeleteTaskCommand struct {
	tasks domain.TaskRepository
}

func NewDeleteTaskCommand(tasks domain.TaskRepository) *DeleteTaskCommand {
	return &DeleteTaskCommand{tasks: tasks}
}

func (c *DeleteTaskCommand) Execute(params DeleteTaskParams) error {
	id, err := domain.NewTaskID(params.ID)
	if err != nil {
		return err
	}

	if _, err := c.tasks.FindByID(id); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return apperrors.NewNotFoundError("not_found.task", nil)
		}
		return err
	}

	return c.tasks.Delete(id)
}
