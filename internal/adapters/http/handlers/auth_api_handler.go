package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

const maxEmailLength = 254

// AuthAPIHandler handles JSON authentication endpoints for the SvelteKit SPA.
type AuthAPIHandler struct {
	authSvc      ports.UserAuthService
	userRepo     ports.UserRepository
	loginLimiter *middleware.LoginLimiter
	auditSvc     ports.AuditService
}

// NewAuthAPIHandler creates a new AuthAPIHandler.
func NewAuthAPIHandler(authSvc ports.UserAuthService, userRepo ports.UserRepository, loginLimiter *middleware.LoginLimiter, auditSvc ports.AuditService) *AuthAPIHandler {
	return &AuthAPIHandler{
		authSvc:      authSvc,
		userRepo:     userRepo,
		loginLimiter: loginLimiter,
		auditSvc:     auditSvc,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Plan     string `json:"plan"`
	IsAdmin  bool   `json:"is_admin"`
}

func isValidEmail(email string) bool {
	if len(email) > maxEmailLength {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:       u.ID.String(),
		Email:    u.Email,
		Username: u.Username,
		Plan:     string(u.Plan),
		IsAdmin:  u.IsAdmin,
	}
}

// Login authenticates a user via JSON and sets a session cookie.
// POST /api/v1/auth/login
func (h *AuthAPIHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}
	if !isValidEmail(req.Email) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	user, err := h.authSvc.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		h.loginLimiter.RecordFailure(c.RealIP(), req.Email)
		if h.auditSvc != nil {
			h.auditSvc.LogEvent(c.Request().Context(), nil, domain.AuditLoginFailed, c.RealIP(), map[string]string{"email": req.Email})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &user.ID, domain.AuditLoginSuccess, c.RealIP(), map[string]string{"email": req.Email})
	}

	if err := middleware.SetUserID(c, user.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	resp := map[string]any{
		"user": toUserResponse(user),
	}
	if user.PasswordChangedAt == nil {
		resp["must_change_password"] = true
	}

	return c.JSON(http.StatusOK, resp)
}

// Register creates a new user account via JSON.
// POST /api/v1/auth/register
func (h *AuthAPIHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}
	if !isValidEmail(req.Email) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}

	if req.Password != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "passwords do not match"})
	}

	user, err := h.authSvc.Register(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already registered"})
		}
		slog.Error("registration failed", "email", req.Email, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "registration failed"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"user": toUserResponse(user),
	})
}

// Logout clears the session cookie.
// POST /api/v1/auth/logout
func (h *AuthAPIHandler) Logout(c echo.Context) error {
	if err := middleware.ClearSession(c); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to clear session"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me returns the currently authenticated user.
// GET /api/v1/auth/me
func (h *AuthAPIHandler) Me(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	resp := map[string]any{
		"user": toUserResponse(user),
	}
	if user.PasswordChangedAt == nil {
		resp["must_change_password"] = true
	}

	return c.JSON(http.StatusOK, resp)
}

// Setup creates the first admin account (only works when no users exist).
// POST /api/v1/auth/setup
func (h *AuthAPIHandler) Setup(c echo.Context) error {
	ctx := c.Request().Context()

	count, err := h.userRepo.Count(ctx)
	if err != nil {
		slog.Error("setup: failed to count users", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	if count > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "setup already completed"})
	}

	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}
	if !isValidEmail(req.Email) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}

	if req.Password != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "passwords do not match"})
	}

	user, err := h.authSvc.Register(ctx, req.Email, req.Password)
	if err != nil {
		slog.Error("setup: registration failed", "email", req.Email, "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to create account"})
	}

	user.IsAdmin = true
	if err := h.userRepo.Update(ctx, user); err != nil {
		slog.Error("setup: failed to promote user to admin", "user_id", user.ID, "error", err)
	}

	slog.Info("setup complete: admin account created", "email", req.Email)

	return c.JSON(http.StatusCreated, map[string]any{
		"user": toUserResponse(user),
	})
}

// NeedsSetup returns whether the system needs initial setup.
// GET /api/v1/auth/needs-setup
func (h *AuthAPIHandler) NeedsSetup(c echo.Context) error {
	count, err := h.userRepo.Count(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	return c.JSON(http.StatusOK, map[string]bool{
		"needs_setup": count == 0,
	})
}
