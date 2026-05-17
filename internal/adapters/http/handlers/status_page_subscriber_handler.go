package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

const genericSubscribeMessage = "If that email is valid, we've sent a confirmation link."

// StatusPageSubscriberHandler exposes the public subscribe / confirm /
// unsubscribe endpoints. All three are unauthenticated (status pages are public).
type StatusPageSubscriberHandler struct {
	svc          *services.StatusPageSubscriberService
	statusPages  ports.StatusPageRepository
	loginLimiter *middleware.LoginLimiter
}

// NewStatusPageSubscriberHandler constructs the handler.
func NewStatusPageSubscriberHandler(
	svc *services.StatusPageSubscriberService,
	statusPages ports.StatusPageRepository,
	loginLimiter *middleware.LoginLimiter,
) *StatusPageSubscriberHandler {
	return &StatusPageSubscriberHandler{svc: svc, statusPages: statusPages, loginLimiter: loginLimiter}
}

type subscribeRequest struct {
	Email string `json:"email"`
}

// Subscribe handles POST /api/v1/public/status/:username/:slug/subscribe.
// Always returns 200 with a generic message — anti-enumeration.
func (h *StatusPageSubscriberHandler) Subscribe(c echo.Context) error {
	var req subscribeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, map[string]string{"message": genericSubscribeMessage})
	}
	if req.Email == "" || !isValidEmail(req.Email) {
		return c.JSON(http.StatusOK, map[string]string{"message": genericSubscribeMessage})
	}

	page, err := h.statusPages.GetByUserAndSlug(c.Request().Context(), c.Param("username"), c.Param("slug"))
	if err != nil || page == nil {
		// Don't leak whether page exists; behave identically to a real send.
		return c.JSON(http.StatusOK, map[string]string{"message": genericSubscribeMessage})
	}

	if h.loginLimiter != nil {
		h.loginLimiter.RecordFailure(c.RealIP(), req.Email)
	}

	if err := h.svc.Subscribe(c.Request().Context(), page.ID, page.Name, req.Email); err != nil {
		slog.Error("subscribe: unexpected error", slog.String("error", err.Error()))
	}
	return c.JSON(http.StatusOK, map[string]string{"message": genericSubscribeMessage})
}

// Confirm handles GET /api/v1/public/status-subscriber/confirm?token=...
// Redirects the browser to a friendly landing page on either outcome — caller's
// browser shows the result, machine clients can follow the redirect themselves.
func (h *StatusPageSubscriberHandler) Confirm(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.Redirect(http.StatusSeeOther, "/subscribe-confirmed?error=1")
	}
	err := h.svc.Confirm(c.Request().Context(), token)
	if errors.Is(err, services.ErrInvalidSubscriberToken) {
		return c.Redirect(http.StatusSeeOther, "/subscribe-confirmed?error=1")
	}
	if err != nil {
		slog.Error("confirm: unexpected", slog.String("error", err.Error()))
		return c.Redirect(http.StatusSeeOther, "/subscribe-confirmed?error=1")
	}
	return c.Redirect(http.StatusSeeOther, "/subscribe-confirmed")
}

// Unsubscribe handles GET /api/v1/public/status-subscriber/unsubscribe?token=...
// Always redirects to /unsubscribed — idempotent + no information leak.
func (h *StatusPageSubscriberHandler) Unsubscribe(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.Redirect(http.StatusSeeOther, "/unsubscribed?error=1")
	}
	if err := h.svc.Unsubscribe(c.Request().Context(), token); err != nil {
		// Don't leak; log internally.
		slog.Warn("unsubscribe: token rejected or repo error", slog.String("error", err.Error()))
	}
	return c.Redirect(http.StatusSeeOther, "/unsubscribed")
}
