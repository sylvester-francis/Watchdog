package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/crypto"
)

// AdminHandler handles admin dashboard HTTP requests.
type AdminHandler struct {
	userRepo       ports.UserRepository
	agentRepo      ports.AgentRepository
	monitorRepo    ports.MonitorRepository
	usageEventRepo ports.UsageEventRepository
	hasher         *crypto.PasswordHasher
	templates      *view.Templates
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	userRepo ports.UserRepository,
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	usageEventRepo ports.UsageEventRepository,
	hasher *crypto.PasswordHasher,
	templates *view.Templates,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		agentRepo:      agentRepo,
		monitorRepo:    monitorRepo,
		usageEventRepo: usageEventRepo,
		hasher:         hasher,
		templates:      templates,
	}
}

// Dashboard renders the admin dashboard page.
func (h *AdminHandler) Dashboard(c echo.Context) error {
	ctx := c.Request().Context()

	// Plan breakdown
	planCounts, err := h.userRepo.CountByPlan(ctx)
	if err != nil {
		planCounts = map[domain.Plan]int{}
	}

	// Platform stats
	totalUsers, _ := h.userRepo.Count(ctx)

	// All users with usage
	allUsers, err := h.userRepo.GetAllWithUsage(ctx)
	if err != nil {
		allUsers = nil
	}

	// Near-limit users
	nearLimitUsers, err := h.userRepo.GetUsersNearLimits(ctx)
	if err != nil {
		nearLimitUsers = nil
	}

	// Recent usage events
	recentEvents, err := h.usageEventRepo.GetRecent(ctx, 50)
	if err != nil {
		recentEvents = nil
	}

	// Limit hits in last 24h
	limitHitCount, _ := h.usageEventRepo.CountByEventType(ctx, domain.EventLimitHit, time.Now().Add(-24*time.Hour))

	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"Title":          "Admin",
		"IsAdmin":        true,
		"PlanFree":       planCounts[domain.PlanFree],
		"PlanPro":        planCounts[domain.PlanPro],
		"PlanTeam":       planCounts[domain.PlanTeam],
		"TotalUsers":     totalUsers,
		"AllUsers":       allUsers,
		"NearLimitUsers": nearLimitUsers,
		"RecentEvents":   recentEvents,
		"LimitHitCount":  limitHitCount,
	})
}

// CreateUser handles POST /admin/users to create a new user.
func (h *AdminHandler) CreateUser(c echo.Context) error {
	ctx := c.Request().Context()

	email := c.FormValue("email")
	password := c.FormValue("password")
	plan := domain.Plan(c.FormValue("plan"))
	isAdmin := c.FormValue("is_admin") == "on"

	// Validate
	if email == "" || password == "" {
		return c.String(http.StatusBadRequest, "Email and password are required")
	}
	if len(password) < 8 {
		return c.String(http.StatusBadRequest, "Password must be at least 8 characters")
	}
	if !plan.IsValid() {
		plan = domain.PlanFree
	}

	// Check email uniqueness
	exists, err := h.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to check email")
	}
	if exists {
		return c.String(http.StatusConflict, "Email already exists")
	}

	// Hash password
	hash, err := h.hasher.Hash(password)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to hash password")
	}

	// Create user
	user := domain.NewUser(email, hash)
	user.Plan = plan
	user.IsAdmin = isAdmin

	if err := h.userRepo.Create(ctx, user); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create user")
	}

	// Return the new user row for HTMX swap
	limits := plan.Limits()
	return c.Render(http.StatusOK, "admin_user_row", map[string]interface{}{
		"User": ports.AdminUserView{
			ID:           user.ID,
			Email:        user.Email,
			Plan:         user.Plan,
			IsAdmin:      user.IsAdmin,
			AgentCount:   0,
			MonitorCount: 0,
			AgentMax:     limits.MaxAgents,
			MonitorMax:   limits.MaxMonitors,
			CreatedAt:    user.CreatedAt,
		},
	})
}

// UpdateUser handles POST /admin/users/:id to update a user's plan or admin status.
func (h *AdminHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	user, err := h.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return c.String(http.StatusNotFound, "User not found")
	}

	// Update plan if provided
	if planStr := c.FormValue("plan"); planStr != "" {
		plan := domain.Plan(planStr)
		if plan.IsValid() {
			user.Plan = plan
		}
	}

	// Update admin status if provided
	if c.FormValue("toggle_admin") == "1" {
		user.IsAdmin = !user.IsAdmin
	}

	user.UpdatedAt = time.Now()

	if err := h.userRepo.Update(ctx, user); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to update user: %v", err))
	}

	// Fetch fresh usage data for the row render
	allUsers, err := h.userRepo.GetAllWithUsage(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch user data")
	}

	for _, u := range allUsers {
		if u.ID == id {
			return c.Render(http.StatusOK, "admin_user_row", map[string]interface{}{
				"User": u,
			})
		}
	}

	return c.String(http.StatusNotFound, "User not found after update")
}

// DeleteUser handles DELETE /admin/users/:id.
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	if err := h.userRepo.Delete(ctx, id); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to delete user: %v", err))
	}

	// Return empty string so HTMX removes the row
	return c.String(http.StatusOK, "")
}
