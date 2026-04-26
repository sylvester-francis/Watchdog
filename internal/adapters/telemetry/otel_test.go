package telemetry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/sylvester-francis/watchdog/internal/adapters/telemetry"
	"github.com/sylvester-francis/watchdog/internal/config"
)

func TestNewTracerProvider_DisabledReturnsNoop(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://does-not-matter.example.com")
	cfg := config.TelemetryConfig{Enabled: false}

	tp, shutdown, err := telemetry.NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, shutdown)

	// Force-disable wins over a configured endpoint.
	_, isSDK := tp.(*sdktrace.TracerProvider)
	assert.False(t, isSDK, "Enabled=false must return no-op even with endpoint set")

	require.NoError(t, shutdown(context.Background()))

	// Tracer must be usable without panic; satisfies the interface contract.
	var _ trace.TracerProvider = tp
	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	span.End()
}

func TestNewTracerProvider_EnabledWithoutEndpointReturnsNoop(t *testing.T) {
	// Make sure no inherited endpoint env from the parent shell leaks into the test.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	cfg := config.TelemetryConfig{Enabled: true, ServiceName: "test-svc"}

	tp, shutdown, err := telemetry.NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	// Default-on must still produce a no-op when no OTLP endpoint is
	// configured — the alternative would be the SDK retrying against
	// localhost:4318 and spamming logs.
	_, isSDK := tp.(*sdktrace.TracerProvider)
	assert.False(t, isSDK, "Enabled=true with no endpoint must return no-op (no log spam)")
}

func TestNewTracerProvider_EnabledWithEndpointReturnsSDKProvider(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp.example.com")
	cfg := config.TelemetryConfig{Enabled: true, ServiceName: "test-svc"}

	tp, shutdown, err := telemetry.NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	_, isSDK := tp.(*sdktrace.TracerProvider)
	require.True(t, isSDK, "Enabled=true with endpoint must produce an SDK provider")
}
