package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/sylvester-francis/watchdog/internal/config"
)

// NewMeterProvider builds a MeterProvider that always feeds the existing
// Prometheus /metrics endpoint and additionally pushes to an OTLP backend
// when one is configured.
//
// The Prom exporter auto-registers with the default Prometheus registerer
// inside otelprom.New(), so callers don't need a separate hookup — once
// the MeterProvider records into a meter, the value flows out of both
// /metrics (pull) and the OTLP push channel (when configured).
//
// Gate semantics mirror NewTracerProvider / NewLoggerProvider:
//
//   - The OTLP reader is added iff cfg.Enabled is true AND an OTLP metrics
//     endpoint is configured (OTEL_EXPORTER_OTLP_ENDPOINT or
//     OTEL_EXPORTER_OTLP_METRICS_ENDPOINT).
//   - The Prom reader is unconditional — /metrics is core observability
//     that doesn't depend on the OTel toggle.
func NewMeterProvider(ctx context.Context, cfg config.TelemetryConfig) (*sdkmetric.MeterProvider, func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	promExp, err := otelprom.New()
	if err != nil {
		return nil, nil, fmt.Errorf("build prometheus exporter: %w", err)
	}

	options := []sdkmetric.Option{
		sdkmetric.WithReader(promExp),
		sdkmetric.WithResource(res),
	}

	if cfg.Enabled && hasOTLPMetricsEndpoint() {
		otlpExp, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("build OTLP HTTP metric exporter: %w", err)
		}
		options = append(options, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlpExp)))
	}

	mp := sdkmetric.NewMeterProvider(options...)
	return mp, mp.Shutdown, nil
}

// hasOTLPMetricsEndpoint reports whether the OTel SDK will find a configured
// OTLP metrics endpoint. Either the generic OTEL_EXPORTER_OTLP_ENDPOINT or
// the signal-specific OTEL_EXPORTER_OTLP_METRICS_ENDPOINT counts.
func hasOTLPMetricsEndpoint() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != ""
}
