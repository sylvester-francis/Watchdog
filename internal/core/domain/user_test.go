package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	email := "test@example.com"
	passwordHash := "hashed_password"

	user := NewUser(email, passwordHash)

	require.NotNil(t, user)
	assert.NotEqual(t, user.ID.String(), "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, email, user.Email)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.Nil(t, user.StripeID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.Equal(t, user.CreatedAt, user.UpdatedAt)
}

func TestUser_SetStripeID(t *testing.T) {
	user := NewUser("test@example.com", "hash")
	originalUpdatedAt := user.UpdatedAt

	stripeID := "cus_123456"
	user.SetStripeID(stripeID)

	require.NotNil(t, user.StripeID)
	assert.Equal(t, stripeID, *user.StripeID)
	assert.True(t, user.UpdatedAt.After(originalUpdatedAt) || user.UpdatedAt.Equal(originalUpdatedAt))
}

func TestUser_HasStripeID(t *testing.T) {
	tests := []struct {
		name     string
		stripeID *string
		want     bool
	}{
		{
			name:     "nil stripe ID",
			stripeID: nil,
			want:     false,
		},
		{
			name:     "empty stripe ID",
			stripeID: strPtr(""),
			want:     false,
		},
		{
			name:     "valid stripe ID",
			stripeID: strPtr("cus_123456"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := NewUser("test@example.com", "hash")
			user.StripeID = tt.stripeID

			got := user.HasStripeID()
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
