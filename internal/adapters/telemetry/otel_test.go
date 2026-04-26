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
	cfg := config.TelemetryConfig{Enabled: false}

	tp, shutdown, err := telemetry.NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, shutdown)

	// Disabled must NOT instantiate an SDK provider.
	_, isSDK := tp.(*sdktrace.TracerProvider)
	assert.False(t, isSDK, "disabled config must return no-op provider, not SDK provider")

	// Shutdown returns nil and is safe to call.
	require.NoError(t, shutdown(context.Background()))

	// Tracer must be usable without panic; satisfies the interface contract.
	var _ trace.TracerProvider = tp
	_, span := tp.Tracer("test").Start(context.Background(), "test-span")
	span.End()
}

func TestNewTracerProvider_EnabledReturnsSDKProvider(t *testing.T) {
	cfg := config.TelemetryConfig{Enabled: true, ServiceName: "test-svc"}

	tp, shutdown, err := telemetry.NewTracerProvider(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, tp)
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	_, isSDK := tp.(*sdktrace.TracerProvider)
	require.True(t, isSDK, "enabled config must produce an SDK provider")
}
