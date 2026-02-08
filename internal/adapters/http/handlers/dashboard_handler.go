package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester/watchdog/internal/adapters/http/view"
	"github.com/sylvester/watchdog/internal/core/domain"
	"github.com/sylvester/watchdog/internal/core/ports"
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
	ActiveIncidents int
}

// DashboardHandler handles dashboard-related HTTP requests.
type DashboardHandler struct {
	agentRepo   ports.AgentRepository
	monitorRepo ports.MonitorRepository
	incidentSvc ports.IncidentService
	templates   *view.Templates
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	incidentSvc ports.IncidentService,
	templates *view.Templates,
) *DashboardHandler {
	return &DashboardHandler{
		agentRepo:   agentRepo,
		monitorRepo: monitorRepo,
		incidentSvc: incidentSvc,
		templates:   templates,
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

	// Count monitors for all agents
	for _, agent := range agents {
		monitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err == nil {
			stats.TotalMonitors += len(monitors)
		}
	}

	return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
		"Title":           "Dashboard",
		"Agents":          agents,
		"ActiveIncidents": incidents,
		"Stats":           stats,
	})
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

	// Convert to a JSON-friendly format
	type AgentResponse struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Status     string  `json:"status"`
		LastSeenAt *string `json:"lastSeenAt,omitempty"`
	}

	response := make([]AgentResponse, len(agents))
	for i, agent := range agents {
		ar := AgentResponse{
			ID:     agent.ID.String(),
			Name:   agent.Name,
			Status: string(agent.Status),
		}
		if agent.LastSeenAt != nil {
			t := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
			ar.LastSeenAt = &t
		}
		response[i] = ar
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

	type AgentResponse struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Status     string  `json:"status"`
		LastSeenAt *string `json:"lastSeenAt,omitempty"`
	}

	ar := AgentResponse{
		ID:     agent.ID.String(),
		Name:   agent.Name,
		Status: string(agent.Status),
	}
	if agent.LastSeenAt != nil {
		t := agent.LastSeenAt.Format("2006-01-02T15:04:05Z07:00")
		ar.LastSeenAt = &t
	}

	return c.JSON(http.StatusOK, ar)
}
