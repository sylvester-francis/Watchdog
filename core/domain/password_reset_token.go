package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PasswordResetTTL is the lifetime of a password reset token.
const PasswordResetTTL = 30 * time.Minute

// PasswordResetToken represents a single-use password reset token.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string     // SHA-256 hex of the plaintext (never store plaintext)
	ExpiresAt time.Time
	UsedAt    *time.Time // nil until consumed
	IPAddress string     // IP that requested the reset (audit trail)
	CreatedAt time.Time
}

// GeneratePasswordResetToken creates a new token and returns the entity plus the
// plaintext token (shown once in the reset email link). Format: wd_pr_<32 hex>.
func GeneratePasswordResetToken(userID uuid.UUID, ipAddress string) (*PasswordResetToken, string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	plaintext := "wd_pr_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plaintext))
	return &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hex.EncodeToString(hash[:]),
		ExpiresAt: time.Now().Add(PasswordResetTTL),
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}, plaintext, nil
}

// HashPasswordResetToken returns the SHA-256 hex digest of a plaintext token.
// Used by the repository to look up tokens by their hash.
func HashPasswordResetToken(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(hash[:])
}

// IsValid reports whether the token can still be consumed.
func (t *PasswordResetToken) IsValid() bool {
	if t.UsedAt != nil {
		return false
	}
	return time.Now().Before(t.ExpiresAt)
}
