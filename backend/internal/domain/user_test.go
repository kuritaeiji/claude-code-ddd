package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kuritaeiji/claude-code-ddd/internal/domain"
	apperrors "github.com/kuritaeiji/claude-code-ddd/pkg/errors"
)

func TestNewUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid uuid v4", uuid.NewString(), false},
		{"empty string", "", true},
		{"non-uuid", "not-a-uuid", true},
		{"truncated uuid", "550e8400-e29b-41d4-a716", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := domain.NewUserID(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.value, id.String())
		})
	}
}

func TestGenerateUserID_IsValid(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	_, err := uuid.Parse(id.String())
	assert.NoError(t, err)
}

func TestUserID_Equals(t *testing.T) {
	t.Parallel()

	value := uuid.NewString()
	a, err := domain.NewUserID(value)
	require.NoError(t, err)
	b, err := domain.NewUserID(value)
	require.NoError(t, err)
	c := domain.GenerateUserID()

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestNewEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"valid with subdomain", "user@mail.example.co.jp", false},
		{"empty", "", true},
		{"missing at-sign", "userexample.com", true},
		{"with display name (rejected)", "User <user@example.com>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			email, err := domain.NewEmail(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.value, email.String())
		})
	}
}

func TestEmail_Equals(t *testing.T) {
	t.Parallel()

	a, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	b, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	c, err := domain.NewEmail("other@example.com")
	require.NoError(t, err)

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestNewUserStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    domain.UserStatus
		wantErr bool
	}{
		{"active", "ACTIVE", domain.UserStatusActive, false},
		{"inactive", "INACTIVE", domain.UserStatusInactive, false},
		{"lowercase rejected", "active", "", true},
		{"unknown", "DELETED", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewUserStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserStatus_IsActive(t *testing.T) {
	t.Parallel()

	assert.True(t, domain.UserStatusActive.IsActive())
	assert.False(t, domain.UserStatusInactive.IsActive())
}

func TestReconstructUser(t *testing.T) {
	t.Parallel()

	id := domain.GenerateUserID()
	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)

	u := domain.ReconstructUser(id, email, "Bob", domain.UserStatusInactive)
	assert.True(t, id.Equals(u.ID()))
	assert.Equal(t, domain.UserStatusInactive, u.Status())
	assert.Empty(t, u.PullEvents(), "reconstruction must not enqueue events")
}

// 復元はリポジトリの DB 値を信頼するため、業務ルール（BR-U-02）の再検査を行わない。
func TestReconstructUser_SkipsDisplayNameValidation(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)

	t.Run("display name longer than 50 chars is accepted on reconstruction", func(t *testing.T) {
		t.Parallel()
		over := strings.Repeat("a", 60)
		u := domain.ReconstructUser(domain.GenerateUserID(), email, over, domain.UserStatusActive)
		assert.Equal(t, over, u.DisplayName())
	})

	t.Run("empty display name is accepted on reconstruction", func(t *testing.T) {
		t.Parallel()
		u := domain.ReconstructUser(domain.GenerateUserID(), email, "", domain.UserStatusActive)
		assert.Equal(t, "", u.DisplayName())
	})
}

func TestRestoreUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"valid uuid", uuid.NewString()},
		{"not a uuid (still accepted)", "not-a-uuid"},
		{"empty (still accepted)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id := domain.RestoreUserID(tt.value)
			assert.Equal(t, tt.value, id.String())
		})
	}
}

func TestRestoreEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"valid", "user@example.com"},
		{"missing at-sign (still accepted)", "userexample.com"},
		{"empty (still accepted)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			email := domain.RestoreEmail(tt.value)
			assert.Equal(t, tt.value, email.String())
		})
	}
}

func TestUser_Deactivate(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)

	t.Run("active to inactive enqueues event", func(t *testing.T) {
		t.Parallel()
		u := domain.ReconstructUser(domain.GenerateUserID(), email, "Alice", domain.UserStatusActive)

		require.NoError(t, u.Deactivate())

		assert.Equal(t, domain.UserStatusInactive, u.Status())
		events := u.PullEvents()
		require.Len(t, events, 1)
		evt, ok := events[0].(domain.UserDeactivated)
		require.True(t, ok, "queued event must be UserDeactivated")
		assert.True(t, u.ID().Equals(evt.UserID()))
		assert.Equal(t, "user.deactivated", evt.EventName())
	})

	t.Run("already-inactive user returns validation error (BR-U-03)", func(t *testing.T) {
		t.Parallel()
		u := domain.ReconstructUser(domain.GenerateUserID(), email, "Bob", domain.UserStatusInactive)

		err := u.Deactivate()

		require.Error(t, err)
		var ve *apperrors.ValidationError
		assert.ErrorAs(t, err, &ve, "must be ValidationError")
		assert.Equal(t, domain.UserStatusInactive, u.Status())
		assert.Empty(t, u.PullEvents(), "must not enqueue events on rejected deactivation")
	})

	t.Run("double deactivate rejects the second call", func(t *testing.T) {
		t.Parallel()
		u := domain.ReconstructUser(domain.GenerateUserID(), email, "Carol", domain.UserStatusActive)

		require.NoError(t, u.Deactivate())
		err := u.Deactivate()

		assert.Error(t, err)
		assert.Len(t, u.PullEvents(), 1, "only the first deactivation enqueues an event")
	})
}

func TestUser_PullEvents(t *testing.T) {
	t.Parallel()

	email, err := domain.NewEmail("user@example.com")
	require.NoError(t, err)
	u := domain.ReconstructUser(domain.GenerateUserID(), email, "Alice", domain.UserStatusActive)
	require.NoError(t, u.Deactivate())

	events := u.PullEvents()
	assert.Len(t, events, 1)

	again := u.PullEvents()
	assert.Empty(t, again, "PullEvents must clear the queue")
}
