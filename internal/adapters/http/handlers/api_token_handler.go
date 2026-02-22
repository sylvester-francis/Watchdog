package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// APITokenHandler handles API token management HTTP requests.
type APITokenHandler struct {
	tokenRepo        ports.APITokenRepository
	alertChannelRepo ports.AlertChannelRepository
	userRepo         ports.UserRepository
	templates        *view.Templates
	auditSvc         ports.AuditService
}

// NewAPITokenHandler creates a new APITokenHandler.
func NewAPITokenHandler(tokenRepo ports.APITokenRepository, alertChannelRepo ports.AlertChannelRepository, userRepo ports.UserRepository, templates *view.Templates, auditSvc ports.AuditService) *APITokenHandler {
	return &APITokenHandler{
		tokenRepo:        tokenRepo,
		alertChannelRepo: alertChannelRepo,
		userRepo:         userRepo,
		templates:        templates,
		auditSvc:         auditSvc,
	}
}

// List returns the API tokens page.
func (h *APITokenHandler) List(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	tokens, err := h.tokenRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		tokens = nil
	}

	channels, err := h.alertChannelRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		channels = nil
	}

	return c.Render(http.StatusOK, "settings.html", map[string]interface{}{
		"Title":    "Settings",
		"Tokens":   tokens,
		"Channels": channels,
	})
}

// Create handles POST /settings/tokens to create a new API token.
func (h *APITokenHandler) Create(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Token name is required")
	}

	var expiresAt *time.Time
	switch c.FormValue("expires") {
	case "30d":
		t := time.Now().Add(30 * 24 * time.Hour)
		expiresAt = &t
	case "90d":
		t := time.Now().Add(90 * 24 * time.Hour)
		expiresAt = &t
	}

	scope := domain.TokenScope(c.FormValue("scope"))
	if !scope.IsValid() {
		scope = domain.TokenScopeAdmin
	}

	token, plaintext, err := domain.GenerateAPIToken(userID, name, expiresAt, scope)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to generate token")
	}

	if err := h.tokenRepo.Create(c.Request().Context(), token); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save token")
	}

	// Audit log
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenCreated, c.RealIP(), map[string]string{
			"token_id": token.ID.String(),
			"name":     token.Name,
			"scope":    string(token.Scope),
		})
	}

	return c.Render(http.StatusOK, "token_created", map[string]interface{}{
		"Token":     token,
		"Plaintext": plaintext,
	})
}

// Delete handles DELETE /settings/tokens/:id.
func (h *APITokenHandler) Delete(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid token ID")
	}

	// Verify the token belongs to this user
	tokens, err := h.tokenRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to verify token ownership")
	}

	owned := false
	for _, t := range tokens {
		if t.ID == id {
			owned = true
			break
		}
	}
	if !owned {
		return c.String(http.StatusForbidden, "Token not found")
	}

	if err := h.tokenRepo.Delete(c.Request().Context(), id); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to delete token")
	}

	// Audit log
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenRevoked, c.RealIP(), map[string]string{
			"token_id": id.String(),
		})
	}

	return c.String(http.StatusOK, "")
}

// UpdateUsername handles POST /settings/username.
func (h *APITokenHandler) UpdateUsername(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	username := strings.ToLower(strings.TrimSpace(c.FormValue("username")))
	if !domain.IsValidUsername(username) {
		return c.HTML(http.StatusOK, `<div class="px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-md text-xs text-red-400">Username must be 3-50 characters, lowercase alphanumeric and hyphens only.</div>`)
	}

	ctx := c.Request().Context()

	// Check if username is taken by another user
	existing, err := h.userRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil && existing.ID != userID {
		return c.HTML(http.StatusOK, `<div class="px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-md text-xs text-red-400">Username is already taken.</div>`)
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return c.String(http.StatusInternalServerError, "Failed to load user")
	}

	user.Username = username
	user.UpdatedAt = time.Now()
	if err := h.userRepo.Update(ctx, user); err != nil {
		return c.HTML(http.StatusOK, `<div class="px-3 py-2 bg-red-500/10 border border-red-500/20 rounded-md text-xs text-red-400">Failed to update username.</div>`)
	}

	return c.HTML(http.StatusOK, `<div class="px-3 py-2 bg-emerald-500/10 border border-emerald-500/20 rounded-md text-xs text-emerald-400">Username updated!</div>`)
}
