package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

func TestUserDeactivated(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	before := time.Now()
	evt := domain.NewUserDeactivated(id)
	after := time.Now()

	assert.True(t, id.Equals(evt.UserID()))
	assert.Equal(t, "user.deactivated", evt.EventName())
	assert.False(t, evt.OccurredAt().Before(before))
	assert.False(t, evt.OccurredAt().After(after))
}

func TestUserDeactivated_ImplementsDomainEvent(t *testing.T) {
	t.Parallel()

	var _ domain.DomainEvent = domain.NewUserDeactivated(domain.GenerateUserID())
}
