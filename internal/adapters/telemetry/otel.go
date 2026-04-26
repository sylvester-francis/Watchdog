// Package telemetry wires the OpenTelemetry SDK into the engine.
//
// Scope is intentionally narrow: a single constructor that returns either
// a no-op TracerProvider (when disabled) or an SDK-backed provider with
// an OTLP HTTP exporter (when enabled). All endpoint, header, and sampler
// configuration flows through the SDK's standard OTEL_* environment
// variables — see https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/.
package telemetry

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/sylvester-francis/watchdog/internal/config"
)

// NewTracerProvider returns a TracerProvider and a shutdown function.
//
// The SDK is initialized only when both:
//   - cfg.Enabled is true (default; set WATCHDOG_OTEL_ENABLED=false to
//     force-disable when an endpoint happens to be configured)
//   - an OTLP traces endpoint is configured via OTEL_EXPORTER_OTLP_ENDPOINT
//     or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
//
// Otherwise a no-op provider is returned with a no-op shutdown — no
// exporter is created, no network egress, no log spam from failed
// export retries against the SDK's localhost:4318 fallback.
//
// Endpoint, headers, sampler, and other transport details come from
// standard OTEL_* env vars read by the SDK directly.
//
// Callers SHOULD defer the returned shutdown during graceful termination
// so any batched spans flush before exit.
func NewTracerProvider(ctx context.Context, cfg config.TelemetryConfig) (trace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Enabled || !hasOTLPTracesEndpoint() {
		return tracenoop.NewTracerProvider(), noopShutdown, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("build OTLP HTTP trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	return tp, tp.Shutdown, nil
}

// hasOTLPTracesEndpoint reports whether the OTel SDK will find a
// configured OTLP traces endpoint. Either the generic OTEL_EXPORTER_OTLP_ENDPOINT
// or the signal-specific OTEL_EXPORTER_OTLP_TRACES_ENDPOINT counts.
func hasOTLPTracesEndpoint() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}

func noopShutdown(context.Context) error { return nil }
