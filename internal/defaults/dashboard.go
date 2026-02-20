package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const moduleDashboardRenderer = "dashboard_renderer"

var (
	_ registry.Module        = (*dashboardModule)(nil)
	_ ports.DashboardRenderer = (*dashboardModule)(nil)
)

// dashboardModule wraps the existing Templates for standard template rendering.
// Handlers use Templates directly via echo.Renderer by default.
type dashboardModule struct {
	templates *view.Templates
}

func newDashboardModule(templates *view.Templates) *dashboardModule {
	return &dashboardModule{templates: templates}
}

func (m *dashboardModule) Name() string                    { return moduleDashboardRenderer }
func (m *dashboardModule) Init(_ context.Context) error    { return nil }
func (m *dashboardModule) Health(_ context.Context) error   { return nil }
func (m *dashboardModule) Shutdown(_ context.Context) error { return nil }

func (m *dashboardModule) RenderPage(_ context.Context, _ string, _ any) ([]byte, error) {
	return nil, nil
}

func (m *dashboardModule) RegisterTemplates(_ string) error {
	return nil
}
