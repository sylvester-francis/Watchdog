package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

const (
	defaultAnomalyListWindow  = 24 * time.Hour
	defaultAnomalyCountWindow = time.Hour
	maxAnomalyListWindow      = 7 * 24 * time.Hour
)

// AnomalyHandler exposes latency anomaly endpoints under /api/v1/monitors/:id.
type AnomalyHandler struct {
	svc *services.AnomalyService
}

// NewAnomalyHandler constructs the handler.
func NewAnomalyHandler(svc *services.AnomalyService) *AnomalyHandler {
	return &AnomalyHandler{svc: svc}
}

// ListAnomalies handles GET /api/v1/monitors/:id/anomalies?window=24h
func (h *AnomalyHandler) ListAnomalies(c echo.Context) error {
	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}
	window := parseAnomalyWindow(c.QueryParam("window"), defaultAnomalyListWindow, maxAnomalyListWindow)
	anomalies, err := h.svc.RecentAnomalies(c.Request().Context(), monitorID, window)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to compute anomalies")
	}
	if anomalies == nil {
		anomalies = []domain.LatencyAnomaly{}
	}
	return c.JSON(http.StatusOK, map[string]any{"data": anomalies})
}

// CountAnomalies handles GET /api/v1/monitors/:id/anomalies/count?window=1h
func (h *AnomalyHandler) CountAnomalies(c echo.Context) error {
	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}
	window := parseAnomalyWindow(c.QueryParam("window"), defaultAnomalyCountWindow, maxAnomalyListWindow)
	count, err := h.svc.CountSince(c.Request().Context(), monitorID, window)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to compute anomaly count")
	}
	return c.JSON(http.StatusOK, map[string]int{"count": count})
}

// parseAnomalyWindow accepts a Go duration string OR integer seconds. Falls
// back to defWindow on parse error or out-of-range value.
func parseAnomalyWindow(raw string, defWindow, max time.Duration) time.Duration {
	if raw == "" {
		return defWindow
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 && d <= max {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		d := time.Duration(secs) * time.Second
		if d <= max {
			return d
		}
	}
	return defWindow
}

