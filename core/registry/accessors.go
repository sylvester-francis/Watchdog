package registry

import "github.com/sylvester-francis/watchdog/core/ports"

// Module name constants used by default implementations and typed accessors.
const (
	ModuleTenantResolver     = "tenant_resolver"
	ModuleAuthProvider       = "auth_provider"
	ModuleAgentManager       = "agent_manager"
	ModuleAlertRouter        = "alert_router"
	ModuleStorageBackend     = "storage_backend"
	ModuleAuditLogger    = "audit_logger"
	ModuleWorkflowEngine = "workflow_engine"
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

// AuditLogger returns the registered AuditLogger module.
func (r *Registry) AuditLogger() ports.AuditLogger {
	return r.MustGet(ModuleAuditLogger).(ports.AuditLogger)
}

// WorkflowEngine returns the registered WorkflowEngine module, or nil if not registered.
func (r *Registry) WorkflowEngine() ports.WorkflowEngine {
	m, ok := r.Get(ModuleWorkflowEngine)
	if !ok {
		return nil
	}
	return m.(ports.WorkflowEngine)
}
