package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// APIV1Handler serves the public JSON API endpoints (token-authenticated).
type APIV1Handler struct {
	agentRepo     ports.AgentRepository
	monitorRepo   ports.MonitorRepository
	heartbeatRepo ports.HeartbeatRepository
	incidentSvc   ports.IncidentService
	monitorSvc    ports.MonitorService
	agentAuthSvc  ports.AgentAuthService
}

// NewAPIV1Handler creates a new APIV1Handler.
func NewAPIV1Handler(
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	heartbeatRepo ports.HeartbeatRepository,
	incidentSvc ports.IncidentService,
	monitorSvc ports.MonitorService,
	agentAuthSvc ports.AgentAuthService,
) *APIV1Handler {
	return &APIV1Handler{
		agentRepo:     agentRepo,
		monitorRepo:   monitorRepo,
		heartbeatRepo: heartbeatRepo,
		incidentSvc:   incidentSvc,
		monitorSvc:    monitorSvc,
		agentAuthSvc:  agentAuthSvc,
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

// --- CRUD endpoints ---

type createMonitorRequest struct {
	AgentID  string `json:"agent_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Interval int    `json:"interval_seconds"`
	Timeout  int    `json:"timeout_seconds"`
}

// CreateMonitor creates a new monitor.
// POST /api/v1/monitors
func (h *APIV1Handler) CreateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req createMonitorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" || req.Type == "" || req.Target == "" || req.AgentID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name, type, target, and agent_id are required"})
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent_id"})
	}

	// Verify agent ownership
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
	}

	monitor, err := h.monitorSvc.CreateMonitor(ctx, userID, agentID, req.Name, domain.MonitorType(req.Type), req.Target)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Apply optional interval/timeout
	if req.Interval > 0 {
		monitor.SetInterval(req.Interval)
	}
	if req.Timeout > 0 {
		monitor.SetTimeout(req.Timeout)
	}
	if req.Interval > 0 || req.Timeout > 0 {
		_ = h.monitorSvc.UpdateMonitor(ctx, monitor)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
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
	})
}

type updateMonitorRequest struct {
	Name     *string `json:"name"`
	Target   *string `json:"target"`
	Interval *int    `json:"interval_seconds"`
	Timeout  *int    `json:"timeout_seconds"`
	Enabled  *bool   `json:"enabled"`
}

// UpdateMonitor updates an existing monitor.
// PUT /api/v1/monitors/:id
func (h *APIV1Handler) UpdateMonitor(c echo.Context) error {
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

	// Verify ownership
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	var req updateMonitorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name != nil {
		monitor.Name = *req.Name
	}
	if req.Target != nil {
		monitor.Target = *req.Target
	}
	if req.Interval != nil {
		monitor.SetInterval(*req.Interval)
	}
	if req.Timeout != nil {
		monitor.SetTimeout(*req.Timeout)
	}
	if req.Enabled != nil {
		if *req.Enabled {
			monitor.Enable()
		} else {
			monitor.Disable()
		}
	}

	if err := h.monitorSvc.UpdateMonitor(ctx, monitor); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update monitor"})
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
	})
}

// DeleteMonitor deletes a monitor.
// DELETE /api/v1/monitors/:id
func (h *APIV1Handler) DeleteMonitor(c echo.Context) error {
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

	// Verify ownership
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	if err := h.monitorSvc.DeleteMonitor(ctx, monitorID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete monitor"})
	}

	return c.NoContent(http.StatusNoContent)
}

type createAgentRequest struct {
	Name string `json:"name"`
}

// CreateAgent creates a new agent and returns its API key.
// POST /api/v1/agents
func (h *APIV1Handler) CreateAgent(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req createAgentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	agent, apiKey, err := h.agentAuthSvc.CreateAgent(ctx, userID.String(), req.Name)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"data": map[string]string{
			"id":      agent.ID.String(),
			"name":    agent.Name,
			"api_key": apiKey,
		},
	})
}

// DeleteAgent deletes an agent.
// DELETE /api/v1/agents/:id
func (h *APIV1Handler) DeleteAgent(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent ID"})
	}

	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
	}

	if err := h.agentRepo.Delete(ctx, agentID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete agent"})
	}

	return c.NoContent(http.StatusNoContent)
}

// AcknowledgeIncident acknowledges an incident.
// POST /api/v1/incidents/:id/acknowledge
func (h *APIV1Handler) AcknowledgeIncident(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	if err := h.incidentSvc.AcknowledgeIncident(ctx, incidentID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to acknowledge incident"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "acknowledged"})
}

// ResolveIncident resolves an incident.
// POST /api/v1/incidents/:id/resolve
func (h *APIV1Handler) ResolveIncident(c echo.Context) error {
	ctx := c.Request().Context()
	_, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	if err := h.incidentSvc.ResolveIncident(ctx, incidentID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to resolve incident"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resolved"})
}

// DashboardStats returns summary statistics.
// GET /api/v1/dashboard/stats
func (h *APIV1Handler) DashboardStats(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch stats"})
	}

	totalAgents := len(agents)
	onlineAgents := 0
	totalMonitors := 0
	monitorsUp := 0
	monitorsDown := 0

	for _, a := range agents {
		if a.Status == domain.AgentStatusOnline {
			onlineAgents++
		}
		monitors, err := h.monitorRepo.GetByAgentID(ctx, a.ID)
		if err != nil {
			continue
		}
		for _, m := range monitors {
			totalMonitors++
			switch m.Status {
			case domain.MonitorStatusUp:
				monitorsUp++
			case domain.MonitorStatusDown:
				monitorsDown++
			}
		}
	}

	activeIncidents, _ := h.incidentSvc.GetActiveIncidents(ctx)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_monitors":   totalMonitors,
		"monitors_up":      monitorsUp,
		"monitors_down":    monitorsDown,
		"active_incidents": len(activeIncidents),
		"total_agents":     totalAgents,
		"online_agents":    onlineAgents,
	})
}
