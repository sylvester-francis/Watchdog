package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

const genericPasswordResetMessage = "If an account exists for that email, a reset link has been sent."

// PasswordResetHandler exposes the self-serve password reset endpoints.
type PasswordResetHandler struct {
	svc          *services.PasswordResetService
	loginLimiter *middleware.LoginLimiter
	auditSvc     ports.AuditService
}

// NewPasswordResetHandler creates a new handler.
func NewPasswordResetHandler(
	svc *services.PasswordResetService,
	loginLimiter *middleware.LoginLimiter,
	auditSvc ports.AuditService,
) *PasswordResetHandler {
	return &PasswordResetHandler{svc: svc, loginLimiter: loginLimiter, auditSvc: auditSvc}
}

type passwordResetRequestBody struct {
	Email string `json:"email"`
}

type passwordResetCompleteBody struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// RequestReset handles POST /api/v1/auth/password/request.
// Always returns 200 with a generic message regardless of input — anti-enumeration.
func (h *PasswordResetHandler) RequestReset(c echo.Context) error {
	var req passwordResetRequestBody
	if err := c.Bind(&req); err != nil {
		// Even on bind error, return the generic 200 so attackers can't probe.
		return c.JSON(http.StatusOK, map[string]string{"message": genericPasswordResetMessage})
	}

	ip := c.RealIP()

	// Validate-but-don't-leak: if email is missing/malformed, still return 200.
	// Skip the service call (no point hitting the DB) but keep the response identical.
	if req.Email == "" || !isValidEmail(req.Email) {
		return c.JSON(http.StatusOK, map[string]string{"message": genericPasswordResetMessage})
	}

	// Rate-limit by IP + email. Use the existing login limiter — same shape of
	// attack (credential probing) so the same window/budget applies.
	if h.loginLimiter != nil {
		h.loginLimiter.RecordFailure(ip, req.Email)
	}

	if err := h.svc.RequestReset(c.Request().Context(), req.Email, ip); err != nil {
		// Service swallows all internal failures and returns nil — anything
		// here is a programming error. Log + still return 200 so we don't leak.
		slog.Error("password reset request: unexpected error", slog.String("error", err.Error()))
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), nil, domain.AuditPasswordResetRequested, ip, map[string]string{
			"email": req.Email,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": genericPasswordResetMessage})
}

// CompleteReset handles POST /api/v1/auth/password/reset.
// Validates the token, updates the password, marks token used, clears any
// existing session for the user. Returns generic 400 on any failure path
// (anti-enumeration; user just hears "link is no good").
func (h *PasswordResetHandler) CompleteReset(c echo.Context) error {
	var req passwordResetCompleteBody
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" || req.NewPassword == "" {
		return errJSON(c, http.StatusBadRequest, "token and new_password are required")
	}
	if len(req.NewPassword) < 8 {
		return errJSON(c, http.StatusBadRequest, "password must be at least 8 characters")
	}
	if len(req.NewPassword) > 128 {
		return errJSON(c, http.StatusBadRequest, "password must be at most 128 characters")
	}

	ctx := c.Request().Context()
	ip := c.RealIP()

	// Resolve the user before consuming the token so we can audit successfully
	// against a user_id. Resolution failure is also an "invalid token" outcome.
	user, lookupErr := h.svc.ResolveUserByToken(ctx, req.Token)

	if err := h.svc.CompleteReset(ctx, req.Token, req.NewPassword); err != nil {
		if errors.Is(err, services.ErrInvalidResetToken) {
			if h.auditSvc != nil {
				meta := map[string]string{"reason": "invalid_or_expired_token"}
				if user != nil {
					meta["email"] = user.Email
				}
				h.auditSvc.LogEvent(ctx, nil, domain.AuditPasswordResetFailed, ip, meta)
			}
			return errJSON(c, http.StatusBadRequest, "invalid or expired reset link — please request a new one")
		}
		slog.Error("password reset complete: unexpected error", slog.String("error", err.Error()))
		return errJSON(c, http.StatusInternalServerError, "could not reset password")
	}

	// Audit. Existing sessions are invalidated via user.PasswordChangedAt
	// (set in the service) — the auth middleware rejects sessions issued
	// before that timestamp.
	if h.auditSvc != nil && lookupErr == nil && user != nil {
		h.auditSvc.LogEvent(ctx, &user.ID, domain.AuditPasswordResetCompleted, ip, map[string]string{
			"email": user.Email,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Your password has been updated. You can now log in.",
	})
}
