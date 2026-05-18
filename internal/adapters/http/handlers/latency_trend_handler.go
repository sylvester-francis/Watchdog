package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type latencyTrendService interface {
	GetTrend(ctx context.Context, monitorID uuid.UUID, window domain.TrendWindow, now time.Time) (*domain.LatencyTrend, error)
}

type LatencyTrendHandler struct {
	svc latencyTrendService
	now func() time.Time
}

func NewLatencyTrendHandler(svc latencyTrendService) *LatencyTrendHandler {
	return &LatencyTrendHandler{svc: svc, now: func() time.Time { return time.Now().UTC() }}
}

// GetLatencyTrend serves p50/p95/p99 buckets plus current+previous period
// summaries. Window selectable via ?window=7d|30d|90d (defaults to 7d).
func (h *LatencyTrendHandler) GetLatencyTrend(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid monitor id")
	}
	window := domain.ParseTrendWindow(c.QueryParam("window"))
	trend, err := h.svc.GetTrend(c.Request().Context(), id, window, h.now())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "load latency trend")
	}
	return c.JSON(http.StatusOK, trend)
}
