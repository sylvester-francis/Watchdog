package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TokenScope defines the permission level of an API token.
type TokenScope string

const (
	TokenScopeAdmin    TokenScope = "admin"     // Full read/write access
	TokenScopeReadOnly TokenScope = "read_only" // Read-only access
)

// IsValid checks if the scope is a valid TokenScope.
func (s TokenScope) IsValid() bool {
	switch s {
	case TokenScopeAdmin, TokenScopeReadOnly:
		return true
	default:
		return false
	}
}

// APIToken represents a user's API token for programmatic access.
type APIToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	TokenHash  string
	Prefix     string
	Scope      TokenScope
	LastUsedAt *time.Time
	LastUsedIP *string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

// GenerateAPIToken creates a new APIToken and returns the plaintext token (shown once).
// Token format: wd_<32 hex chars> (e.g. wd_a1b2c3d4e5f6789012345678901234ab)
func GenerateAPIToken(userID uuid.UUID, name string, expiresAt *time.Time, scope TokenScope) (*APIToken, string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	plaintext := "wd_" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plaintext))
	hashHex := hex.EncodeToString(hash[:])
	prefix := plaintext[:11] // "wd_" + first 8 hex chars

	if !scope.IsValid() {
		scope = TokenScopeAdmin
	}

	token := &APIToken{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		TokenHash: hashHex,
		Prefix:    prefix,
		Scope:     scope,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return token, plaintext, nil
}

// HashToken returns the SHA-256 hex digest of a plaintext token.
func HashToken(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(hash[:])
}

// IsExpired returns true if the token has an expiry and it has passed.
func (t *APIToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}
