package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// IncidentStatus represents the status of an incident.
type IncidentStatus string

const (
	IncidentStatusOpen         IncidentStatus = "open"
	IncidentStatusAcknowledged IncidentStatus = "acknowledged"
	IncidentStatusResolved     IncidentStatus = "resolved"
)

// IsValid checks if the status is a valid IncidentStatus.
func (s IncidentStatus) IsValid() bool {
	switch s {
	case IncidentStatusOpen, IncidentStatusAcknowledged, IncidentStatusResolved:
		return true
	default:
		return false
	}
}

// IsActive returns true if the incident is still active (not resolved).
func (s IncidentStatus) IsActive() bool {
	return s == IncidentStatusOpen || s == IncidentStatusAcknowledged
}

// Errors for incident state transitions.
var (
	ErrIncidentAlreadyResolved     = errors.New("incident is already resolved")
	ErrIncidentAlreadyAcknowledged = errors.New("incident is already acknowledged")
	ErrIncidentNotAcknowledged     = errors.New("incident must be acknowledged before resolving")
)

// Incident represents a monitoring incident (downtime event).
type Incident struct {
	ID             uuid.UUID
	MonitorID      uuid.UUID
	StartedAt      time.Time
	ResolvedAt     *time.Time
	TTRSeconds     *int
	AcknowledgedBy *uuid.UUID
	AcknowledgedAt *time.Time
	Status         IncidentStatus
	CreatedAt      time.Time
}

// NewIncident creates a new open incident.
func NewIncident(monitorID uuid.UUID) *Incident {
	now := time.Now()
	return &Incident{
		ID:        uuid.New(),
		MonitorID: monitorID,
		StartedAt: now,
		Status:    IncidentStatusOpen,
		CreatedAt: now,
	}
}

// Acknowledge marks the incident as acknowledged by a user.
func (i *Incident) Acknowledge(userID uuid.UUID) error {
	if i.Status == IncidentStatusResolved {
		return ErrIncidentAlreadyResolved
	}
	if i.Status == IncidentStatusAcknowledged {
		return ErrIncidentAlreadyAcknowledged
	}

	now := time.Now()
	i.AcknowledgedBy = &userID
	i.AcknowledgedAt = &now
	i.Status = IncidentStatusAcknowledged
	return nil
}

// Resolve marks the incident as resolved and calculates TTR.
func (i *Incident) Resolve() error {
	if i.Status == IncidentStatusResolved {
		return ErrIncidentAlreadyResolved
	}

	now := time.Now()
	i.ResolvedAt = &now

	ttr := int(now.Sub(i.StartedAt).Seconds())
	i.TTRSeconds = &ttr

	i.Status = IncidentStatusResolved
	return nil
}

// Duration returns the duration of the incident.
// For resolved incidents, returns the time between start and resolution.
// For active incidents, returns the time since start.
func (i *Incident) Duration() time.Duration {
	if i.ResolvedAt != nil {
		return i.ResolvedAt.Sub(i.StartedAt)
	}
	return time.Since(i.StartedAt)
}

// IsOpen returns true if the incident is open.
func (i *Incident) IsOpen() bool {
	return i.Status == IncidentStatusOpen
}

// IsAcknowledged returns true if the incident is acknowledged.
func (i *Incident) IsAcknowledged() bool {
	return i.Status == IncidentStatusAcknowledged
}

// IsResolved returns true if the incident is resolved.
func (i *Incident) IsResolved() bool {
	return i.Status == IncidentStatusResolved
}

// IsActive returns true if the incident is still active (open or acknowledged).
func (i *Incident) IsActive() bool {
	return i.Status.IsActive()
}
