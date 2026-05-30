package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
)

func TestToModel(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	id := domain.GenerateUserID()
	u := domain.ReconstructUser(id.String(), email.String(), "Alice", domain.UserStatusActive)

	m := toModel(u)

	assert.Equal(t, id.String(), m.ID)
	assert.Equal(t, "user@example.com", m.Email)
	assert.Equal(t, "Alice", m.DisplayName)
	assert.Equal(t, string(domain.UserStatusActive), m.Status)
}

func TestToDomain(t *testing.T) {
	t.Parallel()

	idStr := uuid.NewString()
	m := &userModel{
		ID:          idStr,
		Email:       "user@example.com",
		DisplayName: "Alice",
		Status:      string(domain.UserStatusInactive),
	}

	u := toDomain(m)

	assert.Equal(t, idStr, u.ID().String())
	assert.Equal(t, "user@example.com", u.Email().String())
	assert.Equal(t, "Alice", u.DisplayName())
	assert.Equal(t, domain.UserStatusInactive, u.Status())
	assert.Empty(t, u.PullEvents(), "restoration must not enqueue events")
}

func TestUserModel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users", userModel{}.TableName())
}
