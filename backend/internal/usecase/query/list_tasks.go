package query

import "github.com/kuritaeiji/claude-code-ddd/internal/domain"

// ListTasksParams はタスク一覧取得のフィルタ条件。各フィールドは空スライスで未指定（フィルタなし）を表す。
// 同一フィールド内の複数値は OR、フィールド間は AND として扱う。
type ListTasksParams struct {
	Statuses   []string
	Priorities []string
}

// ListTasksQuery は Q-01（タスク一覧取得）のオーケストレーション。
// status / priority の列挙値検証のみを担い、実際のデータ取得は TaskQueryRepository に委ねる。
type ListTasksQuery struct {
	repo TaskQueryRepository
}

func NewListTasksQuery(repo TaskQueryRepository) *ListTasksQuery {
	return &ListTasksQuery{repo: repo}
}

func (q *ListTasksQuery) Execute(params ListTasksParams) ([]TaskListItemDTO, error) {
	for _, s := range params.Statuses {
		if _, err := domain.NewTaskStatus(s); err != nil {
			return nil, err
		}
	}
	for _, p := range params.Priorities {
		if _, err := domain.NewPriority(p); err != nil {
			return nil, err
		}
	}
	return q.repo.ListTasks(params)
}
