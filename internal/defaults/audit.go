package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleAuditLogger = "audit_logger"

var (
	_ registry.Module  = (*auditModule)(nil)
	_ ports.AuditLogger = (*auditModule)(nil)
)

// auditModule wraps the existing AuditService.
type auditModule struct {
	ports.AuditService
}

func newAuditModule(svc ports.AuditService) *auditModule {
	return &auditModule{AuditService: svc}
}

func (m *auditModule) Name() string                    { return moduleAuditLogger }
func (m *auditModule) Init(_ context.Context) error    { return nil }
func (m *auditModule) Health(_ context.Context) error   { return nil }
func (m *auditModule) Shutdown(_ context.Context) error { return nil }
