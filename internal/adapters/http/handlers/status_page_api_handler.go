package handlers

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

// statusPageResponse is the JSON DTO for a status page.
type statusPageResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	IsPublic    bool     `json:"is_public"`
	MonitorIDs  []string `json:"monitor_ids"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// availableMonitorResponse is the JSON DTO for a monitor in the available monitors list.
type availableMonitorResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Target string `json:"target"`
	Status string `json:"status"`
}

// StatusPageAPIHandler handles JSON API requests for status page CRUD.
type StatusPageAPIHandler struct {
	statusPageRepo ports.StatusPageRepository
	monitorRepo    ports.MonitorRepository
	agentRepo      ports.AgentRepository
	heartbeatRepo  ports.HeartbeatRepository
	incidentSvc    ports.IncidentService
}

// NewStatusPageAPIHandler creates a new StatusPageAPIHandler.
func NewStatusPageAPIHandler(
	statusPageRepo ports.StatusPageRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	heartbeatRepo ports.HeartbeatRepository,
	incidentSvc ports.IncidentService,
) *StatusPageAPIHandler {
	return &StatusPageAPIHandler{
		statusPageRepo: statusPageRepo,
		monitorRepo:    monitorRepo,
		agentRepo:      agentRepo,
		heartbeatRepo:  heartbeatRepo,
		incidentSvc:    incidentSvc,
	}
}

// toStatusPageResponse converts a domain StatusPage and its monitor IDs into a JSON DTO.
func toStatusPageResponse(page *domain.StatusPage, monitorIDs []uuid.UUID) statusPageResponse {
	ids := make([]string, 0, len(monitorIDs))
	for _, id := range monitorIDs {
		ids = append(ids, id.String())
	}

	return statusPageResponse{
		ID:          page.ID.String(),
		Name:        page.Name,
		Slug:        page.Slug,
		Description: page.Description,
		IsPublic:    page.IsPublic,
		MonitorIDs:  ids,
		CreatedAt:   page.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   page.UpdatedAt.Format(time.RFC3339),
	}
}

// List handles GET /api/v1/status-pages.
func (h *StatusPageAPIHandler) List(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()

	pages, err := h.statusPageRepo.GetByUserID(ctx, userID)
	if err != nil {
		pages = nil
	}

	result := make([]statusPageResponse, 0, len(pages))
	for _, page := range pages {
		monitorIDs, _ := h.statusPageRepo.GetMonitorIDs(ctx, page.ID)
		result = append(result, toStatusPageResponse(page, monitorIDs))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": result,
	})
}

// Create handles POST /api/v1/status-pages.
func (h *StatusPageAPIHandler) Create(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	slug := domain.GenerateSlug(name)

	exists, _ := h.statusPageRepo.SlugExistsForUser(ctx, userID, slug)
	if exists {
		slug = slug + "-" + uuid.New().String()[:8]
	}

	page := domain.NewStatusPage(userID, name, slug)
	page.Description = strings.TrimSpace(req.Description)

	if err := h.statusPageRepo.Create(ctx, page); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create status page"})
	}

	resp := toStatusPageResponse(page, nil)
	return c.JSON(http.StatusCreated, map[string]any{
		"data": resp,
	})
}

// Get handles GET /api/v1/status-pages/:id.
func (h *StatusPageAPIHandler) Get(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ID"})
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	monitorIDs, _ := h.statusPageRepo.GetMonitorIDs(ctx, pageID)
	pageResp := toStatusPageResponse(page, monitorIDs)

	allMonitors, err := h.getUserMonitors(ctx, userID)
	if err != nil {
		allMonitors = nil
	}

	availableMonitors := make([]availableMonitorResponse, 0, len(allMonitors))
	for _, m := range allMonitors {
		availableMonitors = append(availableMonitors, availableMonitorResponse{
			ID:     m.ID.String(),
			Name:   m.Name,
			Type:   string(m.Type),
			Target: m.Target,
			Status: string(m.Status),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":               pageResp,
		"available_monitors": availableMonitors,
	})
}

// Update handles PUT /api/v1/status-pages/:id.
func (h *StatusPageAPIHandler) Update(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ID"})
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		IsPublic    bool     `json:"is_public"`
		MonitorIDs  []string `json:"monitor_ids"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	page.Name = strings.TrimSpace(req.Name)
	page.Description = strings.TrimSpace(req.Description)
	page.IsPublic = req.IsPublic

	if err := h.statusPageRepo.Update(ctx, page); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update status page"})
	}

	// Verify all requested monitors belong to the authenticated user.
	userMonitors, _ := h.getUserMonitors(ctx, userID)
	allowedIDs := make(map[uuid.UUID]struct{}, len(userMonitors))
	for _, m := range userMonitors {
		allowedIDs[m.ID] = struct{}{}
	}

	var monitorIDs []uuid.UUID
	for _, idStr := range req.MonitorIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		if _, ok := allowedIDs[id]; !ok {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "monitor not owned by user",
			})
		}
		monitorIDs = append(monitorIDs, id)
	}
	if err := h.statusPageRepo.SetMonitors(ctx, pageID, monitorIDs); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "status page updated but failed to save monitor assignments"})
	}

	resp := toStatusPageResponse(page, monitorIDs)
	return c.JSON(http.StatusOK, map[string]any{
		"data": resp,
	})
}

// Delete handles DELETE /api/v1/status-pages/:id.
func (h *StatusPageAPIHandler) Delete(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()

	pageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ID"})
	}

	page, err := h.statusPageRepo.GetByID(ctx, pageID)
	if err != nil || page.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	if err := h.statusPageRepo.Delete(ctx, pageID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete status page"})
	}

	return c.NoContent(http.StatusNoContent)
}

// getUserMonitors returns all monitors owned by the user across all their agents.
func (h *StatusPageAPIHandler) getUserMonitors(ctx context.Context, userID uuid.UUID) ([]*domain.Monitor, error) {
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

// --- Public status page JSON API ---

type publicMonitorResponse struct {
	Name            string             `json:"name"`
	Type            string             `json:"type"`
	Status          string             `json:"status"`
	UptimePercent   float64            `json:"uptime_percent"`
	LatencyMs       int                `json:"latency_ms"`
	HasLatency      bool               `json:"has_latency"`
	MetricValue     string             `json:"metric_value,omitempty"`
	MonitoringSince string             `json:"monitoring_since"`
	DataDays        int                `json:"data_days"`
	UptimeHistory   []dayUptimeResponse `json:"uptime_history"`
}

type dayUptimeResponse struct {
	Date    string  `json:"date"`
	Percent float64 `json:"percent"`
}

type publicIncidentResponse struct {
	MonitorName     string  `json:"monitor_name"`
	StartedAt       string  `json:"started_at"`
	ResolvedAt      *string `json:"resolved_at"`
	DurationSeconds int     `json:"duration_seconds"`
	Status          string  `json:"status"`
	IsActive        bool    `json:"is_active"`
}

// PublicView handles GET /api/v1/public/status/:username/:slug (no auth required).
func (h *StatusPageAPIHandler) PublicView(c echo.Context) error {
	ctx := c.Request().Context()
	username := c.Param("username")
	slug := c.Param("slug")

	page, err := h.statusPageRepo.GetByUserAndSlug(ctx, username, slug)
	if err != nil || !page.IsPublic {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	monitorIDs, _ := h.statusPageRepo.GetMonitorIDs(ctx, page.ID)

	monitors := make([]publicMonitorResponse, 0)
	incidents := make([]publicIncidentResponse, 0)
	allUp := true
	now := time.Now().UTC()
	ninetyDaysAgo := now.AddDate(0, 0, -90)
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	var totalUp, totalChecks int

	for _, mid := range monitorIDs {
		m, err := h.monitorRepo.GetByID(ctx, mid)
		if err != nil || m == nil {
			continue
		}
		status := string(m.Status)
		if m.Status != domain.MonitorStatusUp {
			allUp = false
		}

		uptimeHistory := make([]dayUptimeResponse, 0)
		var uptimePercent float64 = -1
		var latencyMs int
		var hasLatency bool
		var metricValue string

		isNonLatency := m.Type == domain.MonitorTypeSystem || m.Type == domain.MonitorTypeDocker || m.Type == domain.MonitorTypeService

		heartbeats, err := h.heartbeatRepo.GetByMonitorIDInRange(ctx, mid, ninetyDaysAgo, now)
		if err == nil && len(heartbeats) > 0 {
			if isNonLatency {
				if m.Type == domain.MonitorTypeSystem {
					for _, hb := range heartbeats {
						if hb.ErrorMessage != nil {
							metricValue = formatMetricReading(m.Target, *hb.ErrorMessage)
							break
						}
					}
				}
			} else {
				for _, hb := range heartbeats {
					if hb.LatencyMs != nil {
						latencyMs = *hb.LatencyMs
						hasLatency = true
						break
					}
				}
			}

			dayMap := make(map[string]struct{ up, total int })
			monitorUp := 0
			monitorTotal := 0
			for _, hb := range heartbeats {
				day := hb.Time.Format("2006-01-02")
				entry := dayMap[day]
				entry.total++
				if hb.Status.IsSuccess() {
					entry.up++
					monitorUp++
				}
				monitorTotal++
				dayMap[day] = entry
			}

			if monitorTotal > 0 {
				uptimePercent = float64(monitorUp) / float64(monitorTotal) * 100
				totalUp += monitorUp
				totalChecks += monitorTotal
			}

			for i := 89; i >= 0; i-- {
				day := now.AddDate(0, 0, -i).Format("2006-01-02")
				if entry, ok := dayMap[day]; ok && entry.total > 0 {
					pct := float64(entry.up) / float64(entry.total) * 100
					uptimeHistory = append(uptimeHistory, dayUptimeResponse{Date: day, Percent: pct})
				} else {
					uptimeHistory = append(uptimeHistory, dayUptimeResponse{Date: day, Percent: -1})
				}
			}
		}

		dataDays := int(math.Min(90, math.Ceil(now.Sub(m.CreatedAt).Hours()/24)))
		if dataDays < 0 {
			dataDays = 0
		}

		monitors = append(monitors, publicMonitorResponse{
			Name:            m.Name,
			Type:            string(m.Type),
			Status:          status,
			UptimePercent:   uptimePercent,
			LatencyMs:       latencyMs,
			HasLatency:      hasLatency,
			MetricValue:     metricValue,
			MonitoringSince: m.CreatedAt.Format(time.RFC3339),
			DataDays:        dataDays,
			UptimeHistory:   uptimeHistory,
		})

		// Fetch incidents for this monitor (last 30 days)
		monitorIncidents, err := h.incidentSvc.GetIncidentsByMonitor(ctx, mid)
		if err == nil {
			for _, inc := range monitorIncidents {
				if inc.StartedAt.Before(thirtyDaysAgo) {
					continue
				}
				var resolvedAt *string
				if inc.ResolvedAt != nil {
					s := inc.ResolvedAt.Format(time.RFC3339)
					resolvedAt = &s
				}
				incidents = append(incidents, publicIncidentResponse{
					MonitorName:     m.Name,
					StartedAt:       inc.StartedAt.Format(time.RFC3339),
					ResolvedAt:      resolvedAt,
					DurationSeconds: int(inc.Duration().Seconds()),
					Status:          string(inc.Status),
					IsActive:        inc.IsActive(),
				})
			}
		}
	}

	// Sort incidents: active first, then by StartedAt DESC
	sort.Slice(incidents, func(i, j int) bool {
		if incidents[i].IsActive != incidents[j].IsActive {
			return incidents[i].IsActive
		}
		return incidents[i].StartedAt > incidents[j].StartedAt
	})

	var aggregateUptime float64
	if totalChecks > 0 {
		aggregateUptime = float64(totalUp) / float64(totalChecks) * 100
	}

	overallStatus := "operational"
	if !allUp && len(monitors) > 0 {
		overallStatus = "degraded"
	}
	if len(monitors) == 0 {
		overallStatus = "no_monitors"
	}

	return c.JSON(http.StatusOK, map[string]any{
		"page": map[string]string{
			"name":        page.Name,
			"description": page.Description,
		},
		"monitors":         monitors,
		"incidents":        incidents,
		"overall_status":   overallStatus,
		"all_up":           allUp,
		"aggregate_uptime": aggregateUptime,
	})
}
