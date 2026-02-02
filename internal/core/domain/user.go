package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a registered user in the system.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	StripeID     *string
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
