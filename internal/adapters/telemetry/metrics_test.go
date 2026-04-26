package telemetry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/telemetry"
	"github.com/sylvester-francis/watchdog/internal/config"
)

func TestNewMeterProvider_AlwaysReturnsMeterProvider(t *testing.T) {
	// Even with OTel telemetry disabled and no endpoint, the Prom reader
	// must still be wired (auto-registered with the default Prom
	// registerer inside otelprom.New) so the existing /metrics endpoint
	// keeps working.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")
	cfg := config.TelemetryConfig{Enabled: false}

	mp, shutdown, err := telemetry.NewMeterProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, mp, "MeterProvider must always be non-nil")
	require.NotNil(t, shutdown)

	require.NoError(t, shutdown(context.Background()))
}

func TestNewMeterProvider_EnabledWithEndpointBuildsOK(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp.example.com")
	cfg := config.TelemetryConfig{Enabled: true, ServiceName: "test-svc"}

	mp, shutdown, err := telemetry.NewMeterProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, mp)
	t.Cleanup(func() { _ = shutdown(context.Background()) })
}

func TestNewMeterProvider_DisabledOmitsOTLPReader(t *testing.T) {
	// Force-disable beats endpoint presence — caller relies on this to
	// suppress OTLP egress while still serving /metrics locally.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://does-not-matter.example.com")
	cfg := config.TelemetryConfig{Enabled: false}

	mp, shutdown, err := telemetry.NewMeterProvider(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, mp)
	require.NoError(t, shutdown(context.Background()))
}
