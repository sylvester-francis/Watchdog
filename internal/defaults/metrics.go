package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleMetricsExporter = "metrics_exporter"

var (
	_ registry.Module      = (*metricsModule)(nil)
	_ ports.MetricsExporter = (*metricsModule)(nil)
)

// metricsModule is a no-op metrics exporter.
type metricsModule struct{}

func newMetricsModule() *metricsModule {
	return &metricsModule{}
}

func (m *metricsModule) Name() string                    { return moduleMetricsExporter }
func (m *metricsModule) Init(_ context.Context) error    { return nil }
func (m *metricsModule) Health(_ context.Context) error   { return nil }
func (m *metricsModule) Shutdown(_ context.Context) error { return nil }

func (m *metricsModule) Export(_ context.Context, _ []ports.Metric) error {
	return nil
}
