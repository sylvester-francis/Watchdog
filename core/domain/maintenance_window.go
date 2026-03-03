package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MaintenanceWindow represents a scheduled maintenance period for an agent.
// During an active window, agent-offline incidents are suppressed.
type MaintenanceWindow struct {
	ID        uuid.UUID
	AgentID   uuid.UUID
	UserID    uuid.UUID
	Name      string
	StartsAt  time.Time
	EndsAt    time.Time
	TenantID  string
	CreatedAt time.Time
}

// NewMaintenanceWindow creates a new MaintenanceWindow with defaults.
func NewMaintenanceWindow(agentID, userID uuid.UUID, name string, startsAt, endsAt time.Time) *MaintenanceWindow {
	return &MaintenanceWindow{
		ID:        uuid.New(),
		AgentID:   agentID,
		UserID:    userID,
		Name:      name,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		CreatedAt: time.Now(),
	}
}

// Validate checks that the maintenance window fields are valid.
func (mw *MaintenanceWindow) Validate() error {
	if mw.Name == "" {
		return fmt.Errorf("maintenance window name is required")
	}
	if !mw.EndsAt.After(mw.StartsAt) {
		return fmt.Errorf("end time must be after start time")
	}
	return nil
}

// IsActive returns true if the window is currently active.
func (mw *MaintenanceWindow) IsActive() bool {
	now := time.Now()
	return !now.Before(mw.StartsAt) && now.Before(mw.EndsAt)
}

// IsExpired returns true if the window has ended.
func (mw *MaintenanceWindow) IsExpired() bool {
	return time.Now().After(mw.EndsAt)
}

// IsFuture returns true if the window hasn't started yet.
func (mw *MaintenanceWindow) IsFuture() bool {
	return time.Now().Before(mw.StartsAt)
}
