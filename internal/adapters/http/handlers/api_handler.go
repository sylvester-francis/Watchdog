package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// HeartbeatPoint is the JSON-serializable heartbeat for chart data.
type HeartbeatPoint struct {
	Time      string `json:"time"`
	Status    string `json:"status"`
	LatencyMs *int   `json:"latencyMs"`
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

	idStr := c.Param("id")
	monitorID, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
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
			Time:      hb.Time.Format(time.RFC3339),
			Status:    string(hb.Status),
			LatencyMs: hb.LatencyMs,
		})
	}

	return c.JSON(http.StatusOK, points)
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

	for _, agent := range agents {
		if agent.IsOnline() {
			onlineAgents++
		}
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err == nil {
			totalMonitors += len(monitors)
			for _, m := range monitors {
				if m.Status == "up" {
					monitorsUp++
				} else if m.Status == "down" {
					monitorsDown++
				}
			}
		}
	}

	incidents, err := h.incidentSvc.GetActiveIncidents(ctx)
	activeIncidents := 0
	if err == nil {
		activeIncidents = len(incidents)
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
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Type      string `json:"type"`
		Target    string `json:"target"`
		Latencies []int  `json:"latencies"`
		UptimeUp  int    `json:"uptimeUp"`
		UptimeDown int   `json:"uptimeDown"`
		Total     int    `json:"total"`
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

			summaries = append(summaries, MonitorSummary{
				ID:         monitor.ID.String(),
				Name:       monitor.Name,
				Status:     string(monitor.Status),
				Type:       string(monitor.Type),
				Target:     monitor.Target,
				Latencies:  latencies,
				UptimeUp:   up,
				UptimeDown: down,
				Total:      len(heartbeats),
			})
		}
	}

	return c.JSON(http.StatusOK, summaries)
}
