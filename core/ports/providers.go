package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// TenantResolver resolves tenant context for multi-org self-hosted deployments.
type TenantResolver interface {
	Resolve(ctx context.Context) string
	TenantID(ctx context.Context, userID uuid.UUID) (string, error)
}

// requestMetadataKey is the context key for storing request metadata.
type requestMetadataKey struct{}

// RequestMetadata holds HTTP request details for tenant resolution.
// Injected into context by TenantScope middleware so resolvers can
// read headers and host without depending on Echo or net/http.
type RequestMetadata struct {
	Host     string
	Headers  map[string]string
	RemoteIP string
}

// WithRequestMetadata returns a context with the given request metadata.
func WithRequestMetadata(ctx context.Context, md *RequestMetadata) context.Context {
	return context.WithValue(ctx, requestMetadataKey{}, md)
}

// RequestMetadataFromContext extracts request metadata from context.
func RequestMetadataFromContext(ctx context.Context) *RequestMetadata {
	if md, ok := ctx.Value(requestMetadataKey{}).(*RequestMetadata); ok {
		return md
	}
	return nil
}

// AuthProvider provides pluggable authentication backends.
type AuthProvider interface {
	UserAuthService
	AgentAuthService
}

// AgentManager abstracts agent lifecycle management.
type AgentManager interface {
	CreateAgent(ctx context.Context, userID string, name string) (*domain.Agent, string, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) error
	ListAgents(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Agent, error)
}

// AlertRouter decouples alert routing from channel dispatch.
type AlertRouter interface {
	RouteIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
	RouteIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error
}

// StorageBackend abstracts database access with tenant-scoped query isolation.
type StorageBackend interface {
	WithTenantScope(ctx context.Context, tenantID string) context.Context
	Health(ctx context.Context) error
}

// Metric represents a single metric data point for external export.
type Metric struct {
	Name      string
	Value     float64
	Tags      map[string]string
	Timestamp time.Time
}

// MetricsExporter allows exporting metrics to external systems.
type MetricsExporter interface {
	Export(ctx context.Context, metrics []Metric) error
}

// DashboardRenderer provides pluggable dashboard rendering for custom themes.
type DashboardRenderer interface {
	RenderPage(ctx context.Context, name string, data any) ([]byte, error)
	RegisterTemplates(dir string) error
}

// ReportConfig defines parameters for generating operational reports.
type ReportConfig struct {
	Type   string
	From   time.Time
	To     time.Time
	Format string
}

// ReportGenerator generates operational reports.
type ReportGenerator interface {
	Generate(ctx context.Context, config ReportConfig) ([]byte, error)
}

// AuditLogger logs operational actions.
type AuditLogger interface {
	AuditService
}

// ClusterCoordinator supports clustered deployment.
type ClusterCoordinator interface {
	IsLeader() bool
	NodeID() string
}

// StatusPageProvider manages public status pages.
type StatusPageProvider interface {
	StatusPageRepository
}
