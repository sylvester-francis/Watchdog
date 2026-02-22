package handlers

import (
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// MonitorWithHeartbeats holds a monitor with its sparkline and uptime data.
type MonitorWithHeartbeats struct {
	Monitor     *domain.Monitor
	Agent       *domain.Agent
	Latencies   []int
	UptimeUp    int
	UptimeDown  int
	UptimeTotal int
}

// MonitorHandler handles monitor-related HTTP requests.
type MonitorHandler struct {
	monitorSvc    ports.MonitorService
	agentRepo     ports.AgentRepository
	heartbeatRepo ports.HeartbeatRepository
	templates     *view.Templates
	hub           *realtime.Hub
	auditSvc      ports.AuditService
}

// NewMonitorHandler creates a new MonitorHandler.
func NewMonitorHandler(
	monitorSvc ports.MonitorService,
	agentRepo ports.AgentRepository,
	heartbeatRepo ports.HeartbeatRepository,
	templates *view.Templates,
	hub *realtime.Hub,
	auditSvc ports.AuditService,
) *MonitorHandler {
	return &MonitorHandler{
		monitorSvc:    monitorSvc,
		agentRepo:     agentRepo,
		heartbeatRepo: heartbeatRepo,
		templates:     templates,
		hub:           hub,
		auditSvc:      auditSvc,
	}
}

// List renders the monitors list page.
func (h *MonitorHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Get user's agents
	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "monitors.html", map[string]interface{}{
			"Title": "Monitors",
			"Error": "Failed to load agents",
		})
	}

	// Get monitors for each agent
	type AgentWithMonitors struct {
		Agent    *domain.Agent
		Monitors []*domain.Monitor
	}

	agentsWithMonitors := make([]AgentWithMonitors, 0, len(agents))
	var allMonitors []*domain.Monitor
	var monitorsWithHeartbeats []MonitorWithHeartbeats

	for _, agent := range agents {
		monitors, err := h.monitorSvc.GetMonitorsByAgent(ctx, agent.ID)
		if err != nil {
			monitors = []*domain.Monitor{}
		}
		agentsWithMonitors = append(agentsWithMonitors, AgentWithMonitors{
			Agent:    agent,
			Monitors: monitors,
		})
		for _, m := range monitors {
			allMonitors = append(allMonitors, m)

			heartbeats, err := h.heartbeatRepo.GetByMonitorID(ctx, m.ID, 20)
			if err != nil {
				heartbeats = nil
			}
			latencies := make([]int, 0, len(heartbeats))
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
			monitorsWithHeartbeats = append(monitorsWithHeartbeats, MonitorWithHeartbeats{
				Monitor:     m,
				Agent:       agent,
				Latencies:   latencies,
				UptimeUp:    up,
				UptimeDown:  down,
				UptimeTotal: len(heartbeats),
			})
		}
	}

	// Split monitors into service (network) and infrastructure (local) groups
	var serviceMonitors, infraMonitors []MonitorWithHeartbeats
	for _, mwh := range monitorsWithHeartbeats {
		switch domain.MonitorType(mwh.Monitor.Type) {
		case domain.MonitorTypeDocker, domain.MonitorTypeDatabase, domain.MonitorTypeSystem:
			infraMonitors = append(infraMonitors, mwh)
		default:
			serviceMonitors = append(serviceMonitors, mwh)
		}
	}

	return c.Render(http.StatusOK, "monitors.html", map[string]interface{}{
		"Title":                  "Monitors",
		"Agents":                 agents,
		"AgentsWithMonitors":     agentsWithMonitors,
		"Monitors":               allMonitors,
		"MonitorsWithHeartbeats": monitorsWithHeartbeats,
		"ServiceMonitors":        serviceMonitors,
		"InfraMonitors":          infraMonitors,
		"MonitorTypes":           domain.ValidMonitorTypeStrings(),
	})
}

// NewForm renders the new monitor form.
func (h *MonitorHandler) NewForm(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "monitors.html", map[string]interface{}{
			"Title": "New Monitor",
			"Error": "Failed to load agents",
		})
	}

	return c.Render(http.StatusOK, "monitors.html", map[string]interface{}{
		"Title":        "New Monitor",
		"ShowForm":     true,
		"Agents":       agents,
		"MonitorTypes": domain.ValidMonitorTypeStrings(),
	})
}

// Create handles monitor creation.
func (h *MonitorHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Parse form values
	agentIDStr := c.FormValue("agent_id")
	name := c.FormValue("name")
	monitorType := c.FormValue("type")
	target := c.FormValue("target")
	intervalStr := c.FormValue("interval")
	timeoutStr := c.FormValue("timeout")

	// Validation
	if agentIDStr == "" || name == "" || monitorType == "" || target == "" {
		return h.renderError(c, "All fields are required", userID)
	}
	if len(name) > 255 {
		return h.renderError(c, "Monitor name must be 255 characters or less", userID)
	}
	if len(target) > 500 {
		return h.renderError(c, "Target must be 500 characters or less", userID)
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return h.renderError(c, "Invalid agent ID", userID)
	}

	// Verify agent belongs to user
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return h.renderError(c, "Agent not found", userID)
	}

	// Create monitor
	mt := domain.MonitorType(monitorType)
	if !mt.IsValid() {
		return h.renderError(c, "Invalid monitor type", userID)
	}

	// Parse type-specific metadata
	metadata := make(map[string]string)
	switch mt {
	case domain.MonitorTypeHTTP:
		if ec := c.FormValue("expected_content"); ec != "" {
			metadata["expected_content"] = ec
		}
	case domain.MonitorTypeDatabase:
		if dbType := c.FormValue("db_type"); dbType != "" {
			metadata["db_type"] = dbType
		}
		if cs := c.FormValue("connection_string"); cs != "" {
			metadata["connection_string"] = cs
		}
		if pw := c.FormValue("password"); pw != "" {
			metadata["password"] = pw
		}
	}

	monitor, err := h.monitorSvc.CreateMonitor(ctx, userID, agentID, name, mt, target, metadata)
	if err != nil {
		if errors.Is(err, domain.ErrMonitorLimitReached) {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4 mb-4">
						<p class="text-yellow-400 font-medium">Monitor limit reached</p>
						<p class="text-gray-400 text-sm mt-1">You've reached the maximum number of monitors for your account.</p>
					</div>`)
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "monitor limit reached"})
		}
		return h.renderError(c, "Failed to create monitor", userID)
	}

	// Set interval and timeout if provided
	if intervalStr != "" {
		interval, err := strconv.Atoi(intervalStr)
		if err != nil {
			return h.renderError(c, "Invalid interval value", userID)
		}
		if !monitor.SetInterval(interval) {
			return h.renderError(c, "Interval must be between 5 and 3600 seconds", userID)
		}
	}
	if timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return h.renderError(c, "Invalid timeout value", userID)
		}
		if !monitor.SetTimeout(timeout) {
			return h.renderError(c, "Timeout must be between 1 and 60 seconds", userID)
		}
	}

	// Update if interval or timeout were set
	if intervalStr != "" || timeoutStr != "" {
		if err := h.monitorSvc.UpdateMonitor(ctx, monitor); err != nil {
			return h.renderError(c, "Monitor created but failed to apply interval/timeout settings", userID)
		}
	}

	// Audit log
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorCreated, c.RealIP(), map[string]string{
			"monitor_id": monitor.ID.String(),
			"name":       monitor.Name,
			"type":       string(monitor.Type),
			"target":     monitor.Target,
		})
	}

	// Notify agent if connected
	taskMsg := protocol.NewTaskMessageWithMetadata(
		monitor.ID.String(), string(monitor.Type),
		monitor.Target, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Metadata,
	)
	h.hub.SendToAgent(monitor.AgentID, taskMsg)

	// If HTMX request, return the new row
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "monitorCreated")
		return c.Render(http.StatusOK, "monitor_row", map[string]interface{}{
			"Monitor": monitor,
			"Agent":   agent,
		})
	}

	return c.Redirect(http.StatusFound, "/monitors")
}

// Detail renders the monitor detail page.
func (h *MonitorHandler) Detail(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Redirect(http.StatusFound, "/monitors")
	}

	monitor, err := h.monitorSvc.GetMonitor(ctx, id)
	if err != nil || monitor == nil {
		return c.Redirect(http.StatusFound, "/monitors")
	}

	agent, _ := h.agentRepo.GetByID(ctx, monitor.AgentID)

	// Fetch recent heartbeats for this monitor
	heartbeats, err := h.heartbeatRepo.GetByMonitorID(ctx, monitor.ID, 20)
	if err != nil {
		heartbeats = nil
	}

	latencies := make([]int, 0, len(heartbeats))
	up, down := 0, 0
	// Reverse for chronological order (oldest first)
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

	uptimePercent := 0.0
	if len(heartbeats) > 0 {
		uptimePercent = float64(up) / float64(len(heartbeats)) * 100
	}

	// Get latest heartbeat for TLS cert data
	latestHB, _ := h.heartbeatRepo.GetLatestByMonitorID(ctx, monitor.ID)
	var certExpiryDays *int
	var certIssuer *string
	if latestHB != nil {
		certExpiryDays = latestHB.CertExpiryDays
		certIssuer = latestHB.CertIssuer
	}

	return c.Render(http.StatusOK, "monitor_detail.html", map[string]interface{}{
		"Title":          monitor.Name,
		"Monitor":        monitor,
		"Agent":          agent,
		"Heartbeats":     heartbeats,
		"Latencies":      latencies,
		"UptimeUp":       up,
		"UptimeDown":     down,
		"UptimeTotal":    len(heartbeats),
		"UptimePercent":  uptimePercent,
		"CertExpiryDays": certExpiryDays,
		"CertIssuer":     certIssuer,
	})
}

// EditForm renders the edit monitor form.
func (h *MonitorHandler) EditForm(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	monitor, err := h.monitorSvc.GetMonitor(ctx, id)
	if err != nil || monitor == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	return c.Redirect(http.StatusFound, "/monitors/"+id.String())
}

// Update handles monitor update.
func (h *MonitorHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	monitor, err := h.monitorSvc.GetMonitor(ctx, id)
	if err != nil || monitor == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}

	// Update fields from form
	if name := c.FormValue("name"); name != "" {
		monitor.Name = name
	}
	if target := c.FormValue("target"); target != "" {
		monitor.Target = target
	}
	if intervalStr := c.FormValue("interval"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			monitor.SetInterval(interval)
		}
	}
	if timeoutStr := c.FormValue("timeout"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			monitor.SetTimeout(timeout)
		}
	}
	if enabled := c.FormValue("enabled"); enabled == "true" {
		monitor.Enable()
	} else if enabled == "false" {
		monitor.Disable()
	}

	if err := h.monitorSvc.UpdateMonitor(ctx, monitor); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update monitor"})
	}

	// Audit log
	if h.auditSvc != nil {
		userID, _ := middleware.GetUserID(c)
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorUpdated, c.RealIP(), map[string]string{
			"monitor_id": monitor.ID.String(),
			"name":       monitor.Name,
		})
	}

	// Notify agent of the change
	if monitor.Enabled {
		taskMsg := protocol.NewTaskMessageWithMetadata(
			monitor.ID.String(), string(monitor.Type),
			monitor.Target, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Metadata,
		)
		h.hub.SendToAgent(monitor.AgentID, taskMsg)
	} else {
		h.hub.SendToAgent(monitor.AgentID, protocol.NewTaskCancelMessage(monitor.ID.String()))
	}

	// If HTMX request, return updated row
	if c.Request().Header.Get("HX-Request") == "true" {
		agent, _ := h.agentRepo.GetByID(ctx, monitor.AgentID)
		return c.Render(http.StatusOK, "monitor_row", map[string]interface{}{
			"Monitor": monitor,
			"Agent":   agent,
		})
	}

	return c.Redirect(http.StatusFound, "/monitors")
}

// Delete handles monitor deletion.
func (h *MonitorHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid monitor ID"})
	}

	// Fetch monitor before deletion to get AgentID for notification
	monitor, _ := h.monitorSvc.GetMonitor(ctx, id)

	if err := h.monitorSvc.DeleteMonitor(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete monitor"})
	}

	// Audit log
	if h.auditSvc != nil {
		userID, _ := middleware.GetUserID(c)
		meta := map[string]string{"monitor_id": id.String()}
		if monitor != nil {
			meta["name"] = monitor.Name
		}
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMonitorDeleted, c.RealIP(), meta)
	}

	// Notify agent to stop the task
	if monitor != nil {
		h.hub.SendToAgent(monitor.AgentID, protocol.NewTaskCancelMessage(id.String()))
	}

	// If HTMX request, return empty response (row removed)
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/monitors")
}

// renderError is a helper to render error messages.
func (h *MonitorHandler) renderError(c echo.Context, msg string, userID uuid.UUID) error {
	ctx := c.Request().Context()
	agents, _ := h.agentRepo.GetByUserID(ctx, userID)

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusBadRequest, `<div class="text-red-400">`+html.EscapeString(msg)+`</div>`)
	}

	return c.Render(http.StatusBadRequest, "monitors.html", map[string]interface{}{
		"Title":        "Monitors",
		"Error":        msg,
		"Agents":       agents,
		"MonitorTypes": domain.ValidMonitorTypeStrings(),
	})
}
