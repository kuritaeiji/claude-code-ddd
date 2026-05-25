package event_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	"github.com/kuritaeiji/claude-code-ddd/internal/infrastructure/event"
)

// stubEvent は任意の EventName を持つテスト用ドメインイベント。
type stubEvent struct {
	name string
}

func (e stubEvent) EventName() string     { return e.name }
func (e stubEvent) OccurredAt() time.Time { return time.Now() }

// recordingHandler は受け取ったイベントを記録するテスト用ハンドラ。
type recordingHandler struct {
	name     string
	received []domain.DomainEvent
	err      error
}

func (h *recordingHandler) EventName() string { return h.name }
func (h *recordingHandler) Handle(event domain.DomainEvent) error {
	h.received = append(h.received, event)
	return h.err
}

func TestInMemoryBus_Publish_DispatchesToMatchingHandlers(t *testing.T) {
	t.Parallel()

	bus := event.NewInMemoryBus()
	h1 := &recordingHandler{name: "user.deactivated"}
	h2 := &recordingHandler{name: "user.deactivated"}
	other := &recordingHandler{name: "task.created"}
	bus.Subscribe(h1)
	bus.Subscribe(h2)
	bus.Subscribe(other)

	evt := stubEvent{name: "user.deactivated"}
	require.NoError(t, bus.Publish(evt))

	assert.Len(t, h1.received, 1)
	assert.Len(t, h2.received, 1)
	assert.Empty(t, other.received, "EventName が一致しないハンドラは呼ばれない")
}

func TestInMemoryBus_Publish_NoHandlerIsNoOp(t *testing.T) {
	t.Parallel()

	bus := event.NewInMemoryBus()
	assert.NoError(t, bus.Publish(stubEvent{name: "user.deactivated"}))
}

func TestInMemoryBus_Publish_ReturnsHandlerError(t *testing.T) {
	t.Parallel()

	bus := event.NewInMemoryBus()
	wantErr := errors.New("handler failed")
	failing := &recordingHandler{name: "user.deactivated", err: wantErr}
	after := &recordingHandler{name: "user.deactivated"}
	bus.Subscribe(failing)
	bus.Subscribe(after)

	err := bus.Publish(stubEvent{name: "user.deactivated"})

	require.ErrorIs(t, err, wantErr)
	assert.Empty(t, after.received, "先行ハンドラがエラーを返したら後続は呼ばれない")
}
