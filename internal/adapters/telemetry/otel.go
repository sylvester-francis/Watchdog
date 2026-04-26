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
//   - cfg.Enabled == false: a no-op provider is returned; shutdown is a
//     no-op returning nil; no exporter is created.
//   - cfg.Enabled == true: the OTel SDK is initialized with an OTLP HTTP
//     exporter. Endpoint, headers, sampler, and other transport details
//     come from standard OTEL_* env vars read by the SDK directly.
//
// Callers SHOULD defer the returned shutdown during graceful termination
// so any batched spans flush before exit.
func NewTracerProvider(ctx context.Context, cfg config.TelemetryConfig) (trace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Enabled {
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

func noopShutdown(context.Context) error { return nil }
