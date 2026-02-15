package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// IncidentWithMonitor combines an incident with its monitor details.
type IncidentWithMonitor struct {
	Incident *domain.Incident
	Monitor  *domain.Monitor
}

// IncidentHandler handles incident-related HTTP requests.
type IncidentHandler struct {
	incidentSvc ports.IncidentService
	monitorRepo ports.MonitorRepository
	templates   *view.Templates
}

// NewIncidentHandler creates a new IncidentHandler.
func NewIncidentHandler(
	incidentSvc ports.IncidentService,
	monitorRepo ports.MonitorRepository,
	templates *view.Templates,
) *IncidentHandler {
	return &IncidentHandler{
		incidentSvc: incidentSvc,
		monitorRepo: monitorRepo,
		templates:   templates,
	}
}

// List renders the incidents list page.
func (h *IncidentHandler) List(c echo.Context) error {
	ctx := c.Request().Context()

	_, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Get filter from query params
	statusFilter := c.QueryParam("status")

	// Get incidents based on filter
	var incidents []*domain.Incident
	var err error

	if statusFilter == "" || statusFilter == "active" {
		incidents, err = h.incidentSvc.GetActiveIncidents(ctx)
	} else {
		// For now, just get active incidents
		// TODO: Add method to get all incidents with filter
		incidents, err = h.incidentSvc.GetActiveIncidents(ctx)
	}

	if err != nil {
		return c.Render(http.StatusInternalServerError, "incidents.html", map[string]interface{}{
			"Title": "Incidents",
			"Error": "Failed to load incidents",
		})
	}

	// Enrich incidents with monitor information
	incidentsWithMonitors := make([]IncidentWithMonitor, 0, len(incidents))
	for _, incident := range incidents {
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		incidentsWithMonitors = append(incidentsWithMonitors, IncidentWithMonitor{
			Incident: incident,
			Monitor:  monitor,
		})
	}

	return c.Render(http.StatusOK, "incidents.html", map[string]interface{}{
		"Title":                 "Incidents",
		"Incidents":             incidents,
		"IncidentsWithMonitors": incidentsWithMonitors,
		"StatusFilter":          statusFilter,
	})
}

// Detail renders the incident detail page.
func (h *IncidentHandler) Detail(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Redirect(http.StatusFound, "/incidents")
	}

	incident, err := h.incidentSvc.GetIncident(ctx, id)
	if err != nil || incident == nil {
		return c.Redirect(http.StatusFound, "/incidents")
	}

	monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)

	return c.Render(http.StatusOK, "incidents.html", map[string]interface{}{
		"Title":      "Incident Detail",
		"Incident":   incident,
		"Monitor":    monitor,
		"ShowDetail": true,
	})
}

// Acknowledge handles incident acknowledgment.
func (h *IncidentHandler) Acknowledge(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	if err := h.incidentSvc.AcknowledgeIncident(ctx, id, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to acknowledge incident"})
	}

	// If HTMX request, return updated row
	if c.Request().Header.Get("HX-Request") == "true" {
		incident, _ := h.incidentSvc.GetIncident(ctx, id)
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		return c.Render(http.StatusOK, "incident_row.html", map[string]interface{}{
			"Incident": incident,
			"Monitor":  monitor,
		})
	}

	return c.Redirect(http.StatusFound, "/incidents")
}

// Resolve handles incident resolution.
func (h *IncidentHandler) Resolve(c echo.Context) error {
	ctx := c.Request().Context()

	_, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid incident ID"})
	}

	if err := h.incidentSvc.ResolveIncident(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to resolve incident"})
	}

	// If HTMX request, return updated row or remove it
	if c.Request().Header.Get("HX-Request") == "true" {
		// When filtering for active incidents, resolved ones should disappear
		if c.QueryParam("filter") == "active" {
			return c.NoContent(http.StatusOK)
		}
		incident, _ := h.incidentSvc.GetIncident(ctx, id)
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		return c.Render(http.StatusOK, "incident_row.html", map[string]interface{}{
			"Incident": incident,
			"Monitor":  monitor,
		})
	}

	return c.Redirect(http.StatusFound, "/incidents")
}
