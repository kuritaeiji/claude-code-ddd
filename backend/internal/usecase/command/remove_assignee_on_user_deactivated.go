package command

import (
	"fmt"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// RemoveAssigneeOnUserDeactivated は UserDeactivated イベントの購読ハンドラ。
// 非活性化されたユーザーを、その担当する全タスクの担当者一覧から除外する
// （domain.md「UserDeactivated の副作用」/ BR-T-04 の結果整合側）。
// EventHandler は Usecase 層に実装し、cmd/api/main.go で EventBus に Subscribe する。
type RemoveAssigneeOnUserDeactivated struct {
	tasks domain.TaskRepository
}

// 実装漏れをビルド時に検出するためのアサーション。
var _ domain.EventHandler = (*RemoveAssigneeOnUserDeactivated)(nil)

func NewRemoveAssigneeOnUserDeactivated(tasks domain.TaskRepository) *RemoveAssigneeOnUserDeactivated {
	return &RemoveAssigneeOnUserDeactivated{tasks: tasks}
}

// EventName は購読対象を UserDeactivated に固定する。
// Subscribe 登録キーと Publish dispatch キーの一致を保証するため、マジック文字列を持たず
// イベント型の EventName() を単一の真実の源として委譲する。
func (h *RemoveAssigneeOnUserDeactivated) EventName() string {
	return domain.UserDeactivated{}.EventName()
}

// Handle は当該ユーザーが担当する全タスクから担当者を除外して永続化する。
// 担当タスクが 0 件なら何もしない。除外・永続化のいずれかが失敗したら即座に中断して返す
// （InMemoryBus が Publish のエラーとして上位へ伝播する）。
func (h *RemoveAssigneeOnUserDeactivated) Handle(event domain.DomainEvent) error {
	e, ok := event.(domain.UserDeactivated)
	if !ok {
		// EventName 一致でのみ dispatch されるため通常は到達しない。誤登録の早期検知のためエラーにする。
		return fmt.Errorf("RemoveAssigneeOnUserDeactivated: unexpected event type %T", event)
	}

	tasks, err := h.tasks.FindByAssigneeID(e.UserID())
	if err != nil {
		return err
	}
	for _, task := range tasks {
		task.RemoveAssignee(e.UserID())
		if err := h.tasks.Update(task); err != nil {
			return err
		}
	}
	return nil
}
