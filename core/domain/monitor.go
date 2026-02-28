package domain

import (
	"time"

	"github.com/google/uuid"
)

// MonitorType represents the type of monitoring check.
type MonitorType string

const (
	MonitorTypePing     MonitorType = "ping"
	MonitorTypeHTTP     MonitorType = "http"
	MonitorTypeTCP      MonitorType = "tcp"
	MonitorTypeDNS      MonitorType = "dns"
	MonitorTypeTLS      MonitorType = "tls"
	MonitorTypeDocker   MonitorType = "docker"
	MonitorTypeDatabase MonitorType = "database"
	MonitorTypeSystem   MonitorType = "system"
	MonitorTypeService  MonitorType = "service"
)

// ValidMonitorTypes lists all valid monitor types.
var ValidMonitorTypes = []MonitorType{
	MonitorTypePing, MonitorTypeHTTP, MonitorTypeTCP, MonitorTypeDNS, MonitorTypeTLS,
	MonitorTypeDocker, MonitorTypeDatabase, MonitorTypeSystem, MonitorTypeService,
}

// ValidMonitorTypeStrings returns monitor types as strings (for templates).
func ValidMonitorTypeStrings() []string {
	out := make([]string, len(ValidMonitorTypes))
	for i, t := range ValidMonitorTypes {
		out[i] = string(t)
	}
	return out
}

// IsValid checks if the type is a valid MonitorType.
func (t MonitorType) IsValid() bool {
	switch t {
	case MonitorTypePing, MonitorTypeHTTP, MonitorTypeTCP, MonitorTypeDNS, MonitorTypeTLS,
		MonitorTypeDocker, MonitorTypeDatabase, MonitorTypeSystem, MonitorTypeService:
		return true
	default:
		return false
	}
}

// MonitorStatus represents the current status of a monitor.
type MonitorStatus string

const (
	MonitorStatusPending  MonitorStatus = "pending"
	MonitorStatusUp       MonitorStatus = "up"
	MonitorStatusDown     MonitorStatus = "down"
	MonitorStatusDegraded MonitorStatus = "degraded"
)

// IsValid checks if the status is a valid MonitorStatus.
func (s MonitorStatus) IsValid() bool {
	switch s {
	case MonitorStatusPending, MonitorStatusUp, MonitorStatusDown, MonitorStatusDegraded:
		return true
	default:
		return false
	}
}

// IsHealthy returns true if the status indicates the monitor is healthy.
func (s MonitorStatus) IsHealthy() bool {
	return s == MonitorStatusUp
}

// Monitor represents a monitoring target configuration.
type Monitor struct {
	ID               uuid.UUID
	AgentID          uuid.UUID
	Name             string
	Type             MonitorType
	Target           string
	IntervalSeconds  int
	TimeoutSeconds   int
	Status           MonitorStatus
	Enabled          bool
	FailureThreshold  int
	Metadata          map[string]string
	SLATargetPercent  *float64
	CreatedAt         time.Time
}

// Default values for monitor configuration.
const (
	DefaultIntervalSeconds  = 30
	DefaultTimeoutSeconds   = 10
	MinIntervalSeconds      = 5
	MaxIntervalSeconds      = 3600
	MinTimeoutSeconds       = 1
	MaxTimeoutSeconds       = 60
	DefaultFailureThreshold = 3
	MinFailureThreshold     = 1
	MaxFailureThreshold     = 20
)

// NewMonitor creates a new Monitor with default settings.
func NewMonitor(agentID uuid.UUID, name string, monitorType MonitorType, target string) *Monitor {
	return &Monitor{
		ID:               uuid.New(),
		AgentID:          agentID,
		Name:             name,
		Type:             monitorType,
		Target:           target,
		IntervalSeconds:  DefaultIntervalSeconds,
		TimeoutSeconds:   DefaultTimeoutSeconds,
		Status:           MonitorStatusPending,
		Enabled:          true,
		FailureThreshold: DefaultFailureThreshold,
		Metadata:         make(map[string]string),
		CreatedAt:        time.Now(),
	}
}

// SetInterval sets the check interval with validation.
func (m *Monitor) SetInterval(seconds int) bool {
	if seconds < MinIntervalSeconds || seconds > MaxIntervalSeconds {
		return false
	}
	m.IntervalSeconds = seconds
	return true
}

// SetTimeout sets the check timeout with validation.
func (m *Monitor) SetTimeout(seconds int) bool {
	if seconds < MinTimeoutSeconds || seconds > MaxTimeoutSeconds {
		return false
	}
	m.TimeoutSeconds = seconds
	return true
}

// UpdateStatus updates the monitor status.
func (m *Monitor) UpdateStatus(status MonitorStatus) {
	m.Status = status
}

// Enable enables the monitor.
func (m *Monitor) Enable() {
	m.Enabled = true
}

// Disable disables the monitor.
func (m *Monitor) Disable() {
	m.Enabled = false
}

// IsEnabled returns true if the monitor is enabled.
func (m *Monitor) IsEnabled() bool {
	return m.Enabled
}
