package domain

import (
	"time"

	"github.com/google/uuid"
)

// Discovery scan statuses.
const (
	DiscoveryStatusPending  = "pending"
	DiscoveryStatusRunning  = "running"
	DiscoveryStatusComplete = "complete"
	DiscoveryStatusError    = "error"
)

// DiscoveryScan represents a network discovery operation.
type DiscoveryScan struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	AgentID      uuid.UUID
	Subnet       string
	Status       string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	HostCount    int
	ErrorMessage string
	CreatedAt    time.Time
}

// DiscoveredDevice represents a device found during a network scan.
type DiscoveredDevice struct {
	ID                  uuid.UUID
	ScanID              uuid.UUID
	UserID              uuid.UUID
	IP                  string
	Hostname            string
	SysDescr            string
	SysObjectID         string
	SysName             string
	SNMPReachable       bool
	PingReachable       bool
	SuggestedTemplateID string
	MonitorCreated      bool
	DiscoveredAt        time.Time
}
