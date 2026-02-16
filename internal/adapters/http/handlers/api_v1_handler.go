package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// APIV1Handler serves the public JSON API endpoints (token-authenticated).
type APIV1Handler struct {
	agentRepo     ports.AgentRepository
	monitorRepo   ports.MonitorRepository
	heartbeatRepo ports.HeartbeatRepository
	incidentSvc   ports.IncidentService
}

// NewAPIV1Handler creates a new APIV1Handler.
func NewAPIV1Handler(
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	heartbeatRepo ports.HeartbeatRepository,
	incidentSvc ports.IncidentService,
) *APIV1Handler {
	return &APIV1Handler{
		agentRepo:     agentRepo,
		monitorRepo:   monitorRepo,
		heartbeatRepo: heartbeatRepo,
		incidentSvc:   incidentSvc,
	}
}

type monitorResponse struct {
	ID       string `json:"id"`
	AgentID  string `json:"agent_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Status   string `json:"status"`
	Enabled  bool   `json:"enabled"`
	Interval int    `json:"interval_seconds"`
	Timeout  int    `json:"timeout_seconds"`
}

type agentResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	LastSeenAt *string `json:"last_seen_at"`
}

type incidentResponse struct {
	ID             string  `json:"id"`
	MonitorID      string  `json:"monitor_id"`
	Status         string  `json:"status"`
	StartedAt      string  `json:"started_at"`
	ResolvedAt     *string `json:"resolved_at"`
	AcknowledgedAt *string `json:"acknowledged_at"`
}

// ListMonitors returns all monitors for the authenticated user.
// GET /api/v1/monitors
func (h *APIV1Handler) ListMonitors(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch agents"})
	}

	var monitors []monitorResponse
	for _, agent := range agents {
		agentMonitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err != nil {
			continue
		}
		for _, m := range agentMonitors {
			monitors = append(monitors, monitorResponse{
				ID:       m.ID.String(),
				AgentID:  m.AgentID.String(),
				Name:     m.Name,
				Type:     string(m.Type),
				Target:   m.Target,
				Status:   string(m.Status),
				Enabled:  m.Enabled,
				Interval: m.IntervalSeconds,
				Timeout:  m.TimeoutSeconds,
			})
		}
	}

	if monitors == nil {
		monitors = []monitorResponse{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": monitors,
	})
}

// GetMonitor returns a single monitor by ID.
// GET /api/v1/monitors/:id
func (h *APIV1Handler) GetMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	monitor, err := h.monitorRepo.GetByID(ctx, monitorID)
	if err != nil || monitor == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	// Verify ownership: monitor's agent must belong to user
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	// Fetch recent heartbeats
	heartbeats, _ := h.heartbeatRepo.GetByMonitorID(ctx, monitorID, 20)
	var latencies []int
	up, down := 0, 0
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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": monitorResponse{
			ID:       monitor.ID.String(),
			AgentID:  monitor.AgentID.String(),
			Name:     monitor.Name,
			Type:     string(monitor.Type),
			Target:   monitor.Target,
			Status:   string(monitor.Status),
			Enabled:  monitor.Enabled,
			Interval: monitor.IntervalSeconds,
			Timeout:  monitor.TimeoutSeconds,
		},
		"heartbeats": map[string]interface{}{
			"latencies":  latencies,
			"uptime_up":  up,
			"uptime_down": down,
			"total":      len(heartbeats),
		},
	})
}

// ListAgents returns all agents for the authenticated user.
// GET /api/v1/agents
func (h *APIV1Handler) ListAgents(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch agents"})
	}

	var result []agentResponse
	for _, a := range agents {
		resp := agentResponse{
			ID:     a.ID.String(),
			Name:   a.Name,
			Status: string(a.Status),
		}
		if a.LastSeenAt != nil {
			t := a.LastSeenAt.Format(time.RFC3339)
			resp.LastSeenAt = &t
		}
		result = append(result, resp)
	}

	if result == nil {
		result = []agentResponse{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": result,
	})
}

// ListIncidents returns all incidents for the authenticated user.
// GET /api/v1/incidents?status=open|acknowledged|resolved|all
func (h *APIV1Handler) ListIncidents(c echo.Context) error {
	ctx := c.Request().Context()
	_, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	status := c.QueryParam("status")
	var incidents []*struct {
		ID             uuid.UUID
		MonitorID      uuid.UUID
		Status         string
		StartedAt      time.Time
		ResolvedAt     *time.Time
		AcknowledgedAt *time.Time
	}

	switch status {
	case "resolved":
		raw, err := h.incidentSvc.GetResolvedIncidents(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch incidents"})
		}
		for _, i := range raw {
			incidents = append(incidents, &struct {
				ID             uuid.UUID
				MonitorID      uuid.UUID
				Status         string
				StartedAt      time.Time
				ResolvedAt     *time.Time
				AcknowledgedAt *time.Time
			}{i.ID, i.MonitorID, string(i.Status), i.StartedAt, i.ResolvedAt, i.AcknowledgedAt})
		}
	default:
		raw, err := h.incidentSvc.GetActiveIncidents(ctx)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch incidents"})
		}
		for _, i := range raw {
			incidents = append(incidents, &struct {
				ID             uuid.UUID
				MonitorID      uuid.UUID
				Status         string
				StartedAt      time.Time
				ResolvedAt     *time.Time
				AcknowledgedAt *time.Time
			}{i.ID, i.MonitorID, string(i.Status), i.StartedAt, i.ResolvedAt, i.AcknowledgedAt})
		}
	}

	var result []incidentResponse
	for _, i := range incidents {
		resp := incidentResponse{
			ID:        i.ID.String(),
			MonitorID: i.MonitorID.String(),
			Status:    i.Status,
			StartedAt: i.StartedAt.Format(time.RFC3339),
		}
		if i.ResolvedAt != nil {
			t := i.ResolvedAt.Format(time.RFC3339)
			resp.ResolvedAt = &t
		}
		if i.AcknowledgedAt != nil {
			t := i.AcknowledgedAt.Format(time.RFC3339)
			resp.AcknowledgedAt = &t
		}
		result = append(result, resp)
	}

	if result == nil {
		result = []incidentResponse{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": result,
	})
}
