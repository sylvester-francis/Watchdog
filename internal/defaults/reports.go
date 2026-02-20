package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const moduleReportGenerator = "report_generator"

var (
	_ registry.Module       = (*reportModule)(nil)
	_ ports.ReportGenerator = (*reportModule)(nil)
)

// reportModule is a no-op report generator.
type reportModule struct{}

func newReportModule() *reportModule {
	return &reportModule{}
}

func (m *reportModule) Name() string                    { return moduleReportGenerator }
func (m *reportModule) Init(_ context.Context) error    { return nil }
func (m *reportModule) Health(_ context.Context) error   { return nil }
func (m *reportModule) Shutdown(_ context.Context) error { return nil }

func (m *reportModule) Generate(_ context.Context, _ ports.ReportConfig) ([]byte, error) {
	return nil, nil
}
