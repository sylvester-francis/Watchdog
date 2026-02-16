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

func TestPlanBeta_Limits(t *testing.T) {
	limits := PlanBeta.Limits()

	assert.Equal(t, 10, limits.MaxAgents, "Beta plan should allow 10 agents")
	assert.Equal(t, -1, limits.MaxMonitors, "Beta plan should allow unlimited monitors")
}

func TestPlanBeta_IsValid(t *testing.T) {
	tests := []struct {
		plan Plan
		want bool
	}{
		{PlanBeta, true},
		{PlanFree, true},
		{PlanPro, true},
		{PlanTeam, true},
		{Plan("invalid"), false},
		{Plan(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			got := tt.plan.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlanBeta_String(t *testing.T) {
	tests := []struct {
		plan Plan
		want string
	}{
		{PlanBeta, "Beta"},
		{PlanFree, "Free"},
		{PlanPro, "Pro"},
		{PlanTeam, "Team"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.plan.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewUser_DefaultsBeta(t *testing.T) {
	user := NewUser("beta@example.com", "hashed_password")

	require.NotNil(t, user)
	assert.Equal(t, PlanBeta, user.Plan, "NewUser should default to PlanBeta")
	assert.Equal(t, "Beta", user.Plan.String())

	limits := user.Plan.Limits()
	assert.Equal(t, 10, limits.MaxAgents)
	assert.Equal(t, -1, limits.MaxMonitors)
}

func strPtr(s string) *string {
	return &s
}
