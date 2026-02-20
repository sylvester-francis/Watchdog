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

// Agent represents a monitoring agent deployed in a private network.
type Agent struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	APIKeyEncrypted []byte
	LastSeenAt      *time.Time
	Status          AgentStatus
	CreatedAt       time.Time
}

// NewAgent creates a new Agent with generated ID and offline status.
func NewAgent(userID uuid.UUID, name string, apiKeyEncrypted []byte) *Agent {
	return &Agent{
		ID:              uuid.New(),
		UserID:          userID,
		Name:            name,
		APIKeyEncrypted: apiKeyEncrypted,
		Status:          AgentStatusOffline,
		CreatedAt:       time.Now(),
	}
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
