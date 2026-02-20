package defaults

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const moduleTenantResolver = "tenant_resolver"

var (
	_ registry.Module    = (*tenantModule)(nil)
	_ ports.TenantResolver = (*tenantModule)(nil)
)

// tenantModule returns a static "default" tenant for all queries.
type tenantModule struct{}

func newTenantModule() *tenantModule {
	return &tenantModule{}
}

func (m *tenantModule) Name() string                        { return moduleTenantResolver }
func (m *tenantModule) Init(_ context.Context) error        { return nil }
func (m *tenantModule) Health(_ context.Context) error       { return nil }
func (m *tenantModule) Shutdown(_ context.Context) error     { return nil }
func (m *tenantModule) Resolve(_ context.Context) string     { return "default" }

func (m *tenantModule) TenantID(_ context.Context, _ uuid.UUID) (string, error) {
	return "default", nil
}
