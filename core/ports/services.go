package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
)

// UserAuthService defines the interface for user authentication operations.
type UserAuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.User, error)
}

// AgentAuthService defines the interface for agent authentication and management.
type AgentAuthService interface {
	ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error)
	CreateAgent(ctx context.Context, userID string, name string) (*domain.Agent, string, error)
}

// MonitorService defines the interface for monitor orchestration.
type MonitorService interface {
	CreateMonitor(ctx context.Context, userID uuid.UUID, agentID uuid.UUID, name string, monitorType domain.MonitorType, target string, metadata map[string]string) (*domain.Monitor, error)
	GetMonitor(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)
	GetMonitorsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	UpdateMonitor(ctx context.Context, monitor *domain.Monitor) error
	DeleteMonitor(ctx context.Context, id uuid.UUID) error
	ProcessHeartbeat(ctx context.Context, heartbeat *domain.Heartbeat) error
	MarkAgentMonitorsDown(ctx context.Context, agentID uuid.UUID) error
}

// IncidentService defines the interface for incident lifecycle management.
type IncidentService interface {
	GetIncident(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetAllIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetIncidentsByMonitor(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)
	AcknowledgeIncident(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ResolveIncident(ctx context.Context, id uuid.UUID) error
	CreateIncidentIfNeeded(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error)
}

// AuditService defines the interface for security audit logging.
type AuditService interface {
	LogEvent(ctx context.Context, userID *uuid.UUID, action domain.AuditAction, ipAddress string, metadata map[string]string)
}

// AgentMessenger sends protocol messages to connected agents via WebSocket.
type AgentMessenger interface {
	SendToAgent(agentID uuid.UUID, msg *protocol.Message) bool
}

// Notifier defines the interface for sending alerts.
type Notifier interface {
	NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
}

// NotifierFactory creates a Notifier from an AlertChannel configuration.
type NotifierFactory interface {
	BuildFromChannel(channel *domain.AlertChannel) (Notifier, error)
}
