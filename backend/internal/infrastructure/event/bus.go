// Package event は Domain 層で定義された EventBus インターフェースのインメモリ実装を提供する。
package event

import (
	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// InMemoryBus は登録された EventHandler を EventName でグルーピングして保持し、
// Publish 時に一致するハンドラを登録順に同期呼び出しするインメモリ EventBus。
// 外部メッセージブローカは使わず、学習目的のシンプルな同期実装にとどめる。
type InMemoryBus struct {
	handlers map[string][]domain.EventHandler
}

var _ domain.EventBus = (*InMemoryBus)(nil)

func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{handlers: make(map[string][]domain.EventHandler)}
}

// Subscribe は handler を EventName() で示すイベントの購読者として登録する。
// DI（cmd/api/main.go）から呼ばれる想定。
func (b *InMemoryBus) Subscribe(handler domain.EventHandler) {
	name := handler.EventName()
	b.handlers[name] = append(b.handlers[name], handler)
}

// Publish は event の EventName に一致するハンドラを登録順に呼び出す。
// ハンドラ未登録のイベントは no-op（エラーにしない）。
// いずれかのハンドラがエラーを返した時点で残りを中断してそのエラーを返す。
func (b *InMemoryBus) Publish(event domain.DomainEvent) error {
	for _, handler := range b.handlers[event.EventName()] {
		if err := handler.Handle(event); err != nil {
			return err
		}
	}
	return nil
}
