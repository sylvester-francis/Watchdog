package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester/watchdog/internal/core/domain"
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	Register(ctx context.Context, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.User, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error)
}

// MonitorService defines the interface for monitor orchestration.
type MonitorService interface {
	CreateMonitor(ctx context.Context, agentID uuid.UUID, name string, monitorType domain.MonitorType, target string) (*domain.Monitor, error)
	GetMonitor(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)
	GetMonitorsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	UpdateMonitor(ctx context.Context, monitor *domain.Monitor) error
	DeleteMonitor(ctx context.Context, id uuid.UUID) error
	ProcessHeartbeat(ctx context.Context, heartbeat *domain.Heartbeat) error
}

// IncidentService defines the interface for incident lifecycle management.
type IncidentService interface {
	GetIncident(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetIncidentsByMonitor(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)
	AcknowledgeIncident(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ResolveIncident(ctx context.Context, id uuid.UUID) error
	CreateIncidentIfNeeded(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error)
}

// Notifier defines the interface for sending alerts.
type Notifier interface {
	NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
}
