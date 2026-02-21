package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// DashboardData holds the data for the dashboard template.
type DashboardData struct {
	User            *domain.User
	Agents          []*domain.Agent
	ActiveIncidents []*domain.Incident
	Stats           DashboardStats
}

// DashboardStats holds dashboard statistics.
type DashboardStats struct {
	TotalAgents     int
	OnlineAgents    int
	TotalMonitors   int
	MonitorsUp      int
	MonitorsDown    int
	ActiveIncidents int
	UptimePercent   float64
}

// MonitorSparkline holds sparkline data for a single monitor.
type MonitorSparkline struct {
	MonitorID   string
	Name        string
	Status      string
	Type        string
	Target      string
	Latencies   []int
	UptimeUp    int
	UptimeDown  int
	UptimeTotal int
	// CheckResults holds per-check results: 1=success, 0=failure, -1=unknown (oldest first)
	CheckResults []int
	MetricValue  string // system monitors: e.g. "CPU 23.5%"
}

// DashboardHandler handles dashboard-related HTTP requests.
type DashboardHandler struct {
	agentRepo     ports.AgentRepository
	monitorRepo   ports.MonitorRepository
	heartbeatRepo ports.HeartbeatRepository
	incidentSvc   ports.IncidentService
	userRepo      ports.UserRepository
	templates     *view.Templates
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	heartbeatRepo ports.HeartbeatRepository,
	incidentSvc ports.IncidentService,
	userRepo ports.UserRepository,
	templates *view.Templates,
) *DashboardHandler {
	return &DashboardHandler{
		agentRepo:     agentRepo,
		monitorRepo:   monitorRepo,
		heartbeatRepo: heartbeatRepo,
		incidentSvc:   incidentSvc,
		userRepo:      userRepo,
		templates:     templates,
	}
}

// Dashboard renders the main dashboard page.
func (h *DashboardHandler) Dashboard(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Get user's agents
	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "dashboard.html", map[string]interface{}{
			"Title": "Dashboard",
			"Error": "Failed to load agents",
		})
	}

	// Get active incidents
	incidents, err := h.incidentSvc.GetActiveIncidents(ctx)
	if err != nil {
		incidents = []*domain.Incident{}
	}

	// Calculate stats
	stats := DashboardStats{
		TotalAgents:     len(agents),
		ActiveIncidents: len(incidents),
	}

	for _, agent := range agents {
		if agent.IsOnline() {
			stats.OnlineAgents++
		}
	}

	// Collect monitors and compute status breakdown
	var allMonitors []*domain.Monitor
	for _, agent := range agents {
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err == nil {
			allMonitors = append(allMonitors, monitors...)
			stats.TotalMonitors += len(monitors)
			for _, m := range monitors {
				if m.Status == domain.MonitorStatusUp {
					stats.MonitorsUp++
				} else if m.Status == domain.MonitorStatusDown {
					stats.MonitorsDown++
				}
			}
		}
	}

	// Build sparkline data for each monitor and compute overall uptime
	sparklines := make([]MonitorSparkline, 0, len(allMonitors))
	totalHeartbeats := 0
	successHeartbeats := 0
	for _, m := range allMonitors {
		heartbeats, err := h.heartbeatRepo.GetByMonitorID(ctx, m.ID, 20)
		if err != nil {
			heartbeats = nil
		}
		latencies := make([]int, 0, len(heartbeats))
		checkResults := make([]int, 0, len(heartbeats))
		monUp := 0
		monDown := 0
		// Reverse for chronological order (oldest first)
		for i := len(heartbeats) - 1; i >= 0; i-- {
			if heartbeats[i].LatencyMs != nil {
				latencies = append(latencies, *heartbeats[i].LatencyMs)
			}
			totalHeartbeats++
			if heartbeats[i].Status.IsSuccess() {
				successHeartbeats++
				monUp++
				checkResults = append(checkResults, 1)
			} else {
				monDown++
				checkResults = append(checkResults, 0)
			}
		}

		// Extract metric value for system monitors
		var metricValue string
		if m.Type == domain.MonitorTypeSystem && len(heartbeats) > 0 {
			for _, hb := range heartbeats {
				if hb.ErrorMessage != nil {
					metricValue = formatMetricReading(m.Target, *hb.ErrorMessage)
					break
				}
			}
		}

		sparklines = append(sparklines, MonitorSparkline{
			MonitorID:    m.ID.String(),
			Name:         m.Name,
			Status:       string(m.Status),
			Type:         string(m.Type),
			Target:       m.Target,
			Latencies:    latencies,
			UptimeUp:     monUp,
			UptimeDown:   monDown,
			UptimeTotal:  len(heartbeats),
			CheckResults: checkResults,
			MetricValue:  metricValue,
		})
	}

	// Compute uptime percentage
	if totalHeartbeats > 0 {
		stats.UptimePercent = float64(successHeartbeats) / float64(totalHeartbeats) * 100
	}

	// Split sparklines into service (network) and infrastructure (local) groups
	var serviceSparklines, infraSparklines []MonitorSparkline
	for _, s := range sparklines {
		switch domain.MonitorType(s.Type) {
		case domain.MonitorTypeDocker, domain.MonitorTypeDatabase, domain.MonitorTypeSystem:
			infraSparklines = append(infraSparklines, s)
		default:
			serviceSparklines = append(serviceSparklines, s)
		}
	}

	// Enrich incidents with monitor names (cap at 5 for dashboard)
	displayIncidents := incidents
	if len(displayIncidents) > 5 {
		displayIncidents = displayIncidents[:5]
	}
	incidentsWithMonitors := make([]IncidentWithMonitor, 0, len(displayIncidents))
	for _, incident := range displayIncidents {
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		incidentsWithMonitors = append(incidentsWithMonitors, IncidentWithMonitor{
			Incident: incident,
			Monitor:  monitor,
		})
	}

	// Fetch user plan info
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		user = &domain.User{Plan: domain.PlanBeta}
	}

	return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
		"Title":                 "Dashboard",
		"Agents":                agents,
		"ActiveIncidents":       incidents,
		"IncidentsWithMonitors": incidentsWithMonitors,
		"Monitors":              allMonitors,
		"Sparklines":            sparklines,
		"ServiceSparklines":     serviceSparklines,
		"InfraSparklines":       infraSparklines,
		"Stats":                 stats,
		"Plan":                  user.Plan.String(),
		"PlanLimits":            user.Plan.Limits(),
		"IsAdmin":               user.IsAdmin,
	})
}

// AgentResponse is the JSON representation of an agent.
type AgentResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	LastSeenAt *string `json:"lastSeenAt,omitempty"`
}

func toAgentResponse(agent *domain.Agent) AgentResponse {
	ar := AgentResponse{
		ID:     agent.ID.String(),
		Name:   agent.Name,
		Status: string(agent.Status),
	}
	if agent.LastSeenAt != nil {
		t := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		ar.LastSeenAt = &t
	}
	return ar
}

// AgentsJSON returns a JSON list of agents for HTMX requests.
func (h *DashboardHandler) AgentsJSON(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agents"})
	}

	response := make([]AgentResponse, len(agents))
	for i, agent := range agents {
		response[i] = toAgentResponse(agent)
	}

	return c.JSON(http.StatusOK, response)
}

// AgentJSON returns a single agent as JSON for HTMX requests.
func (h *DashboardHandler) AgentJSON(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent ID"})
	}

	agent, err := h.agentRepo.GetByID(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load agent"})
	}
	if agent == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
	}

	return c.JSON(http.StatusOK, toAgentResponse(agent))
}
