package query

import "time"

// TaskListItemDTO は Q-01 一覧取得で返す 1 件分のデータ。
// description を省いた軽量ビュー。AssigneeNames は担当ユーザーの display_name 一覧。
type TaskListItemDTO struct {
	ID            string
	Title         string
	Status        string
	Priority      string
	DueDate       *time.Time
	AssigneeNames []string
}

// TaskDetailDTO は Q-02 詳細取得で返すデータ。
// description を含む完全ビュー。AssigneeNames は担当ユーザーの display_name 一覧。
type TaskDetailDTO struct {
	ID            string
	Title         string
	Description   string
	Status        string
	Priority      string
	DueDate       *time.Time
	AssigneeNames []string
}

// TaskQueryRepository は CQRS Read 側専用のリポジトリインターフェース。
// ドメインモデルを経由せず、DB から直接 DTO に詰める（JOIN・集計を最適化）。
type TaskQueryRepository interface {
	// ListTasks はフィルタ条件に合うタスク一覧を返す。
	ListTasks(params ListTasksParams) ([]TaskListItemDTO, error)
	// GetTaskByID は指定 ID のタスク詳細を返す。未存在時は domain.ErrTaskNotFound を返す。
	GetTaskByID(id string) (*TaskDetailDTO, error)
}
