package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// APITokenHandler handles API token management HTTP requests.
type APITokenHandler struct {
	tokenRepo ports.APITokenRepository
	templates *view.Templates
}

// NewAPITokenHandler creates a new APITokenHandler.
func NewAPITokenHandler(tokenRepo ports.APITokenRepository, templates *view.Templates) *APITokenHandler {
	return &APITokenHandler{
		tokenRepo: tokenRepo,
		templates: templates,
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

	return c.Render(http.StatusOK, "settings.html", map[string]interface{}{
		"Title":  "Settings",
		"Tokens": tokens,
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

	token, plaintext, err := domain.GenerateAPIToken(userID, name, expiresAt)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to generate token")
	}

	if err := h.tokenRepo.Create(c.Request().Context(), token); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save token")
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

	return c.String(http.StatusOK, "")
}
