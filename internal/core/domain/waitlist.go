package domain

import (
	"time"

	"github.com/google/uuid"
)

// WaitlistSignup represents an email signup for the beta waitlist.
type WaitlistSignup struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}
