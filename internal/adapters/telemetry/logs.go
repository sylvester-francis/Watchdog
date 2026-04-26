package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/sylvester-francis/watchdog/internal/config"
)

// NewLoggerProvider returns an OpenTelemetry LoggerProvider plus a shutdown
// function. The gate mirrors NewTracerProvider:
//
//   - cfg.Enabled is true (default), AND
//   - an OTLP logs endpoint is configured via OTEL_EXPORTER_OTLP_ENDPOINT
//     or OTEL_EXPORTER_OTLP_LOGS_ENDPOINT
//
// otherwise a no-op LoggerProvider is returned with a no-op shutdown — no
// exporter is created, no network egress.
//
// Callers SHOULD defer the returned shutdown during graceful termination
// so any batched log records flush before exit.
func NewLoggerProvider(ctx context.Context, cfg config.TelemetryConfig) (log.LoggerProvider, func(context.Context) error, error) {
	if !cfg.Enabled || !hasOTLPLogsEndpoint() {
		return lognoop.NewLoggerProvider(), noopShutdown, nil
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

	exporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("build OTLP HTTP log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	return lp, lp.Shutdown, nil
}

// hasOTLPLogsEndpoint reports whether the OTel SDK will find a configured
// OTLP logs endpoint. Either the generic OTEL_EXPORTER_OTLP_ENDPOINT or
// the signal-specific OTEL_EXPORTER_OTLP_LOGS_ENDPOINT counts.
func hasOTLPLogsEndpoint() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") != ""
}

// NewSlogHandler returns a slog.Handler that emits each log record to BOTH
// the supplied base handler and the OTel logs bridge backed by lp.
//
// When lp is the no-op LoggerProvider (telemetry disabled or no endpoint
// configured), the bridge is omitted and base is returned unwrapped — the
// disabled path is zero-cost: no extra dispatch, no record cloning, no
// otelslog handler in the chain at all.
func NewSlogHandler(base slog.Handler, lp log.LoggerProvider, serviceName string) slog.Handler {
	if _, isNoop := lp.(lognoop.LoggerProvider); isNoop {
		return base
	}
	bridge := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp))
	return &teeHandler{primary: base, secondary: bridge}
}

// teeHandler emits each log record to both primary and secondary. Errors
// from primary take precedence — the OTel side is best-effort and must
// not mask a stdout/stderr failure.
type teeHandler struct {
	primary, secondary slog.Handler
}

func (t *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return t.primary.Enabled(ctx, level) || t.secondary.Enabled(ctx, level)
}

func (t *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	primaryErr := t.primary.Handle(ctx, r.Clone())
	secondaryErr := t.secondary.Handle(ctx, r)
	if primaryErr != nil {
		return primaryErr
	}
	return secondaryErr
}

func (t *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{primary: t.primary.WithAttrs(attrs), secondary: t.secondary.WithAttrs(attrs)}
}

func (t *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{primary: t.primary.WithGroup(name), secondary: t.secondary.WithGroup(name)}
}
