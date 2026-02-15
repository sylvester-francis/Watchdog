package handlers

import (
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
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
}

// NewMonitorHandler creates a new MonitorHandler.
func NewMonitorHandler(
	monitorSvc ports.MonitorService,
	agentRepo ports.AgentRepository,
	heartbeatRepo ports.HeartbeatRepository,
	templates *view.Templates,
) *MonitorHandler {
	return &MonitorHandler{
		monitorSvc:    monitorSvc,
		agentRepo:     agentRepo,
		heartbeatRepo: heartbeatRepo,
		templates:     templates,
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

	return c.Render(http.StatusOK, "monitors.html", map[string]interface{}{
		"Title":                  "Monitors",
		"Agents":                 agents,
		"AgentsWithMonitors":     agentsWithMonitors,
		"Monitors":               allMonitors,
		"MonitorsWithHeartbeats": monitorsWithHeartbeats,
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

	// If HTMX request, return just the form partial
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.Render(http.StatusOK, "monitor_form.html", map[string]interface{}{
			"Agents":       agents,
			"MonitorTypes": domain.ValidMonitorTypeStrings(),
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

	monitor, err := h.monitorSvc.CreateMonitor(ctx, userID, agentID, name, mt, target)
	if err != nil {
		if errors.Is(err, domain.ErrMonitorLimitReached) {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4 mb-4">
						<p class="text-yellow-400 font-medium">Monitor limit reached</p>
						<p class="text-gray-400 text-sm mt-1">Upgrade your plan to create more monitors.</p>
					</div>`)
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "monitor limit reached for current plan"})
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
		_ = h.monitorSvc.UpdateMonitor(ctx, monitor)
	}

	// If HTMX request, return the new row
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Trigger", "monitorCreated")
		return c.Render(http.StatusOK, "monitor_row.html", map[string]interface{}{
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

	return c.Render(http.StatusOK, "monitor_detail.html", map[string]interface{}{
		"Title":         monitor.Name,
		"Monitor":       monitor,
		"Agent":         agent,
		"Heartbeats":    heartbeats,
		"Latencies":     latencies,
		"UptimeUp":      up,
		"UptimeDown":    down,
		"UptimeTotal":   len(heartbeats),
		"UptimePercent": uptimePercent,
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

	return c.Render(http.StatusOK, "monitor_edit.html", map[string]interface{}{
		"Monitor":      monitor,
		"MonitorTypes": domain.ValidMonitorTypeStrings(),
	})
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

	// If HTMX request, return updated row
	if c.Request().Header.Get("HX-Request") == "true" {
		agent, _ := h.agentRepo.GetByID(ctx, monitor.AgentID)
		return c.Render(http.StatusOK, "monitor_row.html", map[string]interface{}{
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

	if err := h.monitorSvc.DeleteMonitor(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete monitor"})
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
