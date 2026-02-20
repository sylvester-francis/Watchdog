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
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.Equal(t, user.CreatedAt, user.UpdatedAt)
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
	assert.Equal(t, "Beta", PlanBeta.String())
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

func TestUsernameFromEmail(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"alice@example.com", "alice"},
		{"Bob.Smith@gmail.com", "bob-smith"},
		{"tech+dev@company.io", "tech-dev"},
		{"a@x.com", "a-user"},
		{"UPPER@CASE.COM", "upper"},
		{"dots.and.more@test.com", "dots-and-more"},
		{"special!#chars@test.com", "special-chars"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := UsernameFromEmail(tt.email)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		username string
		valid    bool
	}{
		{"alice", true},
		{"bob-smith", true},
		{"a1b2c3", true},
		{"my-cool-username", true},
		{"ab", false},         // too short
		{"-leading", false},   // leading hyphen
		{"trailing-", false},  // trailing hyphen
		{"UPPERCASE", false},  // must be lowercase
		{"has space", false},  // no spaces
		{"has_under", false},  // no underscores
		{"ok", false},         // 2 chars, too short
		{"abc", true},         // exactly 3 chars
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			got := IsValidUsername(tt.username)
			assert.Equal(t, tt.valid, got)
		})
	}
}

func TestNewUser_GeneratesUsername(t *testing.T) {
	user := NewUser("techwithsyl@gmail.com", "hash")
	assert.Equal(t, "techwithsyl", user.Username)
}

func strPtr(s string) *string {
	return &s
}

