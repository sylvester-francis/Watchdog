package domain

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a user's subscription tier.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
	PlanTeam Plan = "team"
)

// PlanLimits defines the resource limits for a plan.
type PlanLimits struct {
	MaxAgents   int // -1 = unlimited
	MaxMonitors int // -1 = unlimited
}

// Limits returns the resource limits for the plan.
func (p Plan) Limits() PlanLimits {
	switch p {
	case PlanPro:
		return PlanLimits{MaxAgents: 3, MaxMonitors: 25}
	case PlanTeam:
		return PlanLimits{MaxAgents: 10, MaxMonitors: -1}
	default:
		return PlanLimits{MaxAgents: 1, MaxMonitors: 3}
	}
}

// IsValid returns true if the plan is a recognized tier.
func (p Plan) IsValid() bool {
	switch p {
	case PlanFree, PlanPro, PlanTeam:
		return true
	default:
		return false
	}
}

// String returns the display name for the plan.
func (p Plan) String() string {
	switch p {
	case PlanPro:
		return "Pro"
	case PlanTeam:
		return "Team"
	default:
		return "Free"
	}
}

// User represents a registered user in the system.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Plan         Plan
	StripeID     *string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser creates a new User with generated ID and timestamps.
func NewUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Plan:         PlanFree,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetStripeID sets the Stripe customer ID.
func (u *User) SetStripeID(stripeID string) {
	u.StripeID = &stripeID
	u.UpdatedAt = time.Now()
}

// HasStripeID returns true if the user has a Stripe customer ID.
func (u *User) HasStripeID() bool {
	return u.StripeID != nil && *u.StripeID != ""
}
