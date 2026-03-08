package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

// MaintenanceHandler serves CRUD endpoints for maintenance windows.
type MaintenanceHandler struct {
	mwRepo    ports.MaintenanceWindowRepository
	agentRepo ports.AgentRepository
	auditSvc  ports.AuditService
}

// NewMaintenanceHandler creates a new MaintenanceHandler.
func NewMaintenanceHandler(mwRepo ports.MaintenanceWindowRepository, agentRepo ports.AgentRepository, auditSvc ports.AuditService) *MaintenanceHandler {
	return &MaintenanceHandler{mwRepo: mwRepo, agentRepo: agentRepo, auditSvc: auditSvc}
}

type maintenanceWindowResponse struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	AgentName  string `json:"agent_name"`
	Name       string `json:"name"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	Recurrence string `json:"recurrence"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type createMaintenanceWindowRequest struct {
	AgentID    string `json:"agent_id"`
	Name       string `json:"name"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	Recurrence string `json:"recurrence"`
}

type updateMaintenanceWindowRequest struct {
	Name       *string `json:"name"`
	StartsAt   *string `json:"starts_at"`
	EndsAt     *string `json:"ends_at"`
	Recurrence *string `json:"recurrence"`
}

func (h *MaintenanceHandler) toResponse(mw *domain.MaintenanceWindow, agentName string) maintenanceWindowResponse {
	status := "scheduled"
	if mw.IsActive() {
		status = "active"
	} else if mw.IsExpired() {
		status = "expired"
	}
	return maintenanceWindowResponse{
		ID:         mw.ID.String(),
		AgentID:    mw.AgentID.String(),
		AgentName:  agentName,
		Name:       mw.Name,
		StartsAt:   mw.StartsAt.Format(time.RFC3339),
		EndsAt:     mw.EndsAt.Format(time.RFC3339),
		Recurrence: mw.Recurrence,
		Status:     status,
		CreatedAt:  mw.CreatedAt.Format(time.RFC3339),
	}
}

// List returns all maintenance windows for the authenticated user's agents.
// GET /api/v1/maintenance-windows
func (h *MaintenanceHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	windows, err := h.mwRepo.GetByTenant(ctx)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch maintenance windows")
	}

	// Build agent name map scoped to user's agents.
	agents, _ := h.agentRepo.GetByUserID(ctx, userID)
	agentNames := make(map[uuid.UUID]string, len(agents))
	ownedAgents := make(map[uuid.UUID]bool, len(agents))
	for _, a := range agents {
		agentNames[a.ID] = a.Name
		ownedAgents[a.ID] = true
	}

	// Filter windows to only those belonging to the user's agents.
	result := make([]maintenanceWindowResponse, 0, len(windows))
	for _, mw := range windows {
		if !ownedAgents[mw.AgentID] {
			continue
		}
		result = append(result, h.toResponse(mw, agentNames[mw.AgentID]))
	}

	return c.JSON(http.StatusOK, map[string]any{"data": result})
}

// Create creates a new maintenance window.
// POST /api/v1/maintenance-windows
func (h *MaintenanceHandler) Create(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	var req createMaintenanceWindowRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.AgentID == "" || req.StartsAt == "" || req.EndsAt == "" {
		return errJSON(c, http.StatusBadRequest, "name, agent_id, starts_at, and ends_at are required")
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid agent_id")
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid starts_at format, use RFC3339")
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid ends_at format, use RFC3339")
	}

	// Verify agent exists and belongs to the user.
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil {
		return errJSON(c, http.StatusNotFound, "agent not found")
	}
	if agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "agent not found")
	}

	mw := domain.NewMaintenanceWindow(agentID, userID, req.Name, startsAt, endsAt)
	if req.Recurrence != "" {
		mw.Recurrence = req.Recurrence
	}
	if err := mw.Validate(); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	if endsAt.Sub(startsAt) > 30*24*time.Hour {
		return errJSON(c, http.StatusBadRequest, "maintenance window cannot exceed 30 days")
	}

	if err := h.mwRepo.Create(ctx, mw); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to create maintenance window")
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMaintenanceWindowCreated, c.RealIP(), map[string]string{
			"window_id": mw.ID.String(),
			"name":      mw.Name,
			"agent_id":  mw.AgentID.String(),
			"starts_at": mw.StartsAt.Format(time.RFC3339),
			"ends_at":   mw.EndsAt.Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{"data": h.toResponse(mw, agent.Name)})
}

// Update updates an existing maintenance window.
// PUT /api/v1/maintenance-windows/:id
func (h *MaintenanceHandler) Update(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid window ID")
	}

	mw, err := h.mwRepo.GetByID(ctx, id)
	if err != nil || mw == nil {
		return errJSON(c, http.StatusNotFound, "maintenance window not found")
	}

	// Verify ownership via agent.
	agent, err := h.agentRepo.GetByID(ctx, mw.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "maintenance window not found")
	}

	var req updateMaintenanceWindowRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Name != nil {
		mw.Name = *req.Name
	}
	if req.Recurrence != nil {
		mw.Recurrence = *req.Recurrence
	}
	if req.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			return errJSON(c, http.StatusBadRequest, "invalid starts_at format")
		}
		mw.StartsAt = t
	}
	if req.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndsAt)
		if err != nil {
			return errJSON(c, http.StatusBadRequest, "invalid ends_at format")
		}
		mw.EndsAt = t
	}

	if err := mw.Validate(); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	if mw.EndsAt.Sub(mw.StartsAt) > 30*24*time.Hour {
		return errJSON(c, http.StatusBadRequest, "maintenance window cannot exceed 30 days")
	}

	if err := h.mwRepo.Update(ctx, mw); err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to update maintenance window")
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMaintenanceWindowUpdated, c.RealIP(), map[string]string{
			"window_id": mw.ID.String(),
			"name":      mw.Name,
			"agent_id":  mw.AgentID.String(),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"data": h.toResponse(mw, agent.Name)})
}

// Delete removes a maintenance window.
// DELETE /api/v1/maintenance-windows/:id
func (h *MaintenanceHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid window ID")
	}

	// Verify ownership before deleting.
	mw, err := h.mwRepo.GetByID(ctx, id)
	if err != nil || mw == nil {
		return errJSON(c, http.StatusNotFound, "maintenance window not found")
	}
	agent, err := h.agentRepo.GetByID(ctx, mw.AgentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return errJSON(c, http.StatusNotFound, "maintenance window not found")
	}

	if err := h.mwRepo.Delete(ctx, id); err != nil {
		return errJSON(c, http.StatusNotFound, "maintenance window not found")
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditMaintenanceWindowDeleted, c.RealIP(), map[string]string{
			"window_id": id.String(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}
