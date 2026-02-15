package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// AdminHandler handles admin dashboard HTTP requests.
type AdminHandler struct {
	userRepo       ports.UserRepository
	agentRepo      ports.AgentRepository
	monitorRepo    ports.MonitorRepository
	usageEventRepo ports.UsageEventRepository
	templates      *view.Templates
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	userRepo ports.UserRepository,
	agentRepo ports.AgentRepository,
	monitorRepo ports.MonitorRepository,
	usageEventRepo ports.UsageEventRepository,
	templates *view.Templates,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		agentRepo:      agentRepo,
		monitorRepo:    monitorRepo,
		usageEventRepo: usageEventRepo,
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
		"NearLimitUsers": nearLimitUsers,
		"RecentEvents":   recentEvents,
		"LimitHitCount":  limitHitCount,
	})
}
