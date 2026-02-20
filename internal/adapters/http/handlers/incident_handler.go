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

// IncidentWithMonitor combines an incident with its monitor details.
type IncidentWithMonitor struct {
	Incident *domain.Incident
	Monitor  *domain.Monitor
}

// IncidentStats holds summary counts for the incidents page.
type IncidentStats struct {
	Open         int
	Acknowledged int
	Resolved     int
	Total        int
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

	statusFilter := c.QueryParam("status")

	var incidents []*domain.Incident
	var err error

	switch statusFilter {
	case "resolved":
		incidents, err = h.incidentSvc.GetResolvedIncidents(ctx)
	case "all":
		incidents, err = h.incidentSvc.GetAllIncidents(ctx)
	default:
		statusFilter = "active"
		incidents, err = h.incidentSvc.GetActiveIncidents(ctx)
	}

	if err != nil {
		return c.Render(http.StatusInternalServerError, "incidents.html", map[string]interface{}{
			"Title": "Incidents",
			"Error": "Failed to load incidents",
		})
	}

	// Compute stats from the current result set
	stats := IncidentStats{Total: len(incidents)}
	for _, inc := range incidents {
		switch inc.Status {
		case domain.IncidentStatusOpen:
			stats.Open++
		case domain.IncidentStatusAcknowledged:
			stats.Acknowledged++
		case domain.IncidentStatusResolved:
			stats.Resolved++
		}
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
		"IncidentStats":         stats,
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
		if incident == nil {
			return c.Redirect(http.StatusFound, "/incidents")
		}
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		return c.Render(http.StatusOK, "incident_row", map[string]interface{}{
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
		if incident == nil {
			return c.Redirect(http.StatusFound, "/incidents")
		}
		monitor, _ := h.monitorRepo.GetByID(ctx, incident.MonitorID)
		return c.Render(http.StatusOK, "incident_row", map[string]interface{}{
			"Incident": incident,
			"Monitor":  monitor,
		})
	}

	return c.Redirect(http.StatusFound, "/incidents")
}
