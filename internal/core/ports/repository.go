package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

// UserUsageSummary holds user info with current resource counts for admin views.
type UserUsageSummary struct {
	Email        string
	Plan         domain.Plan
	AgentCount   int
	AgentMax     int
	MonitorCount int
	MonitorMax   int
}

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Count(ctx context.Context) (int, error)
	CountByPlan(ctx context.Context) (map[domain.Plan]int, error)
	GetUsersNearLimits(ctx context.Context) ([]UserUsageSummary, error)
}

// AgentRepository defines the interface for agent persistence.
type AgentRepository interface {
	Create(ctx context.Context, agent *domain.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error)
	Update(ctx context.Context, agent *domain.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error
	UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// MonitorRepository defines the interface for monitor persistence.
type MonitorRepository interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error)
	GetByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	GetEnabledByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error)
	Update(ctx context.Context, monitor *domain.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// IncidentRepository defines the interface for incident persistence.
type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error)
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error)
	GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error)
	GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error)
	GetAllIncidents(ctx context.Context) ([]*domain.Incident, error)
	Update(ctx context.Context, incident *domain.Incident) error
	Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	Resolve(ctx context.Context, id uuid.UUID) error
}

// HeartbeatRepository defines the interface for heartbeat persistence.
type HeartbeatRepository interface {
	Create(ctx context.Context, heartbeat *domain.Heartbeat) error
	CreateBatch(ctx context.Context, heartbeats []*domain.Heartbeat) error
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.Heartbeat, error)
	GetByMonitorIDInRange(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Heartbeat, error)
	GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Heartbeat, error)
	GetRecentFailures(ctx context.Context, monitorID uuid.UUID, count int) ([]*domain.Heartbeat, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// UsageEventRepository defines the interface for usage event persistence.
type UsageEventRepository interface {
	Create(ctx context.Context, event *domain.UsageEvent) error
	GetRecentByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UsageEvent, error)
	GetRecent(ctx context.Context, limit int) ([]*domain.UsageEvent, error)
	CountByEventType(ctx context.Context, eventType domain.EventType, since time.Time) (int, error)
}

// WaitlistRepository defines the interface for waitlist signup persistence.
type WaitlistRepository interface {
	Create(ctx context.Context, signup *domain.WaitlistSignup) error
	GetByEmail(ctx context.Context, email string) (*domain.WaitlistSignup, error)
	Count(ctx context.Context) (int, error)
}

// Transactor defines the interface for database transactions.
type Transactor interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
