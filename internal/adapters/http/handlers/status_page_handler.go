package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// DayUptime represents a single day's uptime percentage for a monitor.
type DayUptime struct {
	Date    string  // YYYY-MM-DD
	Percent float64 // 0-100
}

// StatusPageHandler handles status page HTTP requests.
type StatusPageHandler struct {
	statusPageRepo ports.StatusPageRepository
	monitorRepo    ports.MonitorRepository
	agentRepo      ports.AgentRepository
	heartbeatRepo  ports.HeartbeatRepository
	userRepo       ports.UserRepository
	templates      *view.Templates
}

// NewStatusPageHandler creates a new StatusPageHandler.
func NewStatusPageHandler(
	statusPageRepo ports.StatusPageRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	heartbeatRepo ports.HeartbeatRepository,
	userRepo ports.UserRepository,
	templates *view.Templates,
) *StatusPageHandler {
	return &StatusPageHandler{
		statusPageRepo: statusPageRepo,
		monitorRepo:    monitorRepo,
		agentRepo:      agentRepo,
		heartbeatRepo:  heartbeatRepo,
		userRepo:       userRepo,
		templates:      templates,
	}
}

// List renders the status pages management page.
func (h *StatusPageHandler) List(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	pages, err := h.statusPageRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		pages = nil
	}

	return c.Render(http.StatusOK, "status_pages.html", map[string]interface{}{
		"Title":       "Status Pages",
		"StatusPages": pages,
	})
}

// Create handles POST /status-pages to create a new status page.
func (h *StatusPageHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return c.Render(http.StatusOK, "status_pages.html", map[string]interface{}{
			"Title": "Status Pages",
			"Error": "Name is required",
		})
	}

	slug := domain.GenerateSlug(name)

	// Ensure slug uniqueness within this user's pages
	exists, _ := h.statusPageRepo.SlugExistsForUser(ctx, userID, slug)
	if exists {
		slug = slug + "-" + uuid.New().String()[:8]
	}

	page := domain.NewStatusPage(userID, name, slug)
	page.Description = strings.TrimSpace(c.FormValue("description"))

	if err := h.statusPageRepo.Create(ctx, page); err != nil {
		return c.Render(http.StatusOK, "status_pages.html", map[string]interface{}{
			"Title": "Status Pages",
			"Error": "Failed to create status page",
		})
	}

	return c.Redirect(http.StatusFound, "/status-pages")
}

// Edit renders the status page edit form.
func (h *StatusPageHandler) Edit(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return echo.NewHTTPError(http.StatusNotFound, "status page not found")
	}

	monitorIDs, _ := h.statusPageRepo.GetMonitorIDs(ctx, pageID)
	page.MonitorIDs = monitorIDs

	// Get all user monitors for the select list
	allMonitors, err := h.getUserMonitors(ctx, userID)
	if err != nil {
		allMonitors = nil
	}

	// Build selected map
	selectedMap := make(map[string]bool)
	for _, mid := range monitorIDs {
		selectedMap[mid.String()] = true
	}

	return c.Render(http.StatusOK, "status_page_edit.html", map[string]interface{}{
		"Title":       "Edit Status Page",
		"Page":        page,
		"Monitors":    allMonitors,
		"SelectedMap": selectedMap,
	})
}

// Update handles POST /status-pages/:id to update a status page.
func (h *StatusPageHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid ID")
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return echo.NewHTTPError(http.StatusNotFound, "status page not found")
	}

	page.Name = strings.TrimSpace(c.FormValue("name"))
	page.Description = strings.TrimSpace(c.FormValue("description"))
	page.IsPublic = c.FormValue("is_public") == "on"

	if err := h.statusPageRepo.Update(ctx, page); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update")
	}

	// Update assigned monitors
	monitorIDStrs := c.Request().Form["monitor_ids"]
	var monitorIDs []uuid.UUID
	for _, idStr := range monitorIDStrs {
		if id, err := uuid.Parse(idStr); err == nil {
			monitorIDs = append(monitorIDs, id)
		}
	}
	if err := h.statusPageRepo.SetMonitors(ctx, pageID, monitorIDs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "status page updated but failed to save monitor assignments")
	}

	return c.Redirect(http.StatusFound, "/status-pages")
}

// Delete handles DELETE /status-pages/:id.
func (h *StatusPageHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ID"})
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	if err := h.statusPageRepo.Delete(ctx, pageID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
	}

	return c.NoContent(http.StatusNoContent)
}

// PublicView renders a public status page (no auth required).
func (h *StatusPageHandler) PublicView(c echo.Context) error {
	ctx := c.Request().Context()
	username := c.Param("username")
	slug := c.Param("slug")

	page, err := h.statusPageRepo.GetByUserAndSlug(ctx, username, slug)
	if err != nil {
		return c.Render(http.StatusNotFound, "status_page_not_found.html", nil)
	}

	if !page.IsPublic {
		return c.Render(http.StatusNotFound, "status_page_not_found.html", nil)
	}

	monitorIDs, _ := h.statusPageRepo.GetMonitorIDs(ctx, page.ID)

	type monitorStatus struct {
		Name          string
		Target        string
		Type          string
		Status        string
		UptimeHistory []DayUptime
	}

	var monitors []monitorStatus
	allUp := true
	now := time.Now().UTC()
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	for _, mid := range monitorIDs {
		m, err := h.monitorRepo.GetByID(ctx, mid)
		if err != nil || m == nil {
			continue
		}
		status := string(m.Status)
		if m.Status != domain.MonitorStatusUp {
			allUp = false
		}

		// Compute 90-day uptime history
		var uptimeHistory []DayUptime
		heartbeats, err := h.heartbeatRepo.GetByMonitorIDInRange(ctx, mid, ninetyDaysAgo, now)
		if err == nil && len(heartbeats) > 0 {
			// Group by day
			dayMap := make(map[string]struct{ up, total int })
			for _, hb := range heartbeats {
				day := hb.Time.Format("2006-01-02")
				entry := dayMap[day]
				entry.total++
				if hb.Status.IsSuccess() {
					entry.up++
				}
				dayMap[day] = entry
			}
			// Build 90-day array
			for i := 89; i >= 0; i-- {
				day := now.AddDate(0, 0, -i).Format("2006-01-02")
				if entry, ok := dayMap[day]; ok && entry.total > 0 {
					pct := float64(entry.up) / float64(entry.total) * 100
					uptimeHistory = append(uptimeHistory, DayUptime{Date: day, Percent: pct})
				} else {
					uptimeHistory = append(uptimeHistory, DayUptime{Date: day, Percent: -1}) // -1 = no data
				}
			}
		}

		monitors = append(monitors, monitorStatus{
			Name:          m.Name,
			Target:        m.Target,
			Type:          string(m.Type),
			Status:        status,
			UptimeHistory: uptimeHistory,
		})
	}

	overallStatus := "All Systems Operational"
	if !allUp && len(monitors) > 0 {
		overallStatus = "Some Systems Experiencing Issues"
	}
	if len(monitors) == 0 {
		overallStatus = "No Monitors Configured"
	}

	return c.Render(http.StatusOK, "status_page_public.html", map[string]interface{}{
		"Page":          page,
		"Monitors":      monitors,
		"OverallStatus": overallStatus,
		"AllUp":         allUp,
	})
}

// getUserMonitors returns all monitors owned by the user.
func (h *StatusPageHandler) getUserMonitors(ctx context.Context, userID uuid.UUID) ([]*domain.Monitor, error) {
	agents, err := h.agentRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var monitors []*domain.Monitor
	for _, agent := range agents {
		agentMonitors, err := h.monitorRepo.GetByAgentID(ctx, agent.ID)
		if err != nil {
			continue
		}
		monitors = append(monitors, agentMonitors...)
	}
	return monitors, nil
}
