package query

import (
	"errors"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

type GetTaskParams struct {
	ID string
}

// GetTaskQuery は Q-02（タスク詳細取得）のオーケストレーション。
// UUID 検証 → リポジトリ呼び出し → NotFound 変換のみを担う。
type GetTaskQuery struct {
	repo TaskQueryRepository
}

func NewGetTaskQuery(repo TaskQueryRepository) *GetTaskQuery {
	return &GetTaskQuery{repo: repo}
}

func (q *GetTaskQuery) Execute(params GetTaskParams) (*TaskDetailDTO, error) {
	id, err := domain.NewTaskID(params.ID)
	if err != nil {
		return nil, err
	}

	dto, err := q.repo.GetTaskByID(id.String())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, apperrors.NewNotFoundError("not_found.task", nil)
		}
		return nil, err
	}
	return dto, nil
}
