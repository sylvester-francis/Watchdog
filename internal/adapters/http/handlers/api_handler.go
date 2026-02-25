package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// HeartbeatPoint is the JSON-serializable heartbeat for chart data.
type HeartbeatPoint struct {
	Time           string  `json:"time"`
	Status         string  `json:"status"`
	LatencyMs      *int    `json:"latency_ms"`
	ErrorMessage   *string `json:"error_message,omitempty"`
	CertExpiryDays *int    `json:"cert_expiry_days,omitempty"`
	CertIssuer     *string `json:"cert_issuer,omitempty"`
}

// APIHandler serves JSON endpoints for chart data.
type APIHandler struct {
	heartbeatRepo ports.HeartbeatRepository
	monitorRepo   ports.MonitorRepository
	agentRepo     ports.AgentRepository
	incidentSvc   ports.IncidentService
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(
	heartbeatRepo ports.HeartbeatRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	incidentSvc ports.IncidentService,
) *APIHandler {
	return &APIHandler{
		heartbeatRepo: heartbeatRepo,
		monitorRepo:   monitorRepo,
		agentRepo:     agentRepo,
		incidentSvc:   incidentSvc,
	}
}

// MonitorHeartbeats returns heartbeat data for a single monitor.
// GET /api/monitors/:id/heartbeats?period=24h|1h|7d|30d
func (h *APIHandler) MonitorHeartbeats(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	monitorID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	// Verify ownership: monitor's agent must belong to user
	monitor, err := verifyMonitorOwnership(ctx, h.monitorRepo, h.agentRepo, monitorID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch heartbeats"})
	}
	if monitor == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	period := c.QueryParam("period")
	var duration time.Duration
	switch period {
	case "1h":
		duration = time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	now := time.Now()
	from := now.Add(-duration)

	heartbeats, err := h.heartbeatRepo.GetByMonitorIDInRange(ctx, monitorID, from, now)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch heartbeats"})
	}

	points := make([]HeartbeatPoint, 0, len(heartbeats))
	for _, hb := range heartbeats {
		points = append(points, HeartbeatPoint{
			Time:           hb.Time.Format(time.RFC3339),
			Status:         string(hb.Status),
			LatencyMs:      hb.LatencyMs,
			ErrorMessage:   hb.ErrorMessage,
			CertExpiryDays: hb.CertExpiryDays,
			CertIssuer:     hb.CertIssuer,
		})
	}

	return c.JSON(http.StatusOK, points)
}

// LatencyHistoryPoint is the JSON representation of an aggregated latency bucket.
type LatencyHistoryPoint struct {
	Time  string `json:"time"`
	AvgMs int    `json:"avg_ms"`
	MinMs int    `json:"min_ms"`
	MaxMs int    `json:"max_ms"`
}

// MonitorLatencyHistory returns aggregated latency data for charts.
// GET /api/monitors/:id/latency?period=1h|24h|7d|30d
func (h *APIHandler) MonitorLatencyHistory(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	monitor, err := verifyMonitorOwnership(ctx, h.monitorRepo, h.agentRepo, monitorID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify ownership"})
	}
	if monitor == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	// Map period to duration and bucket interval
	var duration time.Duration
	var bucket string
	switch c.QueryParam("period") {
	case "1h":
		duration = time.Hour
		bucket = "1 minute"
	case "7d":
		duration = 7 * 24 * time.Hour
		bucket = "1 hour"
	case "30d":
		duration = 30 * 24 * time.Hour
		bucket = "6 hours"
	default: // 24h
		duration = 24 * time.Hour
		bucket = "15 minutes"
	}

	since := time.Now().Add(-duration)
	points, err := h.heartbeatRepo.GetLatencyHistory(ctx, monitorID, since, bucket)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch latency history"})
	}

	result := make([]LatencyHistoryPoint, 0, len(points))
	for _, p := range points {
		result = append(result, LatencyHistoryPoint{
			Time:  p.Time.Format(time.RFC3339),
			AvgMs: p.AvgMs,
			MinMs: p.MinMs,
			MaxMs: p.MaxMs,
		})
	}

	return c.JSON(http.StatusOK, result)
}

// DashboardStats returns summary stats as JSON.
// GET /api/dashboard/stats
func (h *APIHandler) DashboardStats(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch agents"})
	}

	totalAgents := len(agents)
	onlineAgents := 0
	totalMonitors := 0
	monitorsUp := 0
	monitorsDown := 0

	userMonitorIDs := make(map[uuid.UUID]struct{})
	for _, agent := range agents {
		if agent.IsOnline() {
			onlineAgents++
		}
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err == nil {
			totalMonitors += len(monitors)
			for _, m := range monitors {
				userMonitorIDs[m.ID] = struct{}{}
				if m.Status == "up" {
					monitorsUp++
				} else if m.Status == "down" {
					monitorsDown++
				}
			}
		}
	}

	// Filter active incidents to only those belonging to the user's monitors.
	allIncidents, err := h.incidentSvc.GetActiveIncidents(ctx)
	activeIncidents := 0
	if err == nil {
		for _, inc := range allIncidents {
			if _, ok := userMonitorIDs[inc.MonitorID]; ok {
				activeIncidents++
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]int{
		"totalMonitors":   totalMonitors,
		"monitorsUp":      monitorsUp,
		"monitorsDown":    monitorsDown,
		"totalAgents":     totalAgents,
		"onlineAgents":    onlineAgents,
		"activeIncidents": activeIncidents,
	})
}

// MonitorsSummary returns each monitor with recent heartbeat latency for sparklines.
// GET /api/monitors/summary?heartbeats=20
func (h *APIHandler) MonitorsSummary(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch agents"})
	}

	type MonitorSummary struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Status          string `json:"status"`
		Type            string `json:"type"`
		Target          string `json:"target"`
		IntervalSeconds int    `json:"interval_seconds"`
		Latencies       []int  `json:"latencies"`
		UptimeUp        int    `json:"uptimeUp"`
		UptimeDown      int    `json:"uptimeDown"`
		Total           int    `json:"total"`
		LatestValue     string `json:"latest_value,omitempty"`
	}

	var summaries []MonitorSummary
	for _, agent := range agents {
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err != nil {
			continue
		}
		for _, monitor := range monitors {
			heartbeats, err := h.heartbeatRepo.GetByMonitorID(ctx, monitor.ID, 20)
			if err != nil {
				heartbeats = nil
			}

			latencies := make([]int, 0, len(heartbeats))
			up, down := 0, 0
			var latestValue string
			// Reverse so oldest first (chronological)
			for i := len(heartbeats) - 1; i >= 0; i-- {
				hb := heartbeats[i]
				if hb.LatencyMs != nil {
					latencies = append(latencies, *hb.LatencyMs)
				}
				if hb.Status.IsSuccess() {
					up++
				} else {
					down++
				}
			}
			// Extract latest value from most recent heartbeat (index 0 = newest)
			if len(heartbeats) > 0 {
				latest := heartbeats[0]
				if latest.ErrorMessage != nil && *latest.ErrorMessage != "" {
					latestValue = *latest.ErrorMessage
				}
			}

			summaries = append(summaries, MonitorSummary{
				ID:              monitor.ID.String(),
				Name:            monitor.Name,
				Status:          string(monitor.Status),
				Type:            string(monitor.Type),
				Target:          monitor.Target,
				IntervalSeconds: monitor.IntervalSeconds,
				Latencies:       latencies,
				UptimeUp:        up,
				UptimeDown:      down,
				Total:           len(heartbeats),
				LatestValue:     latestValue,
			})
		}
	}

	return c.JSON(http.StatusOK, summaries)
}
