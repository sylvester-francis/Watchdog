package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const moduleDashboardRenderer = "dashboard_renderer"

var (
	_ registry.Module         = (*dashboardModule)(nil)
	_ ports.DashboardRenderer = (*dashboardModule)(nil)
)

// dashboardModule is a no-op placeholder for the dashboard renderer module.
// The SvelteKit SPA handles all page rendering; this exists only to satisfy
// the module registry interface.
type dashboardModule struct{}

func newDashboardModule() *dashboardModule {
	return &dashboardModule{}
}

func (m *dashboardModule) Name() string                    { return moduleDashboardRenderer }
func (m *dashboardModule) Init(_ context.Context) error    { return nil }
func (m *dashboardModule) Health(_ context.Context) error  { return nil }
func (m *dashboardModule) Shutdown(_ context.Context) error { return nil }

func (m *dashboardModule) RenderPage(_ context.Context, _ string, _ any) ([]byte, error) {
	return nil, nil
}

func (m *dashboardModule) RegisterTemplates(_ string) error {
	return nil
}
