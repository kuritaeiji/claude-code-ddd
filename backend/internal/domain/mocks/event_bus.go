package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

// EventBus インターフェースの実装漏れをビルド時に検出するためのアサーション。
var _ domain.EventBus = (*EventBus)(nil)

type EventBus struct {
	mock.Mock
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (m *EventBus) Publish(event domain.DomainEvent) error {
	args := m.Called(event)
	return args.Error(0)
}
