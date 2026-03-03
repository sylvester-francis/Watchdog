package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Recurrence values for maintenance windows.
const (
	RecurrenceOnce    = "once"
	RecurrenceDaily   = "daily"
	RecurrenceWeekly  = "weekly"
	RecurrenceMonthly = "monthly"
)

// MaintenanceWindow represents a scheduled maintenance period for an agent.
// During an active window, agent-offline incidents are suppressed.
type MaintenanceWindow struct {
	ID         uuid.UUID
	AgentID    uuid.UUID
	UserID     uuid.UUID
	Name       string
	StartsAt   time.Time
	EndsAt     time.Time
	Recurrence string
	TenantID   string
	CreatedAt  time.Time
}

// NewMaintenanceWindow creates a new MaintenanceWindow with defaults.
func NewMaintenanceWindow(agentID, userID uuid.UUID, name string, startsAt, endsAt time.Time) *MaintenanceWindow {
	return &MaintenanceWindow{
		ID:         uuid.New(),
		AgentID:    agentID,
		UserID:     userID,
		Name:       name,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		Recurrence: RecurrenceOnce,
		CreatedAt:  time.Now(),
	}
}

// ValidRecurrence returns true if the recurrence value is valid.
func ValidRecurrence(r string) bool {
	switch r {
	case RecurrenceOnce, RecurrenceDaily, RecurrenceWeekly, RecurrenceMonthly:
		return true
	default:
		return false
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
	if mw.Recurrence == "" {
		mw.Recurrence = RecurrenceOnce
	}
	if !ValidRecurrence(mw.Recurrence) {
		return fmt.Errorf("invalid recurrence: %s", mw.Recurrence)
	}
	return nil
}

// NextOccurrence returns a new MaintenanceWindow shifted forward by one recurrence interval.
// Returns nil for non-recurring windows.
func (mw *MaintenanceWindow) NextOccurrence() *MaintenanceWindow {
	duration := mw.EndsAt.Sub(mw.StartsAt)
	var nextStart time.Time

	switch mw.Recurrence {
	case RecurrenceDaily:
		nextStart = mw.StartsAt.AddDate(0, 0, 1)
	case RecurrenceWeekly:
		nextStart = mw.StartsAt.AddDate(0, 0, 7)
	case RecurrenceMonthly:
		nextStart = mw.StartsAt.AddDate(0, 1, 0)
	default:
		return nil
	}

	return &MaintenanceWindow{
		ID:         uuid.New(),
		AgentID:    mw.AgentID,
		UserID:     mw.UserID,
		Name:       mw.Name,
		StartsAt:   nextStart,
		EndsAt:     nextStart.Add(duration),
		Recurrence: mw.Recurrence,
		TenantID:   mw.TenantID,
		CreatedAt:  time.Now(),
	}
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
