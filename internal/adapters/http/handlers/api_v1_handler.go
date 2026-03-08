package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

// APIV1Handler serves the public JSON API endpoints (token-authenticated).
type APIV1Handler struct {
	agentRepo        ports.AgentRepository
	monitorRepo      ports.MonitorRepository
	heartbeatRepo    ports.HeartbeatRepository
	certDetailsRepo  ports.CertDetailsRepository
	incidentSvc      ports.IncidentService
	monitorSvc       ports.MonitorService
	agentAuthSvc     ports.AgentAuthService
	hub              *realtime.Hub
	auditSvc         ports.AuditService
	investigationSvc ports.InvestigationService
	updateSvc        *services.UpdateService
}

// NewAPIV1Handler creates a new APIV1Handler.
func NewAPIV1Handler(
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	heartbeatRepo ports.HeartbeatRepository,
	certDetailsRepo ports.CertDetailsRepository,
	incidentSvc ports.IncidentService,
	monitorSvc ports.MonitorService,
	agentAuthSvc ports.AgentAuthService,
	hub *realtime.Hub,
	auditSvc ports.AuditService,
) *APIV1Handler {
	return &APIV1Handler{
		agentRepo:       agentRepo,
		monitorRepo:     monitorRepo,
		heartbeatRepo:   heartbeatRepo,
		certDetailsRepo: certDetailsRepo,
		incidentSvc:     incidentSvc,
		monitorSvc:      monitorSvc,
		agentAuthSvc:    agentAuthSvc,
		hub:             hub,
		auditSvc:        auditSvc,
	}
}

type monitorResponse struct {
	ID               string            `json:"id"`
	AgentID          string            `json:"agent_id"`
	AgentName        string            `json:"agent_name"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Target           string            `json:"target"`
	Status           string            `json:"status"`
	Enabled          bool              `json:"enabled"`
	Interval         int               `json:"interval_seconds"`
	Timeout          int               `json:"timeout_seconds"`
	FailureThreshold int               `json:"failure_threshold"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	SLATargetPercent *float64          `json:"sla_target_percent,omitempty"`
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
	MonitorName    string  `json:"monitor_name"`
	Status         string  `json:"status"`
	StartedAt      string  `json:"started_at"`
	ResolvedAt     *string `json:"resolved_at"`
	AcknowledgedAt *string `json:"acknowledged_at"`
	TTRSeconds     *int    `json:"ttr_seconds"`
}

// ListMonitors returns all monitors for the authenticated user.
// GET /api/v1/monitors
func (h *APIV1Handler) ListMonitors(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch agents")
	}

	var monitors []monitorResponse
	for _, agent := range agents {
		agentMonitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err != nil {
			continue
		}
		for _, m := range agentMonitors {
			monitors = append(monitors, monitorResponse{
				ID:               m.ID.String(),
				AgentID:          m.AgentID.String(),
				AgentName:        agent.Name,
				Name:             m.Name,
				Type:             string(m.Type),
				Target:           m.Target,
				Status:           string(m.Status),
				Enabled:          m.Enabled,
				Interval:         m.IntervalSeconds,
				Timeout:          m.TimeoutSeconds,
				FailureThreshold: m.FailureThreshold,
				SLATargetPercent: m.SLATargetPercent,
			})
		}
	}

	if monitors == nil {
		monitors = []monitorResponse{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": monitors,
	})
}

// GetMonitor returns a single monitor by ID.
// GET /api/v1/monitors/:id
func (h *APIV1Handler) GetMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}

	monitor, err := h.monitorRepo.GetByID(ctx, monitorID)
	if err != nil || monitor == nil {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	// Verify ownership: monitor's agent must belong to user
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "monitor not found")
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

	// Build metadata — start with monitor's own metadata, then overlay live data from heartbeats
	meta := make(map[string]string)
	for k, v := range monitor.Metadata {
		meta[k] = v
	}
	// For TLS monitors, inject latest cert data from the most recent heartbeat
	if monitor.Type == domain.MonitorTypeTLS && len(heartbeats) > 0 {
		latest := heartbeats[0] // heartbeats are ordered DESC, so [0] is most recent
		if latest.CertExpiryDays != nil {
			meta["cert_expiry_days"] = fmt.Sprintf("%d", *latest.CertExpiryDays)
		}
		if latest.CertIssuer != nil && *latest.CertIssuer != "" {
			meta["cert_issuer"] = *latest.CertIssuer
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": monitorResponse{
			ID:               monitor.ID.String(),
			AgentID:          monitor.AgentID.String(),
			AgentName:        agent.Name,
			Name:             monitor.Name,
			Type:             string(monitor.Type),
			Target:           monitor.Target,
			Status:           string(monitor.Status),
			Enabled:          monitor.Enabled,
			Interval:         monitor.IntervalSeconds,
			Timeout:          monitor.TimeoutSeconds,
			FailureThreshold: monitor.FailureThreshold,
			SLATargetPercent: monitor.SLATargetPercent,
			Metadata:         meta,
		},
		"heartbeats": map[string]any{
			"latencies":   latencies,
			"uptime_up":   up,
			"uptime_down": down,
			"total":       len(heartbeats),
		},
	})
}

// ListAgents returns all agents for the authenticated user.
// GET /api/v1/agents
func (h *APIV1Handler) ListAgents(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch agents")
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

	return c.JSON(http.StatusOK, map[string]any{
		"data": result,
	})
}

// ListIncidents returns all incidents for the authenticated user.
// GET /api/v1/incidents?status=open|acknowledged|resolved|all
func (h *APIV1Handler) ListIncidents(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	status := c.QueryParam("status")
	var rawIncidents []*domain.Incident
	var err error

	switch status {
	case "resolved":
		rawIncidents, err = h.incidentSvc.GetResolvedIncidents(ctx)
	case "", "all":
		// Fetch both active and resolved for "all" view
		active, err1 := h.incidentSvc.GetActiveIncidents(ctx)
		resolved, err2 := h.incidentSvc.GetResolvedIncidents(ctx)
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		} else {
			rawIncidents = append(active, resolved...)
		}
	default:
		rawIncidents, err = h.incidentSvc.GetActiveIncidents(ctx)
	}
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch incidents")
	}

	// Filter incidents to only those belonging to the user's monitors.
	// Build a monitor name map while iterating.
	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch incidents")
	}
	monitorNames := make(map[uuid.UUID]string)
	for _, agent := range agents {
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err != nil {
			continue
		}
		for _, m := range monitors {
			monitorNames[m.ID] = m.Name
		}
	}

	var result []incidentResponse
	for _, i := range rawIncidents {
		if _, owns := monitorNames[i.MonitorID]; !owns {
			continue
		}
		resp := incidentResponse{
			ID:          i.ID.String(),
			MonitorID:   i.MonitorID.String(),
			MonitorName: monitorNames[i.MonitorID],
			Status:      string(i.Status),
			StartedAt:   i.StartedAt.Format(time.RFC3339),
			TTRSeconds:  i.TTRSeconds,
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

	return c.JSON(http.StatusOK, map[string]any{
		"data": result,
	})
}

// --- CRUD endpoints ---

type createMonitorRequest struct {
	AgentID          string            `json:"agent_id"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Target           string            `json:"target"`
	Interval         int               `json:"interval_seconds"`
	Timeout          int               `json:"timeout_seconds"`
	FailureThreshold *int              `json:"failure_threshold,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	SLATargetPercent *float64          `json:"sla_target_percent,omitempty"`
}

// CreateMonitor creates a new monitor.
// POST /api/v1/monitors
func (h *APIV1Handler) CreateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	var req createMonitorRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.Type == "" || req.Target == "" || req.AgentID == "" {
		return errJSON(c, http.StatusBadRequest, "name, type, target, and agent_id are required")
	}

	if !domain.MonitorType(req.Type).IsValid() {
		return errJSON(c, http.StatusBadRequest, fmt.Sprintf("invalid monitor type: %s", req.Type))
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid agent_id")
	}

	// Verify agent ownership
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "agent not found")
	}

	monitor, err := h.monitorSvc.CreateMonitor(ctx, userID, agentID, req.Name, domain.MonitorType(req.Type), req.Target, req.Metadata)
	if err != nil {
		if errors.Is(err, domain.ErrMonitorLimitReached) {
			return errJSONCode(c, http.StatusForbidden, CodeLimitReached, err.Error())
		}
		return errJSON(c, http.StatusBadRequest, "failed to create monitor")
	}

	// Apply optional interval/timeout/failure_threshold
	if req.Interval > 0 {
		monitor.SetInterval(req.Interval)
	}
	if req.Timeout > 0 {
		monitor.SetTimeout(req.Timeout)
	}
	if req.FailureThreshold != nil {
		ft := *req.FailureThreshold
		if ft < domain.MinFailureThreshold || ft > domain.MaxFailureThreshold {
			return errJSON(c, http.StatusBadRequest, fmt.Sprintf("failure_threshold must be between %d and %d", domain.MinFailureThreshold, domain.MaxFailureThreshold))
		}
		monitor.FailureThreshold = ft
	}
	if req.SLATargetPercent != nil {
		sla := *req.SLATargetPercent
		if sla < 0 || sla > 100 {
			return errJSON(c, http.StatusBadRequest, "sla_target_percent must be between 0 and 100")
		}
		monitor.SLATargetPercent = req.SLATargetPercent
	}
	if req.Interval > 0 || req.Timeout > 0 || req.FailureThreshold != nil || req.SLATargetPercent != nil {
		if err := h.monitorSvc.UpdateMonitor(ctx, monitor); err != nil {
			return errJSON(c, http.StatusInternalServerError, "monitor created but failed to apply settings")
		}
	}

	// Notify agent if connected
	taskMsg := protocol.NewTaskMessageWithMetadata(
		monitor.ID.String(), string(monitor.Type),
		monitor.Target, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Metadata,
	)
	h.hub.SendToAgent(monitor.AgentID, taskMsg)

	// H-011: audit monitor creation.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorCreated, c.RealIP(), map[string]string{
			"monitor_id": monitor.ID.String(), "name": monitor.Name, "type": string(monitor.Type),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": monitorResponse{
			ID:               monitor.ID.String(),
			AgentID:          monitor.AgentID.String(),
			AgentName:        agent.Name,
			Name:             monitor.Name,
			Type:             string(monitor.Type),
			Target:           monitor.Target,
			Status:           string(monitor.Status),
			Enabled:          monitor.Enabled,
			Interval:         monitor.IntervalSeconds,
			Timeout:          monitor.TimeoutSeconds,
			FailureThreshold: monitor.FailureThreshold,
			SLATargetPercent: monitor.SLATargetPercent,
		},
	})
}

type updateMonitorRequest struct {
	Name             *string  `json:"name"`
	Target           *string  `json:"target"`
	Interval         *int     `json:"interval_seconds"`
	Timeout          *int     `json:"timeout_seconds"`
	FailureThreshold *int     `json:"failure_threshold"`
	Enabled          *bool    `json:"enabled"`
	SLATargetPercent *float64 `json:"sla_target_percent"`
	AgentID          *string  `json:"agent_id"`
}

// UpdateMonitor updates an existing monitor.
// PUT /api/v1/monitors/:id
func (h *APIV1Handler) UpdateMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}

	monitor, err := h.monitorRepo.GetByID(ctx, monitorID)
	if err != nil || monitor == nil {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	// Verify ownership
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	var req updateMonitorRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
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
	if req.FailureThreshold != nil {
		ft := *req.FailureThreshold
		if ft < domain.MinFailureThreshold || ft > domain.MaxFailureThreshold {
			return errJSON(c, http.StatusBadRequest, fmt.Sprintf("failure_threshold must be between %d and %d", domain.MinFailureThreshold, domain.MaxFailureThreshold))
		}
		monitor.FailureThreshold = ft
	}
	if req.SLATargetPercent != nil {
		sla := *req.SLATargetPercent
		if sla < 0 || sla > 100 {
			return errJSON(c, http.StatusBadRequest, "sla_target_percent must be between 0 and 100")
		}
		monitor.SLATargetPercent = req.SLATargetPercent
	}
	oldAgentID := monitor.AgentID
	if req.AgentID != nil {
		newAgentID, err := uuid.Parse(*req.AgentID)
		if err != nil {
			return errJSON(c, http.StatusBadRequest, "invalid agent_id")
		}
		// Only validate and reassign if the agent is actually changing
		if newAgentID != oldAgentID {
			newAgent, err := h.agentRepo.GetByID(ctx, newAgentID)
			if err != nil || newAgent == nil || newAgent.UserID != userID {
				return errJSON(c, http.StatusBadRequest, "agent not found or not owned by you")
			}
			monitor.AgentID = newAgentID
		}
	}

	if err := h.monitorSvc.UpdateMonitor(ctx, monitor); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to update monitor")
	}

	// H-011: audit monitor update.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorUpdated, c.RealIP(), map[string]string{
			"monitor_id": monitor.ID.String(), "name": monitor.Name,
		})
	}

	// If agent changed, cancel the monitor on the old agent
	if oldAgentID != monitor.AgentID {
		h.hub.SendToAgent(oldAgentID, protocol.NewTaskCancelMessage(monitor.ID.String()))
	}

	// Notify the (possibly new) agent of the task
	if monitor.Enabled {
		taskMsg := protocol.NewTaskMessageWithMetadata(
			monitor.ID.String(), string(monitor.Type),
			monitor.Target, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Metadata,
		)
		h.hub.SendToAgent(monitor.AgentID, taskMsg)
	} else {
		h.hub.SendToAgent(monitor.AgentID, protocol.NewTaskCancelMessage(monitor.ID.String()))
	}

	// Resolve agent name — use current (possibly reassigned) agent.
	agentName := agent.Name
	if monitor.AgentID != agent.ID {
		if newAgent, err := h.agentRepo.GetByID(ctx, monitor.AgentID); err == nil && newAgent != nil {
			agentName = newAgent.Name
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": monitorResponse{
			ID:               monitor.ID.String(),
			AgentID:          monitor.AgentID.String(),
			AgentName:        agentName,
			Name:             monitor.Name,
			Type:             string(monitor.Type),
			Target:           monitor.Target,
			Status:           string(monitor.Status),
			Enabled:          monitor.Enabled,
			Interval:         monitor.IntervalSeconds,
			Timeout:          monitor.TimeoutSeconds,
			FailureThreshold: monitor.FailureThreshold,
			SLATargetPercent: monitor.SLATargetPercent,
		},
	})
}

// DeleteMonitor deletes a monitor.
// DELETE /api/v1/monitors/:id
func (h *APIV1Handler) DeleteMonitor(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}

	monitor, err := h.monitorRepo.GetByID(ctx, monitorID)
	if err != nil || monitor == nil {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	// Verify ownership
	agent, err := h.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	if err := h.monitorSvc.DeleteMonitor(ctx, monitorID); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to delete monitor")
	}

	// H-011: audit monitor deletion.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorDeleted, c.RealIP(), map[string]string{
			"monitor_id": monitorID.String(), "name": monitor.Name,
		})
	}

	// Notify agent to stop the task
	h.hub.SendToAgent(monitor.AgentID, protocol.NewTaskCancelMessage(monitorID.String()))

	return c.NoContent(http.StatusNoContent)
}

type createAgentRequest struct {
	Name          string `json:"name"`
	ExpiresInDays *int   `json:"expires_in_days,omitempty"` // H-023: override default key expiry
}

// CreateAgent creates a new agent and returns its API key.
// POST /api/v1/agents
func (h *APIV1Handler) CreateAgent(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	var req createAgentRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return errJSON(c, http.StatusBadRequest, "name is required")
	}

	agent, apiKey, err := h.agentAuthSvc.CreateAgent(ctx, userID.String(), req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrAgentLimitReached) {
			return errJSONCode(c, http.StatusForbidden, CodeLimitReached, err.Error())
		}
		return errJSON(c, http.StatusBadRequest, "failed to create agent")
	}

	// H-023: allow client to override the default key expiry.
	if req.ExpiresInDays != nil {
		days := *req.ExpiresInDays
		if days <= 0 {
			// Zero or negative means "never expires".
			agent.APIKeyExpiresAt = nil
		} else {
			exp := time.Now().AddDate(0, 0, days)
			agent.APIKeyExpiresAt = &exp
		}
		// Persist the custom expiry. Non-fatal if it fails — the agent was
		// already created with the default expiry.
		//nolint:errcheck
		_ = h.agentRepo.Update(ctx, agent)
	}

	// H-011: audit agent creation.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditAgentCreated, c.RealIP(), map[string]string{
			"agent_id": agent.ID.String(), "name": agent.Name,
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
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
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid agent ID")
	}

	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "agent not found")
	}

	if err := h.agentRepo.Delete(ctx, agentID); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to delete agent")
	}

	// H-011: audit agent deletion.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditAgentDeleted, c.RealIP(), map[string]string{
			"agent_id": agentID.String(), "name": agent.Name,
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// AcknowledgeIncident acknowledges an incident.
// POST /api/v1/incidents/:id/acknowledge
func (h *APIV1Handler) AcknowledgeIncident(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid incident ID")
	}

	incident, err := verifyIncidentOwnership(ctx, h.incidentSvc, h.monitorRepo, h.agentRepo, incidentID, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to acknowledge incident")
	}
	if incident == nil {
		return errJSON(c, http.StatusNotFound, "incident not found")
	}

	if err := h.incidentSvc.AcknowledgeIncident(ctx, incidentID, userID); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to acknowledge incident")
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditIncidentAcked, c.RealIP(), map[string]string{
			"incident_id": incidentID.String(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "acknowledged"})
}

// ResolveIncident resolves an incident.
// POST /api/v1/incidents/:id/resolve
func (h *APIV1Handler) ResolveIncident(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid incident ID")
	}

	incident, err := verifyIncidentOwnership(ctx, h.incidentSvc, h.monitorRepo, h.agentRepo, incidentID, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to resolve incident")
	}
	if incident == nil {
		return errJSON(c, http.StatusNotFound, "incident not found")
	}

	if err := h.incidentSvc.ResolveIncident(ctx, incidentID); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to resolve incident")
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditIncidentResolved, c.RealIP(), map[string]string{
			"incident_id": incidentID.String(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resolved"})
}

// DashboardStats returns summary statistics.
// GET /api/v1/dashboard/stats
func (h *APIV1Handler) DashboardStats(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch stats")
	}

	totalAgents := len(agents)
	onlineAgents := 0
	totalMonitors := 0
	monitorsUp := 0
	monitorsDown := 0
	userMonitorIDs := make(map[uuid.UUID]struct{})

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
			userMonitorIDs[m.ID] = struct{}{}
			switch m.Status {
			case domain.MonitorStatusUp:
				monitorsUp++
			case domain.MonitorStatusDown:
				monitorsDown++
			}
		}
	}

	// Filter active incidents to only those belonging to the user's monitors.
	allIncidents, _ := h.incidentSvc.GetActiveIncidents(ctx)
	activeIncidents := 0
	for _, inc := range allIncidents {
		if _, ok := userMonitorIDs[inc.MonitorID]; ok {
			activeIncidents++
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"total_monitors":   totalMonitors,
		"monitors_up":      monitorsUp,
		"monitors_down":    monitorsDown,
		"active_incidents": activeIncidents,
		"total_agents":     totalAgents,
		"online_agents":    onlineAgents,
	})
}

type certDetailsResponse struct {
	MonitorID     string   `json:"monitor_id"`
	LastCheckedAt string   `json:"last_checked_at"`
	ExpiryDays    *int     `json:"expiry_days"`
	Issuer        string   `json:"issuer"`
	SANs          []string `json:"sans"`
	Algorithm     string   `json:"algorithm"`
	KeySize       int      `json:"key_size"`
	SerialNumber  string   `json:"serial_number"`
	ChainValid    bool     `json:"chain_valid"`
}

// GetMonitorCertificate returns TLS certificate details for a monitor.
// GET /api/v1/monitors/:id/certificate
func (h *APIV1Handler) GetMonitorCertificate(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}

	monitor, err := verifyMonitorOwnership(ctx, h.monitorRepo, h.agentRepo, monitorID, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch monitor")
	}
	if monitor == nil {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	if h.certDetailsRepo == nil {
		return errJSON(c, http.StatusNotFound, "certificate tracking not available")
	}

	details, err := h.certDetailsRepo.GetByMonitorID(ctx, monitorID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch certificate details")
	}
	if details == nil {
		return errJSON(c, http.StatusNotFound, "no certificate data available")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": certDetailsResponse{
			MonitorID:     details.MonitorID.String(),
			LastCheckedAt: details.LastCheckedAt.Format(time.RFC3339),
			ExpiryDays:    details.ExpiryDays,
			Issuer:        details.Issuer,
			SANs:          details.SANs,
			Algorithm:     details.Algorithm,
			KeySize:       details.KeySize,
			SerialNumber:  details.SerialNumber,
			ChainValid:    details.ChainValid,
		},
	})
}

// GetExpiringCertificates lists monitors with certs expiring within N days.
// GET /api/v1/certificates/expiring?days=30
func (h *APIV1Handler) GetExpiringCertificates(c echo.Context) error {
	ctx := c.Request().Context()
	_, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	days := 30
	if d := c.QueryParam("days"); d != "" {
		parsed, err := strconv.Atoi(d)
		if err != nil || parsed < 1 || parsed > 365 {
			return errJSON(c, http.StatusBadRequest, "days must be between 1 and 365")
		}
		days = parsed
	}

	if h.certDetailsRepo == nil {
		return c.JSON(http.StatusOK, map[string]any{"data": []certDetailsResponse{}})
	}

	results, err := h.certDetailsRepo.GetExpiring(ctx, days)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch expiring certificates")
	}

	var resp []certDetailsResponse
	for _, d := range results {
		resp = append(resp, certDetailsResponse{
			MonitorID:     d.MonitorID.String(),
			LastCheckedAt: d.LastCheckedAt.Format(time.RFC3339),
			ExpiryDays:    d.ExpiryDays,
			Issuer:        d.Issuer,
			SANs:          d.SANs,
			Algorithm:     d.Algorithm,
			KeySize:       d.KeySize,
			SerialNumber:  d.SerialNumber,
			ChainValid:    d.ChainValid,
		})
	}
	if resp == nil {
		resp = []certDetailsResponse{}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": resp})
}

// GetMonitorSLA returns uptime SLA data for a monitor.
// GET /api/v1/monitors/:id/sla?period=7d|30d|90d
func (h *APIV1Handler) GetMonitorSLA(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	monitorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid monitor ID")
	}

	monitor, err := verifyMonitorOwnership(ctx, h.monitorRepo, h.agentRepo, monitorID, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch monitor")
	}
	if monitor == nil {
		return errJSON(c, http.StatusNotFound, "monitor not found")
	}

	// Parse period
	period := c.QueryParam("period")
	var since time.Time
	periodLabel := "30d"
	switch period {
	case "7d":
		since = time.Now().AddDate(0, 0, -7)
		periodLabel = "7d"
	case "90d":
		since = time.Now().AddDate(0, 0, -90)
		periodLabel = "90d"
	default:
		since = time.Now().AddDate(0, 0, -30)
	}

	uptimePercent, err := h.heartbeatRepo.GetUptimePercent(ctx, monitorID, since)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to calculate uptime")
	}

	slaTarget := 99.9 // default
	if monitor.SLATargetPercent != nil {
		slaTarget = *monitor.SLATargetPercent
	}

	margin := uptimePercent - slaTarget
	breached := uptimePercent < slaTarget

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"uptime_percent": uptimePercent,
			"sla_target":     slaTarget,
			"breached":       breached,
			"margin":         margin,
			"period":         periodLabel,
		},
	})
}

// SetInvestigationService sets the investigation service (wired after construction).
func (h *APIV1Handler) SetInvestigationService(svc ports.InvestigationService) {
	h.investigationSvc = svc
}

// SetUpdateService sets the update service for agent auto-update.
func (h *APIV1Handler) SetUpdateService(svc *services.UpdateService) {
	h.updateSvc = svc
}

// GetIncidentInvestigation returns aggregated investigation data for an incident.
// GET /api/v1/incidents/:id/investigation
func (h *APIV1Handler) GetIncidentInvestigation(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid incident ID")
	}

	// Verify ownership
	incident, err := verifyIncidentOwnership(ctx, h.incidentSvc, h.monitorRepo, h.agentRepo, incidentID, userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch incident")
	}
	if incident == nil {
		return errJSON(c, http.StatusNotFound, "incident not found")
	}

	if h.investigationSvc == nil {
		return errJSON(c, http.StatusNotImplemented, "investigation service not available")
	}

	investigation, err := h.investigationSvc.Investigate(ctx, incidentID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to investigate incident")
	}
	if investigation == nil {
		return errJSON(c, http.StatusNotFound, "incident not found")
	}

	// Map previous incidents to snake_case response format (Incident struct has no JSON tags)
	var prevIncidents []incidentResponse
	for _, inc := range investigation.PreviousIncidents {
		resp := incidentResponse{
			ID:         inc.ID.String(),
			MonitorID:  inc.MonitorID.String(),
			Status:     string(inc.Status),
			StartedAt:  inc.StartedAt.Format(time.RFC3339),
			TTRSeconds: inc.TTRSeconds,
		}
		if inc.ResolvedAt != nil {
			t := inc.ResolvedAt.Format(time.RFC3339)
			resp.ResolvedAt = &t
		}
		if inc.AcknowledgedAt != nil {
			t := inc.AcknowledgedAt.Format(time.RFC3339)
			resp.AcknowledgedAt = &t
		}
		prevIncidents = append(prevIncidents, resp)
	}
	if prevIncidents == nil {
		prevIncidents = []incidentResponse{}
	}

	// Map cert_details to snake_case response format (CertDetails struct has no JSON tags)
	var certDetails *certDetailsResponse
	if investigation.CertDetails != nil {
		certDetails = &certDetailsResponse{
			Issuer:       investigation.CertDetails.Issuer,
			ExpiryDays:   investigation.CertDetails.ExpiryDays,
			Algorithm:    investigation.CertDetails.Algorithm,
			ChainValid:   investigation.CertDetails.ChainValid,
			SerialNumber: investigation.CertDetails.SerialNumber,
			KeySize:      investigation.CertDetails.KeySize,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"agent_summary":      investigation.AgentSummary,
			"recurrence_pattern": investigation.RecurrencePattern,
			"mttr_seconds":       investigation.MTTRSeconds,
			"sibling_monitors":   investigation.SiblingMonitors,
			"previous_incidents": prevIncidents,
			"system_metrics":     investigation.SystemMetrics,
			"cert_details":       certDetails,
			"timeline":           investigation.Timeline,
		},
	})
}

// PushAgentUpdate manually triggers an update push to a specific connected agent.
// POST /api/v1/agents/:id/update
func (h *APIV1Handler) PushAgentUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	if h.updateSvc == nil {
		return errJSON(c, http.StatusNotImplemented, "update service not configured")
	}

	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid agent ID")
	}

	// Verify ownership
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "agent not found")
	}

	// Check if agent is connected
	if !h.hub.IsConnected(agentID) {
		return errJSON(c, http.StatusNotFound, "agent not connected")
	}

	// Determine agent's current version and platform from DB
	agentVersion := agent.Version
	if agentVersion == "" {
		return errJSON(c, http.StatusBadRequest, "agent version unknown")
	}

	agentOS := ""
	agentArch := ""
	if agent.Fingerprint != nil {
		agentOS = agent.Fingerprint["os"]
		agentArch = agent.Fingerprint["arch"]
	}

	// Check manifest for an update
	updateMsg := h.updateSvc.GetUpdateForAgent(agentVersion, agentOS, agentArch)
	if updateMsg == nil {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "already up to date",
		})
	}

	// Push the update message to the agent
	h.hub.SendToAgent(agentID, updateMsg)

	payload := h.updateSvc.GetUpdatePayloadForAgent(agentVersion, agentOS, agentArch)
	return c.JSON(http.StatusOK, map[string]any{
		"data": payload,
	})
}
