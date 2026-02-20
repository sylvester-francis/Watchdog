package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleAlertRouter = "alert_router"

var (
	_ registry.Module  = (*alertModule)(nil)
	_ ports.AlertRouter = (*alertModule)(nil)
)

// alertModule wraps the existing Notifier for alert routing.
type alertModule struct {
	notifier ports.Notifier
}

func newAlertModule(notifier ports.Notifier) *alertModule {
	return &alertModule{notifier: notifier}
}

func (m *alertModule) Name() string                    { return moduleAlertRouter }
func (m *alertModule) Init(_ context.Context) error    { return nil }
func (m *alertModule) Health(_ context.Context) error   { return nil }
func (m *alertModule) Shutdown(_ context.Context) error { return nil }

func (m *alertModule) RouteIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	return m.notifier.NotifyIncidentOpened(ctx, incident, monitor)
}

func (m *alertModule) RouteIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	return m.notifier.NotifyIncidentResolved(ctx, incident, monitor)
}
