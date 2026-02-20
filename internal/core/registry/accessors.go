package registry

import "github.com/sylvester-francis/watchdog/internal/core/ports"

// Module name constants used by default implementations and typed accessors.
const (
	ModuleTenantResolver     = "tenant_resolver"
	ModuleAuthProvider       = "auth_provider"
	ModuleAgentManager       = "agent_manager"
	ModuleAlertRouter        = "alert_router"
	ModuleStorageBackend     = "storage_backend"
	ModuleMetricsExporter    = "metrics_exporter"
	ModuleDashboardRenderer  = "dashboard_renderer"
	ModuleReportGenerator    = "report_generator"
	ModuleAuditLogger        = "audit_logger"
	ModuleClusterCoordinator = "cluster_coordinator"
	ModuleStatusPageProvider = "status_page_provider"
)

// TenantResolver returns the registered TenantResolver module.
func (r *Registry) TenantResolver() ports.TenantResolver {
	return r.MustGet(ModuleTenantResolver).(ports.TenantResolver)
}

// AuthProvider returns the registered AuthProvider module.
func (r *Registry) AuthProvider() ports.AuthProvider {
	return r.MustGet(ModuleAuthProvider).(ports.AuthProvider)
}

// AgentManager returns the registered AgentManager module.
func (r *Registry) AgentManager() ports.AgentManager {
	return r.MustGet(ModuleAgentManager).(ports.AgentManager)
}

// AlertRouter returns the registered AlertRouter module.
func (r *Registry) AlertRouter() ports.AlertRouter {
	return r.MustGet(ModuleAlertRouter).(ports.AlertRouter)
}

// StorageBackend returns the registered StorageBackend module.
func (r *Registry) StorageBackend() ports.StorageBackend {
	return r.MustGet(ModuleStorageBackend).(ports.StorageBackend)
}

// MetricsExporter returns the registered MetricsExporter module.
func (r *Registry) MetricsExporter() ports.MetricsExporter {
	return r.MustGet(ModuleMetricsExporter).(ports.MetricsExporter)
}

// DashboardRenderer returns the registered DashboardRenderer module.
func (r *Registry) DashboardRenderer() ports.DashboardRenderer {
	return r.MustGet(ModuleDashboardRenderer).(ports.DashboardRenderer)
}

// ReportGenerator returns the registered ReportGenerator module.
func (r *Registry) ReportGenerator() ports.ReportGenerator {
	return r.MustGet(ModuleReportGenerator).(ports.ReportGenerator)
}

// AuditLogger returns the registered AuditLogger module.
func (r *Registry) AuditLogger() ports.AuditLogger {
	return r.MustGet(ModuleAuditLogger).(ports.AuditLogger)
}

// ClusterCoordinator returns the registered ClusterCoordinator module.
func (r *Registry) ClusterCoordinator() ports.ClusterCoordinator {
	return r.MustGet(ModuleClusterCoordinator).(ports.ClusterCoordinator)
}

// StatusPageProvider returns the registered StatusPageProvider module.
func (r *Registry) StatusPageProvider() ports.StatusPageProvider {
	return r.MustGet(ModuleStatusPageProvider).(ports.StatusPageProvider)
}
