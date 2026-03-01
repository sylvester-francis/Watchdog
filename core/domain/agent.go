package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// AgentStatus represents the connection status of an agent.
type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

// IsValid checks if the status is a valid AgentStatus.
func (s AgentStatus) IsValid() bool {
	switch s {
	case AgentStatusOnline, AgentStatusOffline:
		return true
	default:
		return false
	}
}

// DefaultAPIKeyExpiryDays is the default validity period for newly created
// agent API keys (H-023). Set to 365 days (1 year).
const DefaultAPIKeyExpiryDays = 365

// Agent represents a monitoring agent deployed in a private network.
type Agent struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	Name                  string
	APIKeyEncrypted       []byte
	APIKeyExpiresAt       *time.Time // H-023: optional key expiry (nil = never)
	LastSeenAt            *time.Time
	Status                AgentStatus
	Fingerprint           map[string]string
	FingerprintVerifiedAt *time.Time
	TenantID              string
	CreatedAt             time.Time
}

// NewAgent creates a new Agent with generated ID, offline status, and a
// default API key expiry of DefaultAPIKeyExpiryDays from now (H-023).
func NewAgent(userID uuid.UUID, name string, apiKeyEncrypted []byte) *Agent {
	expiresAt := time.Now().AddDate(0, 0, DefaultAPIKeyExpiryDays)
	return &Agent{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            name,
		APIKeyEncrypted: apiKeyEncrypted,
		APIKeyExpiresAt: &expiresAt,
		Status:          AgentStatusOffline,
		CreatedAt:       time.Now(),
	}
}

// IsAPIKeyExpired returns true if the agent's API key has an expiry and it has
// passed (H-023).
func (a *Agent) IsAPIKeyExpired() bool {
	if a.APIKeyExpiresAt == nil {
		return false // nil = never expires (legacy agents)
	}
	return time.Now().After(*a.APIKeyExpiresAt)
}

// MarkOnline marks the agent as online and updates last seen time.
func (a *Agent) MarkOnline() {
	now := time.Now()
	a.LastSeenAt = &now
	a.Status = AgentStatusOnline
}

// MarkOffline marks the agent as offline.
func (a *Agent) MarkOffline() {
	a.Status = AgentStatusOffline
}

// IsOnline returns true if the agent is currently online.
func (a *Agent) IsOnline() bool {
	return a.Status == AgentStatusOnline
}

// UpdateLastSeen updates the last seen timestamp.
func (a *Agent) UpdateLastSeen() {
	now := time.Now()
	a.LastSeenAt = &now
}

// GenerateAPIKey generates a new random API key (32 bytes, hex encoded = 64 chars).
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
