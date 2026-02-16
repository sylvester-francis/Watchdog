package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authSvc      ports.UserAuthService
	userRepo     ports.UserRepository
	templates    *view.Templates
	loginLimiter *middleware.LoginLimiter
	auditSvc     ports.AuditService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc ports.UserAuthService, userRepo ports.UserRepository, templates *view.Templates, loginLimiter *middleware.LoginLimiter, auditSvc ports.AuditService) *AuthHandler {
	return &AuthHandler{
		authSvc:      authSvc,
		userRepo:     userRepo,
		templates:    templates,
		loginLimiter: loginLimiter,
		auditSvc:     auditSvc,
	}
}

// LoginPage renders the login page.
func (h *AuthHandler) LoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "auth.html", map[string]interface{}{
		"Title":    "Login",
		"IsLogin":  true,
		"Error":    c.QueryParam("error"),
		"Success":  c.QueryParam("success"),
	})
}

// Login handles login form submission.
func (h *AuthHandler) Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {
		return c.Render(http.StatusBadRequest, "auth.html", map[string]interface{}{
			"Title":   "Login",
			"IsLogin": true,
			"Error":   "Email and password are required",
			"Email":   email,
		})
	}

	user, err := h.authSvc.Login(c.Request().Context(), email, password)
	if err != nil {
		h.loginLimiter.RecordFailure(c.RealIP(), email)
		if h.auditSvc != nil {
			h.auditSvc.LogEvent(c.Request().Context(), nil, domain.AuditLoginFailed, c.RealIP(), map[string]string{"email": email})
		}
		errMsg := "Invalid email or password"
		if errors.Is(err, services.ErrInvalidCredentials) {
			errMsg = "Invalid email or password"
		}
		return c.Render(http.StatusUnauthorized, "auth.html", map[string]interface{}{
			"Title":   "Login",
			"IsLogin": true,
			"Error":   errMsg,
			"Email":   email,
		})
	}

	if h.auditSvc != nil {
		h.auditSvc.LogEvent(c.Request().Context(), &user.ID, domain.AuditLoginSuccess, c.RealIP(), map[string]string{"email": email})
	}

	// Set user ID in session
	if err := middleware.SetUserID(c, user.ID); err != nil {
		return c.Render(http.StatusInternalServerError, "auth.html", map[string]interface{}{
			"Title":   "Login",
			"IsLogin": true,
			"Error":   "Failed to create session",
			"Email":   email,
		})
	}

	return c.Redirect(http.StatusFound, "/dashboard")
}

// RegisterPage renders the registration page.
func (h *AuthHandler) RegisterPage(c echo.Context) error {
	return c.Render(http.StatusOK, "auth.html", map[string]interface{}{
		"Title":      "Register",
		"IsRegister": true,
		"Error":      c.QueryParam("error"),
	})
}

// Register handles registration form submission.
func (h *AuthHandler) Register(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	// Validation
	if email == "" || password == "" {
		return c.Render(http.StatusBadRequest, "auth.html", map[string]interface{}{
			"Title":      "Register",
			"IsRegister": true,
			"Error":      "Email and password are required",
			"Email":      email,
		})
	}

	if len(password) < 8 {
		return c.Render(http.StatusBadRequest, "auth.html", map[string]interface{}{
			"Title":      "Register",
			"IsRegister": true,
			"Error":      "Password must be at least 8 characters",
			"Email":      email,
		})
	}

	if password != confirmPassword {
		return c.Render(http.StatusBadRequest, "auth.html", map[string]interface{}{
			"Title":      "Register",
			"IsRegister": true,
			"Error":      "Passwords do not match",
			"Email":      email,
		})
	}

	_, err := h.authSvc.Register(c.Request().Context(), email, password)
	if err != nil {
		errMsg := "Registration failed"
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			errMsg = "Email already registered"
		}
		return c.Render(http.StatusBadRequest, "auth.html", map[string]interface{}{
			"Title":      "Register",
			"IsRegister": true,
			"Error":      errMsg,
			"Email":      email,
		})
	}

	// Redirect to login with success message
	return c.Redirect(http.StatusFound, "/login?success=Account+created+successfully.+Please+login.")
}

// Logout handles logout.
func (h *AuthHandler) Logout(c echo.Context) error {
	if err := middleware.ClearSession(c); err != nil {
		// Log error but still redirect
		_ = err
	}
	return c.Redirect(http.StatusFound, "/login")
}
