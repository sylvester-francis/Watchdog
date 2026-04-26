package metrics

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// Metrics holds OpenTelemetry meter handles for the 5 key metrics.
//
// Backwards compatibility: metric names, label names, and histogram bucket
// boundaries match the previous Prometheus-native definitions exactly,
// so dashboards and Prometheus queries continue to work unchanged. The
// OTel SDK's Prometheus exporter (registered inside telemetry.NewMeterProvider)
// bridges these meters back to the existing /metrics endpoint while an
// optional OTLP push reader sends them to a configured OTel collector.
type Metrics struct {
	httpDuration     metric.Float64Histogram
	heartbeatLatency metric.Float64Histogram

	// Incident counts for the watchdog_incidents_active gauge. Updated
	// synchronously via SetIncidents and read by the ObservableGauge
	// callback registered in New.
	openIncidents         atomic.Int64
	acknowledgedIncidents atomic.Int64

	history *MetricsHistory
}

// New constructs a Metrics value backed by the supplied OTel Meter and
// registers callbacks for the observable instruments. hub provides the
// WebSocket connection count; pool provides DB pool stats.
func New(meter metric.Meter, hub *realtime.Hub, pool *pgxpool.Pool) (*Metrics, error) {
	httpDuration, err := meter.Float64Histogram(
		"watchdog_http_request_duration_seconds",
		metric.WithDescription("HTTP request latency in seconds."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create http duration histogram: %w", err)
	}

	heartbeatLatency, err := meter.Float64Histogram(
		"watchdog_heartbeat_processing_seconds",
		metric.WithDescription("Time to process a heartbeat (validate + store + incident check)."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1),
	)
	if err != nil {
		return nil, fmt.Errorf("create heartbeat latency histogram: %w", err)
	}

	m := &Metrics{
		httpDuration:     httpDuration,
		heartbeatLatency: heartbeatLatency,
		history:          NewMetricsHistory(),
	}

	wsConnections, err := meter.Int64ObservableGauge(
		"watchdog_ws_connections_active",
		metric.WithDescription("Number of active WebSocket agent connections."),
	)
	if err != nil {
		return nil, fmt.Errorf("create ws connections gauge: %w", err)
	}

	dbPoolActive, err := meter.Int64ObservableGauge(
		"watchdog_db_pool_active_connections",
		metric.WithDescription("Number of acquired (in-use) database connections."),
	)
	if err != nil {
		return nil, fmt.Errorf("create db pool gauge: %w", err)
	}

	incidentsActive, err := meter.Int64ObservableGauge(
		"watchdog_incidents_active",
		metric.WithDescription("Number of active (open/acknowledged) incidents."),
	)
	if err != nil {
		return nil, fmt.Errorf("create incidents active gauge: %w", err)
	}

	openAttr := metric.WithAttributes(attribute.String("status", "open"))
	ackAttr := metric.WithAttributes(attribute.String("status", "acknowledged"))

	if _, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(wsConnections, int64(hub.ClientCount()))
		o.ObserveInt64(dbPoolActive, int64(pool.Stat().AcquiredConns()))
		o.ObserveInt64(incidentsActive, m.openIncidents.Load(), openAttr)
		o.ObserveInt64(incidentsActive, m.acknowledgedIncidents.Load(), ackAttr)
		return nil
	}, wsConnections, dbPoolActive, incidentsActive); err != nil {
		return nil, fmt.Errorf("register meter callback: %w", err)
	}

	return m, nil
}

// History returns the metrics history ring buffer.
func (m *Metrics) History() *MetricsHistory {
	return m.history
}

// HTTPMiddleware returns an Echo middleware that records request latency.
func (m *Metrics) HTTPMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip metrics endpoint itself to avoid recursion.
			if c.Path() == "/metrics" {
				return next(c)
			}

			start := time.Now()
			err := next(c)
			duration := time.Since(start).Seconds()

			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				}
			}

			m.httpDuration.Record(c.Request().Context(), duration, metric.WithAttributes(
				attribute.String("method", c.Request().Method),
				attribute.String("path", c.Path()),
				attribute.String("status", strconv.Itoa(status)),
			))

			return err
		}
	}
}

// ObserveHeartbeat records a heartbeat processing duration.
func (m *Metrics) ObserveHeartbeat(d time.Duration) {
	m.heartbeatLatency.Record(context.Background(), d.Seconds())
}

// SetIncidents sets the active incident counts by status. The values are
// surfaced through the watchdog_incidents_active gauge by the periodic
// observer callback registered in New.
func (m *Metrics) SetIncidents(open, acknowledged int) {
	m.openIncidents.Store(int64(open))
	m.acknowledgedIncidents.Store(int64(acknowledged))
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
// The OTel meter values are surfaced here via the Prometheus exporter
// registered with the default Prom registerer in telemetry.NewMeterProvider.
func Handler() echo.HandlerFunc {
	h := promhttp.Handler()
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
