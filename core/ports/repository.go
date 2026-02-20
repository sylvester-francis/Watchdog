package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
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

// AdminUserView holds full user info with resource counts for admin user management.
type AdminUserView struct {
	ID           uuid.UUID
	Email        string
	Plan         domain.Plan
	IsAdmin      bool
	AgentCount   int
	MonitorCount int
	AgentMax     int
	MonitorMax   int
	CreatedAt    time.Time
}

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByEmailGlobal(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	Count(ctx context.Context) (int, error)
	CountByPlan(ctx context.Context) (map[domain.Plan]int, error)
	GetUsersNearLimits(ctx context.Context) ([]UserUsageSummary, error)
	GetAllWithUsage(ctx context.Context) ([]AdminUserView, error)
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
	UpdateFingerprint(ctx context.Context, id uuid.UUID, fingerprint map[string]string) error
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

// APITokenRepository defines the interface for API token persistence.
type APITokenRepository interface {
	Create(ctx context.Context, token *domain.APIToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID, ip string) error
}

// AuditLogRepository defines the interface for audit log persistence.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.AuditLog, error)
	GetRecent(ctx context.Context, limit int) ([]*domain.AuditLog, error)
}

// StatusPageRepository defines the interface for status page persistence.
type StatusPageRepository interface {
	Create(ctx context.Context, page *domain.StatusPage) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.StatusPage, error)
	GetByUserAndSlug(ctx context.Context, username, slug string) (*domain.StatusPage, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.StatusPage, error)
	Update(ctx context.Context, page *domain.StatusPage) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetMonitors(ctx context.Context, pageID uuid.UUID, monitorIDs []uuid.UUID) error
	GetMonitorIDs(ctx context.Context, pageID uuid.UUID) ([]uuid.UUID, error)
	SlugExistsForUser(ctx context.Context, userID uuid.UUID, slug string) (bool, error)
}

// AlertChannelRepository defines the interface for alert channel persistence.
type AlertChannelRepository interface {
	Create(ctx context.Context, channel *domain.AlertChannel) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertChannel, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AlertChannel, error)
	GetEnabledByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AlertChannel, error)
	Update(ctx context.Context, channel *domain.AlertChannel) error
	Delete(ctx context.Context, id uuid.UUID) error
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
