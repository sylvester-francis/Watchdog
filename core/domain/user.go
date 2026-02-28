package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UsernameRegex validates usernames: 3-50 chars, lowercase alphanumeric + hyphens, no leading/trailing hyphens.
var UsernameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// Plan represents a user's account type.
type Plan string

const (
	PlanBeta Plan = "beta"
)

// PlanLimits defines the resource limits for a plan.
type PlanLimits struct {
	MaxAgents   int // -1 = unlimited
	MaxMonitors int // -1 = unlimited
}

// Limits returns the resource limits for the plan.
func (p Plan) Limits() PlanLimits {
	return PlanLimits{MaxAgents: 10, MaxMonitors: -1}
}

// IsValid returns true if the plan is a recognized type.
func (p Plan) IsValid() bool {
	return p == PlanBeta
}

// String returns the display name for the plan.
func (p Plan) String() string {
	return "Beta"
}

// User represents a registered user in the system.
type User struct {
	ID                uuid.UUID
	Email             string
	Username          string
	PasswordHash      string
	Plan              Plan
	IsAdmin           bool
	TenantID          string
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UsernameFromEmail generates a username from an email address.
func UsernameFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	// Lowercase and replace non-alphanumeric chars with hyphens
	username := strings.ToLower(local)
	username = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(username, "-")
	username = strings.Trim(username, "-")
	if len(username) < 3 {
		username = username + "-user"
	}
	if len(username) > 50 {
		username = username[:50]
	}
	return username
}

// IsValidUsername returns true if the username matches the required format.
func IsValidUsername(username string) bool {
	return UsernameRegex.MatchString(username)
}

// NewUser creates a new User with generated ID and timestamps.
func NewUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     UsernameFromEmail(email),
		PasswordHash: passwordHash,
		Plan:         PlanBeta,
		TenantID:     "default",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

