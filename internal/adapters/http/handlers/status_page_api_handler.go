package handlers

import (
	"context"
	"net/http"
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
}

// NewStatusPageAPIHandler creates a new StatusPageAPIHandler.
func NewStatusPageAPIHandler(
	statusPageRepo ports.StatusPageRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
) *StatusPageAPIHandler {
	return &StatusPageAPIHandler{
		statusPageRepo: statusPageRepo,
		monitorRepo:    monitorRepo,
		agentRepo:      agentRepo,
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

	var monitorIDs []uuid.UUID
	for _, idStr := range req.MonitorIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			monitorIDs = append(monitorIDs, id)
		}
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
