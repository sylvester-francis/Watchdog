package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// AlertChannelHandler handles alert channel CRUD via HTMX.
type AlertChannelHandler struct {
	channelRepo ports.AlertChannelRepository
	templates   *view.Templates
}

// NewAlertChannelHandler creates a new AlertChannelHandler.
func NewAlertChannelHandler(channelRepo ports.AlertChannelRepository, templates *view.Templates) *AlertChannelHandler {
	return &AlertChannelHandler{
		channelRepo: channelRepo,
		templates:   templates,
	}
}

// Create handles POST /settings/alerts.
func (h *AlertChannelHandler) Create(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	channelType := domain.AlertChannelType(c.FormValue("type"))
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Channel name is required")
	}

	config := h.extractConfig(c, channelType)

	channel := domain.NewAlertChannel(userID, channelType, name, config)
	if err := channel.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	if err := h.channelRepo.Create(c.Request().Context(), channel); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save alert channel")
	}

	return c.Render(http.StatusOK, "alert_channel_row", channel)
}

// Delete handles DELETE /settings/alerts/:id.
func (h *AlertChannelHandler) Delete(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid channel ID")
	}

	// Verify ownership
	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get channel")
	}
	if channel == nil || channel.UserID != userID {
		return c.String(http.StatusForbidden, "Channel not found")
	}

	if err := h.channelRepo.Delete(c.Request().Context(), id); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete channel")
	}

	return c.String(http.StatusOK, "")
}

// Toggle handles POST /settings/alerts/:id/toggle.
func (h *AlertChannelHandler) Toggle(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid channel ID")
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get channel")
	}
	if channel == nil || channel.UserID != userID {
		return c.String(http.StatusForbidden, "Channel not found")
	}

	channel.Enabled = !channel.Enabled
	if err := h.channelRepo.Update(c.Request().Context(), channel); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to update channel")
	}

	return c.Render(http.StatusOK, "alert_channel_row", channel)
}

// TestChannel handles POST /settings/alerts/:id/test.
func (h *AlertChannelHandler) TestChannel(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid channel ID")
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get channel")
	}
	if channel == nil || channel.UserID != userID {
		return c.String(http.StatusForbidden, "Channel not found")
	}

	notifier, err := notify.BuildFromChannel(channel)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Invalid config: %v", err))
	}

	// Send a test notification using a fake incident and monitor
	testIncident := &domain.Incident{
		ID:        uuid.New(),
		StartedAt: time.Now(),
		Status:    domain.IncidentStatusOpen,
	}
	testMonitor := &domain.Monitor{
		ID:     uuid.New(),
		Name:   "Test Monitor",
		Type:   domain.MonitorTypeHTTP,
		Target: "https://example.com/health",
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	if err := notifier.NotifyIncidentOpened(ctx, testIncident, testMonitor); err != nil {
		return c.HTML(http.StatusBadGateway, `<div class="px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-md flex items-center space-x-2">
			<i data-lucide="x-circle" class="w-3.5 h-3.5 text-red-400 shrink-0"></i>
			<span class="text-xs text-red-400">Test failed. Check your configuration.</span>
		</div>`)
	}

	return c.HTML(http.StatusOK, `<div class="px-3 py-2 bg-emerald-500/10 border border-emerald-500/20 rounded-md flex items-center space-x-2">
		<i data-lucide="check-circle" class="w-3.5 h-3.5 text-emerald-400 shrink-0"></i>
		<span class="text-xs text-emerald-400">Test notification sent!</span>
	</div>`)
}

// extractConfig pulls type-specific config fields from the form.
func (h *AlertChannelHandler) extractConfig(c echo.Context, channelType domain.AlertChannelType) map[string]string {
	config := make(map[string]string)

	switch channelType {
	case domain.AlertChannelDiscord, domain.AlertChannelSlack:
		config["webhook_url"] = c.FormValue("webhook_url")
	case domain.AlertChannelWebhook:
		config["url"] = c.FormValue("url")
	case domain.AlertChannelEmail:
		config["host"] = c.FormValue("host")
		config["port"] = c.FormValue("port")
		config["username"] = c.FormValue("username")
		config["password"] = c.FormValue("password")
		config["from"] = c.FormValue("from")
		config["to"] = c.FormValue("to")
	case domain.AlertChannelTelegram:
		config["bot_token"] = c.FormValue("bot_token")
		config["chat_id"] = c.FormValue("chat_id")
	case domain.AlertChannelPagerDuty:
		config["routing_key"] = c.FormValue("routing_key")
	}

	return config
}
