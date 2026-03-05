package metrics

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// Metrics holds the Prometheus collectors for the 5 key metrics.
type Metrics struct {
	httpDuration     *prometheus.HistogramVec
	wsConnections    prometheus.GaugeFunc
	heartbeatLatency prometheus.Histogram
	incidentsActive  *prometheus.GaugeVec
	dbPoolActive     prometheus.GaugeFunc
}

// New creates and registers the Prometheus metrics.
// hub provides WebSocket connection counts, pool provides DB pool stats.
func New(hub *realtime.Hub, pool *pgxpool.Pool) *Metrics {
	m := &Metrics{
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "watchdog_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),

		wsConnections: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "watchdog_ws_connections_active",
			Help: "Number of active WebSocket agent connections.",
		}, func() float64 {
			return float64(hub.ClientCount())
		}),

		heartbeatLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "watchdog_heartbeat_processing_seconds",
			Help:    "Time to process a heartbeat (validate + store + incident check).",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),

		incidentsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "watchdog_incidents_active",
			Help: "Number of active (open/acknowledged) incidents.",
		}, []string{"status"}),

		dbPoolActive: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "watchdog_db_pool_active_connections",
			Help: "Number of acquired (in-use) database connections.",
		}, func() float64 {
			return float64(pool.Stat().AcquiredConns())
		}),
	}

	prometheus.MustRegister(
		m.httpDuration,
		m.wsConnections,
		m.heartbeatLatency,
		m.incidentsActive,
		m.dbPoolActive,
	)

	return m
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

			m.httpDuration.WithLabelValues(
				c.Request().Method,
				c.Path(),
				strconv.Itoa(status),
			).Observe(duration)

			return err
		}
	}
}

// ObserveHeartbeat records a heartbeat processing duration.
func (m *Metrics) ObserveHeartbeat(d time.Duration) {
	m.heartbeatLatency.Observe(d.Seconds())
}

// SetIncidents sets the active incident counts by status.
func (m *Metrics) SetIncidents(open, acknowledged int) {
	m.incidentsActive.WithLabelValues("open").Set(float64(open))
	m.incidentsActive.WithLabelValues("acknowledged").Set(float64(acknowledged))
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() echo.HandlerFunc {
	h := promhttp.Handler()
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
