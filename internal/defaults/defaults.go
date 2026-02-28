package defaults

import (
	"log/slog"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

// Deps holds existing services and repositories needed to construct default modules.
type Deps struct {
	AuthService    ports.AuthProvider
	AgentAuth      ports.AgentAuthService
	AgentRepo      ports.AgentRepository
	Notifier       ports.Notifier
	AuditService   ports.AuditService
	StatusPageRepo ports.StatusPageRepository
	DB             ports.Transactor
	Logger         *slog.Logger
}

// RegisterAll registers all default module implementations into the registry.
func RegisterAll(reg *registry.Registry, deps Deps) {
	reg.Register(newTenantModule())
	reg.Register(newStorageModule(deps.DB))
	reg.Register(newAuthModule(deps.AuthService))
	reg.Register(newAgentModule(deps.AgentAuth, deps.AgentRepo))
	reg.Register(newAlertModule(deps.Notifier))
	reg.Register(newMetricsModule())
	reg.Register(newAuditModule(deps.AuditService))
	reg.Register(newClusterModule())
	reg.Register(newDashboardModule())
	reg.Register(newReportModule())
	reg.Register(newStatusModule(deps.StatusPageRepo))
}
