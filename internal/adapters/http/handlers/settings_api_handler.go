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
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/crypto"
)

// SettingsAPIHandler serves the JSON API for settings (tokens, channels, profile).
type SettingsAPIHandler struct {
	tokenRepo   ports.APITokenRepository
	channelRepo ports.AlertChannelRepository
	userRepo    ports.UserRepository
	auditSvc    ports.AuditService
	hasher      *crypto.PasswordHasher
}

// NewSettingsAPIHandler creates a new SettingsAPIHandler.
func NewSettingsAPIHandler(
	tokenRepo ports.APITokenRepository,
	channelRepo ports.AlertChannelRepository,
	userRepo ports.UserRepository,
	auditSvc ports.AuditService,
	hasher *crypto.PasswordHasher,
) *SettingsAPIHandler {
	return &SettingsAPIHandler{
		tokenRepo:   tokenRepo,
		channelRepo: channelRepo,
		userRepo:    userRepo,
		auditSvc:    auditSvc,
		hasher:      hasher,
	}
}

// --- Response DTOs ---

type tokenResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Prefix     string  `json:"prefix"`
	Scope      string  `json:"scope"`
	LastUsedAt *string `json:"last_used_at"`
	LastUsedIP *string `json:"last_used_ip"`
	ExpiresAt  *string `json:"expires_at"`
	CreatedAt  string  `json:"created_at"`
}

type channelResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Config    map[string]string `json:"config"`
	Enabled   bool              `json:"enabled"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

func toTokenResponse(t *domain.APIToken) tokenResponse {
	resp := tokenResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Prefix:    t.Prefix,
		Scope:     string(t.Scope),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
	if t.LastUsedAt != nil {
		s := t.LastUsedAt.Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	if t.LastUsedIP != nil {
		resp.LastUsedIP = t.LastUsedIP
	}
	if t.ExpiresAt != nil {
		s := t.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp
}

func toChannelResponse(ch *domain.AlertChannel) channelResponse {
	config := make(map[string]string, len(ch.Config))
	for k, v := range ch.Config {
		if k == "password" {
			config[k] = "\u2022\u2022\u2022\u2022\u2022\u2022"
		} else {
			config[k] = v
		}
	}
	return channelResponse{
		ID:        ch.ID.String(),
		Type:      string(ch.Type),
		Name:      ch.Name,
		Config:    config,
		Enabled:   ch.Enabled,
		CreatedAt: ch.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ch.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Token Endpoints ---

// ListTokens returns all API tokens for the authenticated user.
// GET /api/v1/tokens
func (h *SettingsAPIHandler) ListTokens(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	tokens, err := h.tokenRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch tokens"})
	}

	result := make([]tokenResponse, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, toTokenResponse(t))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": result,
	})
}

// CreateToken creates a new API token.
// POST /api/v1/tokens
func (h *SettingsAPIHandler) CreateToken(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		Name    string `json:"name"`
		Scope   string `json:"scope"`
		Expires string `json:"expires"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	scope := domain.TokenScope(req.Scope)
	if !scope.IsValid() {
		scope = domain.TokenScopeAdmin
	}

	var expiresAt *time.Time
	switch req.Expires {
	case "30d":
		t := time.Now().Add(30 * 24 * time.Hour)
		expiresAt = &t
	case "90d":
		t := time.Now().Add(90 * 24 * time.Hour)
		expiresAt = &t
	case "":
		// no expiry
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expires value; use \"30d\", \"90d\", or \"\""})
	}

	token, plaintext, err := domain.GenerateAPIToken(userID, name, expiresAt, scope)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	if err := h.tokenRepo.Create(c.Request().Context(), token); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenCreated, c.RealIP(), map[string]string{
			"token_id": token.ID.String(),
			"name":     token.Name,
			"scope":    string(token.Scope),
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data":      toTokenResponse(token),
		"plaintext": plaintext,
	})
}

// DeleteToken deletes an API token by ID.
// DELETE /api/v1/tokens/:id
func (h *SettingsAPIHandler) DeleteToken(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token ID"})
	}

	tokens, err := h.tokenRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify token ownership"})
	}

	owned := false
	for _, t := range tokens {
		if t.ID == id {
			owned = true
			break
		}
	}
	if !owned {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "token not found"})
	}

	if err := h.tokenRepo.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete token"})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenRevoked, c.RealIP(), map[string]string{
			"token_id": id.String(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// RegenerateToken deletes an existing token and creates a replacement with the same settings.
// POST /api/v1/tokens/:id/regenerate
func (h *SettingsAPIHandler) RegenerateToken(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid token ID"})
	}

	tokens, err := h.tokenRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to verify token ownership"})
	}

	var old *domain.APIToken
	for _, t := range tokens {
		if t.ID == id {
			old = t
			break
		}
	}
	if old == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "token not found"})
	}

	if err := h.tokenRepo.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete old token"})
	}

	newToken, plaintext, err := domain.GenerateAPIToken(userID, old.Name, old.ExpiresAt, old.Scope)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	if err := h.tokenRepo.Create(c.Request().Context(), newToken); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenRevoked, c.RealIP(), map[string]string{
			"token_id": id.String(),
		})
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditAPITokenCreated, c.RealIP(), map[string]string{
			"token_id": newToken.ID.String(),
			"name":     newToken.Name,
			"scope":    string(newToken.Scope),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data":      toTokenResponse(newToken),
		"plaintext": plaintext,
	})
}

// --- Alert Channel Endpoints ---

// ListChannels returns all alert channels for the authenticated user.
// GET /api/v1/alert-channels
func (h *SettingsAPIHandler) ListChannels(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	channels, err := h.channelRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch channels"})
	}

	result := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		result = append(result, toChannelResponse(ch))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": result,
	})
}

// CreateChannel creates a new alert channel.
// POST /api/v1/alert-channels
func (h *SettingsAPIHandler) CreateChannel(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		Type   string            `json:"type"`
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	channelType := domain.AlertChannelType(req.Type)
	if !domain.ValidAlertChannelTypes[channelType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel type"})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if req.Config == nil {
		req.Config = make(map[string]string)
	}

	channel := domain.NewAlertChannel(userID, channelType, name, req.Config)
	if err := channel.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.channelRepo.Create(c.Request().Context(), channel); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save alert channel"})
	}

	// H-011: audit channel creation.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditChannelCreated, c.RealIP(), map[string]string{
			"channel_id": channel.ID.String(), "type": string(channel.Type), "name": channel.Name,
		})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": toChannelResponse(channel),
	})
}

// DeleteChannel deletes an alert channel by ID.
// DELETE /api/v1/alert-channels/:id
func (h *SettingsAPIHandler) DeleteChannel(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel ID"})
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get channel"})
	}
	if channel == nil || channel.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "channel not found"})
	}

	if err := h.channelRepo.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete channel"})
	}

	// H-011: audit channel deletion.
	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &userID, domain.AuditChannelDeleted, c.RealIP(), map[string]string{
			"channel_id": id.String(), "type": string(channel.Type), "name": channel.Name,
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// ToggleChannel flips the enabled state of an alert channel.
// POST /api/v1/alert-channels/:id/toggle
func (h *SettingsAPIHandler) ToggleChannel(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel ID"})
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get channel"})
	}
	if channel == nil || channel.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "channel not found"})
	}

	channel.Enabled = !channel.Enabled
	if err := h.channelRepo.Update(c.Request().Context(), channel); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update channel"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": toChannelResponse(channel),
	})
}

// TestChannel sends a test notification through the specified alert channel.
// POST /api/v1/alert-channels/:id/test
func (h *SettingsAPIHandler) TestChannel(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel ID"})
	}

	channel, err := h.channelRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get channel"})
	}
	if channel == nil || channel.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "channel not found"})
	}

	notifier, err := notify.BuildFromChannel(channel)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid channel configuration"})
	}

	testMonitor := &domain.Monitor{
		ID:     uuid.New(),
		Name:   "Test Monitor",
		Type:   domain.MonitorTypeHTTP,
		Target: "https://example.com/health",
	}
	testIncident := &domain.Incident{
		ID:        uuid.New(),
		StartedAt: time.Now(),
		Status:    domain.IncidentStatusOpen,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	if err := notifier.NotifyIncidentOpened(ctx, testIncident, testMonitor); err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "Test failed. Check your configuration."})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// --- Profile Endpoints ---

// UpdateProfile updates the authenticated user's profile.
// PATCH /api/v1/users/me
func (h *SettingsAPIHandler) UpdateProfile(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	if !domain.IsValidUsername(username) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username must be 3-50 characters, lowercase alphanumeric and hyphens only"})
	}

	ctx := c.Request().Context()

	existing, err := h.userRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil && existing.ID != userID {
		return c.JSON(http.StatusConflict, map[string]string{"error": "username is already taken"})
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load user"})
	}

	user.Username = username
	user.UpdatedAt = time.Now()
	if err := h.userRepo.Update(ctx, user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update username"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]string{
			"username": user.Username,
		},
	})
}

// ChangePassword allows a user to change their own password.
// POST /api/v1/users/me/password
func (h *SettingsAPIHandler) ChangePassword(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "current password and new password are required"})
	}
	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new password must be at least 8 characters"})
	}
	if req.NewPassword != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new passwords do not match"})
	}
	if req.NewPassword == req.CurrentPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new password must be different from current password"})
	}

	ctx := c.Request().Context()

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to load user"})
	}

	match, err := h.hasher.Verify(req.CurrentPassword, user.PasswordHash)
	if err != nil || !match {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "current password is incorrect"})
	}

	hash, err := h.hasher.Hash(req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	now := time.Now()
	user.PasswordHash = hash
	user.PasswordChangedAt = &now
	user.UpdatedAt = now

	if err := h.userRepo.Update(ctx, user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(ctx, &userID, domain.AuditPasswordChanged, c.RealIP(), nil)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
