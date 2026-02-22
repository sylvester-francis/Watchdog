package domain

import (
	"time"

	"github.com/google/uuid"
)

// HeartbeatStatus represents the result of a monitoring check.
type HeartbeatStatus string

const (
	HeartbeatStatusUp      HeartbeatStatus = "up"
	HeartbeatStatusDown    HeartbeatStatus = "down"
	HeartbeatStatusTimeout HeartbeatStatus = "timeout"
	HeartbeatStatusError   HeartbeatStatus = "error"
)

// IsValid checks if the status is a valid HeartbeatStatus.
func (s HeartbeatStatus) IsValid() bool {
	switch s {
	case HeartbeatStatusUp, HeartbeatStatusDown, HeartbeatStatusTimeout, HeartbeatStatusError:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if the heartbeat indicates success.
func (s HeartbeatStatus) IsSuccess() bool {
	return s == HeartbeatStatusUp
}

// IsFailure returns true if the heartbeat indicates failure.
func (s HeartbeatStatus) IsFailure() bool {
	return s == HeartbeatStatusDown || s == HeartbeatStatusTimeout || s == HeartbeatStatusError
}

// LatencyPoint represents an aggregated latency data point for charts.
type LatencyPoint struct {
	Time  time.Time
	AvgMs int
	MinMs int
	MaxMs int
}

// Heartbeat represents a single monitoring check result.
type Heartbeat struct {
	Time           time.Time
	MonitorID      uuid.UUID
	AgentID        uuid.UUID
	Status         HeartbeatStatus
	LatencyMs      *int
	ErrorMessage   *string
	CertExpiryDays *int
	CertIssuer     *string
}

// NewHeartbeat creates a new heartbeat with the current time.
func NewHeartbeat(monitorID, agentID uuid.UUID, status HeartbeatStatus) *Heartbeat {
	return &Heartbeat{
		Time:      time.Now(),
		MonitorID: monitorID,
		AgentID:   agentID,
		Status:    status,
	}
}

// NewSuccessHeartbeat creates a successful heartbeat with latency.
func NewSuccessHeartbeat(monitorID, agentID uuid.UUID, latencyMs int) *Heartbeat {
	h := NewHeartbeat(monitorID, agentID, HeartbeatStatusUp)
	h.LatencyMs = &latencyMs
	return h
}

// NewFailureHeartbeat creates a failed heartbeat with error message.
func NewFailureHeartbeat(monitorID, agentID uuid.UUID, status HeartbeatStatus, errorMsg string) *Heartbeat {
	h := NewHeartbeat(monitorID, agentID, status)
	h.ErrorMessage = &errorMsg
	return h
}

// SetLatency sets the latency in milliseconds.
func (h *Heartbeat) SetLatency(ms int) {
	h.LatencyMs = &ms
}

// SetError sets the error message.
func (h *Heartbeat) SetError(msg string) {
	h.ErrorMessage = &msg
}

// HasLatency returns true if latency was recorded.
func (h *Heartbeat) HasLatency() bool {
	return h.LatencyMs != nil
}

// HasError returns true if an error message was recorded.
func (h *Heartbeat) HasError() bool {
	return h.ErrorMessage != nil && *h.ErrorMessage != ""
}

// IsSuccess returns true if this heartbeat indicates success.
func (h *Heartbeat) IsSuccess() bool {
	return h.Status.IsSuccess()
}

// IsFailure returns true if this heartbeat indicates failure.
func (h *Heartbeat) IsFailure() bool {
	return h.Status.IsFailure()
}
