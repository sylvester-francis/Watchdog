package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StatusPageSubscriber is an email subscription to incident notifications
// for a specific public status page.
type StatusPageSubscriber struct {
	ID                     uuid.UUID
	StatusPageID           uuid.UUID
	Email                  string
	TokenHash              string     // SHA-256 hex of the plaintext token
	ConfirmedAt            *time.Time
	UnsubscribedAt         *time.Time
	LastConfirmationSentAt time.Time
	CreatedAt              time.Time
}

// GenerateStatusPageSubscriber creates a new subscriber row + returns the
// plaintext token used for confirmation + unsubscribe links. Token format:
// wd_sub_<32 hex chars>.
func GenerateStatusPageSubscriber(pageID uuid.UUID, email string) (*StatusPageSubscriber, string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate subscriber token: %w", err)
	}
	plaintext := "wd_sub_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plaintext))
	now := time.Now()
	return &StatusPageSubscriber{
		ID:                     uuid.New(),
		StatusPageID:           pageID,
		Email:                  email,
		TokenHash:              hex.EncodeToString(hash[:]),
		LastConfirmationSentAt: now,
		CreatedAt:              now,
	}, plaintext, nil
}

// HashStatusPageSubscriberToken returns the SHA-256 hex digest of a plaintext
// token, for repository lookup.
func HashStatusPageSubscriberToken(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(hash[:])
}

// IsActive reports whether the subscriber should currently receive emails.
func (s *StatusPageSubscriber) IsActive() bool {
	return s.ConfirmedAt != nil && s.UnsubscribedAt == nil
}
