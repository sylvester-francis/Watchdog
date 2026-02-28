package http

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/core/registry"
	"github.com/sylvester-francis/watchdog/internal/crypto"
)

// Dependencies holds all the dependencies required by the router.
type Dependencies struct {
	UserAuthService  ports.UserAuthService
	AgentAuthService ports.AgentAuthService
	MonitorService   ports.MonitorService
	IncidentService  ports.IncidentService
	UserRepo         ports.UserRepository
	AgentRepo        ports.AgentRepository
	MonitorRepo      ports.MonitorRepository
	HeartbeatRepo    ports.HeartbeatRepository
	UsageEventRepo   ports.UsageEventRepository
	WaitlistRepo     ports.WaitlistRepository
	APITokenRepo     ports.APITokenRepository
	StatusPageRepo   ports.StatusPageRepository
	AlertChannelRepo ports.AlertChannelRepository
	CertDetailsRepo  ports.CertDetailsRepository
	Hub              *realtime.Hub
	Hasher           *crypto.PasswordHasher
	AuditService     ports.AuditService
	AuditLogRepo     ports.AuditLogRepository
	DB               *repository.DB
	Config           *config.Config
	StartTime        time.Time
	Logger           *slog.Logger
	SessionSecret    string
	SecureCookies    bool
	AllowedOrigins   []string
	Registry         *registry.Registry   // optional: module registry for health checks
}

// Router handles HTTP routing and handler registration.
type Router struct {
	echo *echo.Echo
	deps Dependencies

	// Handlers
	sseHandler           *handlers.SSEHandler
	wsHandler            *handlers.WSHandler
	apiHandler           *handlers.APIHandler
	apiV1Handler         *handlers.APIV1Handler
	authAPIHandler       *handlers.AuthAPIHandler
	settingsAPIHandler   *handlers.SettingsAPIHandler
	statusPageAPIHandler *handlers.StatusPageAPIHandler
	systemAPIHandler     *handlers.SystemAPIHandler

	// Rate limiters (kept for graceful shutdown)
	authRateLimiter    *middleware.RateLimiter
	generalRateLimiter *middleware.RateLimiter
	loginLimiter       *middleware.LoginLimiter
	registerLimiter    *middleware.RegisterLimiter

	// H-017: concurrent session tracker
	sessionTracker *middleware.SessionTracker
}

// NewRouter creates a new Router instance.
func NewRouter(e *echo.Echo, deps Dependencies) (*Router, error) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	loginLimiter := middleware.NewLoginLimiter()
	registerLimiter := middleware.NewRegisterLimiter()
	sessionTracker := middleware.NewSessionTracker() // H-017

	r := &Router{
		echo:            e,
		deps:            deps,
		loginLimiter:    loginLimiter,
		registerLimiter: registerLimiter,
		sessionTracker:  sessionTracker,
	}

	// Initialize handlers
	r.sseHandler = handlers.NewSSEHandler(deps.Hub, deps.AgentRepo, deps.MonitorRepo, deps.IncidentService)
	r.wsHandler = handlers.NewWSHandler(deps.AgentAuthService, deps.MonitorService, deps.AgentRepo, deps.CertDetailsRepo, deps.Hub, logger, deps.AllowedOrigins)
	r.apiHandler = handlers.NewAPIHandler(deps.HeartbeatRepo, deps.MonitorRepo, deps.AgentRepo, deps.IncidentService)
	r.apiV1Handler = handlers.NewAPIV1Handler(deps.AgentRepo, deps.MonitorRepo, deps.HeartbeatRepo, deps.CertDetailsRepo, deps.IncidentService, deps.MonitorService, deps.AgentAuthService, deps.Hub, deps.AuditService)
	r.authAPIHandler = handlers.NewAuthAPIHandler(deps.UserAuthService, deps.UserRepo, loginLimiter, registerLimiter, deps.AuditService, sessionTracker)
	r.settingsAPIHandler = handlers.NewSettingsAPIHandler(deps.APITokenRepo, deps.AlertChannelRepo, deps.UserRepo, deps.AuditService, deps.Hasher)
	r.statusPageAPIHandler = handlers.NewStatusPageAPIHandler(deps.StatusPageRepo, deps.MonitorRepo, deps.AgentRepo, deps.HeartbeatRepo, deps.IncidentService)
	r.systemAPIHandler = handlers.NewSystemAPIHandler(deps.DB, deps.Hub, deps.Config, deps.AuditLogRepo, deps.UserRepo, deps.AgentRepo, deps.MonitorRepo, deps.AuditService, deps.Hasher, deps.StartTime)

	return r, nil
}

// RegisterRoutes registers all HTTP routes.
func (r *Router) RegisterRoutes() {
	e := r.echo

	// Session middleware
	store := sessions.NewCookieStore([]byte(r.deps.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // H-002: 24 hours (was 7 days)
		HttpOnly: true,
		Secure:   r.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	e.Use(middleware.SessionMiddleware(store))

	// CSRF protection for HTML form submissions
	e.Use(middleware.CSRFMiddleware(r.deps.SecureCookies))

	// Rate limiters (H-019: documented rationale for each config).
	//
	// Auth rate limiter: stricter limits for login/register/token-create endpoints.
	//   - Rate=5/s, Burst=10: login is a high-value target for credential stuffing.
	//     5 req/s per IP is generous for legitimate use but limits brute-force.
	r.authRateLimiter = middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            5,
		Burst:           10,
		CleanupInterval: 5 * time.Minute,
	})
	// General API rate limiter: see DefaultRateLimiterConfig() for rationale.
	// SSE and WebSocket upgrade paths are excluded in the middleware itself.
	r.generalRateLimiter = middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())
	e.Use(r.generalRateLimiter.Middleware())

	// H-001: Reject request bodies larger than 1 MB to prevent DoS via oversized payloads.
	e.Use(echomw.BodyLimit("1M"))

	// Static files (legacy — still serves openapi.json, favicon, etc.)
	e.Static("/static", "web/static")

	// --- Go-handled routes (API, SSE, WS, health, install) ---

	authRL := r.authRateLimiter.Middleware()

	// Health check (public)
	e.GET("/health", r.healthCheck)

	// Agent install script (public)
	e.GET("/install", r.installScript)

	// WebSocket endpoint for agents (public — authenticated via API key in handshake)
	e.GET("/ws/agent", r.wsHandler.HandleConnection)

	// Tenant scope middleware — resolve tenant ID into request context
	tenantMW := r.tenantMiddleware()

	// SSE for real-time updates (auth required)
	sseGroup := e.Group("")
	sseGroup.Use(middleware.NoCacheHeaders)
	sseGroup.Use(middleware.AuthRequired)
	sseGroup.Use(tenantMW)
	sseGroup.Use(middleware.UserContext(r.deps.UserRepo))
	sseGroup.GET("/sse/events", r.sseHandler.Events)

	// Public API v1 auth routes (no auth required)
	v1Public := e.Group("/api/v1")
	v1Public.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:   r.deps.AllowedOrigins,
		AllowMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:   []string{echo.HeaderAuthorization, echo.HeaderContentType},
		AllowCredentials: true,
		MaxAge:         86400,
	}))
	// H-024: reject non-JSON Content-Type on mutating endpoints.
	v1Public.Use(middleware.RequireJSONContentType())
	loginLLJSON := r.loginLimiter.MiddlewareJSON()
	v1Public.POST("/auth/login", r.authAPIHandler.Login, authRL, loginLLJSON)
	regRL := r.registerLimiter.Middleware()
	v1Public.POST("/auth/register", r.authAPIHandler.Register, authRL, regRL)
	v1Public.POST("/auth/setup", r.authAPIHandler.Setup, authRL)
	v1Public.POST("/auth/logout", r.authAPIHandler.Logout)
	v1Public.GET("/auth/needs-setup", r.authAPIHandler.NeedsSetup)

	// Public status page API (no auth required)
	v1Public.GET("/public/status/:username/:slug", r.statusPageAPIHandler.PublicView)

	// API v1 (hybrid auth: Bearer token OR session cookie)
	v1 := e.Group("/api/v1")
	v1.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:   r.deps.AllowedOrigins,
		AllowMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:   []string{echo.HeaderAuthorization, echo.HeaderContentType},
		AllowCredentials: true,
		MaxAge:         86400,
	}))
	// H-024: reject non-JSON Content-Type on mutating endpoints.
	v1.Use(middleware.RequireJSONContentType())
	v1.Use(middleware.HybridAuth(r.deps.APITokenRepo))
	v1.Use(middleware.RequireWriteScope())
	// H-003: reject sessions issued before last password change.
	v1.Use(middleware.SessionPasswordCheck(r.deps.UserRepo))
	v1.Use(tenantMW)

	// Auth API (authenticated)
	v1.GET("/auth/me", r.authAPIHandler.Me)

	// Monitors
	v1.GET("/monitors", r.apiV1Handler.ListMonitors)
	v1.GET("/monitors/:id", r.apiV1Handler.GetMonitor)
	v1.GET("/monitors/:id/heartbeats", r.apiHandler.MonitorHeartbeats)
	v1.GET("/monitors/:id/latency", r.apiHandler.MonitorLatencyHistory)
	v1.POST("/monitors", r.apiV1Handler.CreateMonitor)
	v1.PUT("/monitors/:id", r.apiV1Handler.UpdateMonitor)
	v1.DELETE("/monitors/:id", r.apiV1Handler.DeleteMonitor)
	v1.GET("/monitors/:id/certificate", r.apiV1Handler.GetMonitorCertificate)
	v1.GET("/monitors/:id/sla", r.apiV1Handler.GetMonitorSLA)
	v1.GET("/certificates/expiring", r.apiV1Handler.GetExpiringCertificates)

	// Agents
	v1.GET("/agents", r.apiV1Handler.ListAgents)
	v1.POST("/agents", r.apiV1Handler.CreateAgent)
	v1.DELETE("/agents/:id", r.apiV1Handler.DeleteAgent)

	// Incidents
	v1.GET("/incidents", r.apiV1Handler.ListIncidents)
	v1.POST("/incidents/:id/acknowledge", r.apiV1Handler.AcknowledgeIncident)
	v1.POST("/incidents/:id/resolve", r.apiV1Handler.ResolveIncident)

	// Dashboard
	v1.GET("/dashboard/stats", r.apiV1Handler.DashboardStats)
	v1.GET("/monitors/summary", r.apiHandler.MonitorsSummary)

	// Settings: tokens (tighter rate limit on mutating endpoints — H-008)
	v1.GET("/tokens", r.settingsAPIHandler.ListTokens)
	v1.POST("/tokens", r.settingsAPIHandler.CreateToken, authRL)
	v1.DELETE("/tokens/:id", r.settingsAPIHandler.DeleteToken, authRL)
	v1.POST("/tokens/:id/regenerate", r.settingsAPIHandler.RegenerateToken, authRL)

	// Settings: alert channels
	v1.GET("/alert-channels", r.settingsAPIHandler.ListChannels)
	v1.POST("/alert-channels", r.settingsAPIHandler.CreateChannel)
	v1.DELETE("/alert-channels/:id", r.settingsAPIHandler.DeleteChannel)
	v1.POST("/alert-channels/:id/toggle", r.settingsAPIHandler.ToggleChannel)
	v1.POST("/alert-channels/:id/test", r.settingsAPIHandler.TestChannel)

	// Settings: profile
	v1.PATCH("/users/me", r.settingsAPIHandler.UpdateProfile)

	// Status pages
	v1.GET("/status-pages", r.statusPageAPIHandler.List)
	v1.POST("/status-pages", r.statusPageAPIHandler.Create)
	v1.GET("/status-pages/:id", r.statusPageAPIHandler.Get)
	v1.PUT("/status-pages/:id", r.statusPageAPIHandler.Update)
	v1.DELETE("/status-pages/:id", r.statusPageAPIHandler.Delete)

	// System dashboard (admin-only)
	v1.GET("/system", r.systemAPIHandler.GetSystemInfo)

	// Self-service password change
	v1.POST("/users/me/password", r.settingsAPIHandler.ChangePassword)

	// Admin routes (admin-only middleware, returns JSON 403)
	admin := v1.Group("/admin")
	admin.Use(middleware.AdminRequiredJSON(r.deps.UserRepo))
	admin.GET("/users", r.systemAPIHandler.ListUsers)
	admin.POST("/users/:id/reset-password", r.systemAPIHandler.ResetUserPassword, authRL)
	admin.DELETE("/users/:id", r.systemAPIHandler.DeleteUser)
	admin.GET("/security-events", r.systemAPIHandler.GetSecurityEvents)

	// SvelteKit SPA — serve build output from root (catches all non-API routes)
	r.registerSvelteRoutes()
}

// tenantMiddleware returns the tenant scope middleware.
// If a TenantResolver is registered in the module registry, it is used.
// Otherwise, a default middleware that injects "default" is used.
func (r *Router) tenantMiddleware() echo.MiddlewareFunc {
	if r.deps.Registry != nil {
		if mod, ok := r.deps.Registry.Get("tenant_resolver"); ok {
			if resolver, ok := mod.(ports.TenantResolver); ok {
				return middleware.TenantScope(resolver)
			}
		}
	}
	// Fallback: inject "default" tenant into every request context.
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := repository.WithTenantID(c.Request().Context(), "default")
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// Stop cleans up router resources (rate limiters, session tracker).
func (r *Router) Stop() {
	if r.authRateLimiter != nil {
		r.authRateLimiter.Stop()
	}
	if r.generalRateLimiter != nil {
		r.generalRateLimiter.Stop()
	}
	if r.loginLimiter != nil {
		r.loginLimiter.Stop()
	}
	if r.registerLimiter != nil {
		r.registerLimiter.Stop()
	}
	if r.sessionTracker != nil {
		r.sessionTracker.Stop()
	}
}

// installScript serves the agent install script for curl-pipe-sh installs.
func (r *Router) installScript(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", installScriptContent)
}

// registerSvelteRoutes serves the SvelteKit build output from web/svelte/build/ at root.
// All routes that don't match an API, SSE, WS, health, or install path are
// served by the SPA (index.html fallback for client-side routing).
func (r *Router) registerSvelteRoutes() {
	buildDir := "web/svelte/build"
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		return
	}

	slog.Info("serving SvelteKit SPA from " + buildDir)

	absBuildDir, _ := filepath.Abs(buildDir)
	indexHTMLPath := filepath.Join(absBuildDir, "index.html")

	// Read index.html template once at startup.
	indexHTMLBytes, err := os.ReadFile(indexHTMLPath)
	if err != nil {
		slog.Error("failed to read SvelteKit index.html", "error", err)
		return
	}
	indexHTMLTemplate := string(indexHTMLBytes)

	// serveSPA injects the per-request CSP nonce into the SvelteKit bootstrap
	// script and serves the modified index.html.
	serveSPA := func(c echo.Context) error {
		nonce, _ := c.Get(middleware.NonceContextKey).(string)
		html := strings.Replace(indexHTMLTemplate, "<script>", `<script nonce="`+nonce+`">`, 1)
		return c.HTML(http.StatusOK, html)
	}

	// Serve SvelteKit static assets (_app/*, favicon, etc.) from the build directory.
	// Echo's file server at "/" would conflict with the wildcard, so we use a
	// catch-all handler that checks for static files first, then falls back to index.html.
	r.echo.GET("/*", func(c echo.Context) error {
		reqPath := c.Param("*")
		if reqPath == "" {
			return serveSPA(c)
		}

		// Sanitize path to prevent directory traversal
		cleanPath := filepath.Clean(reqPath)
		if strings.Contains(cleanPath, "..") {
			return serveSPA(c)
		}
		requestedFile := filepath.Join(absBuildDir, cleanPath)
		// Verify resolved path stays within build directory
		if !strings.HasPrefix(requestedFile, absBuildDir) {
			return serveSPA(c)
		}
		// H-015: resolve symlinks and verify the real path stays within build dir.
		realPath, err := filepath.EvalSymlinks(requestedFile)
		if err == nil && !strings.HasPrefix(realPath, absBuildDir) {
			return serveSPA(c)
		}
		if info, err := os.Stat(requestedFile); err == nil && !info.IsDir() {
			return c.File(requestedFile)
		}
		// Fall back to index.html for client-side routing
		return serveSPA(c)
	})

	// Root route — serve SPA
	r.echo.GET("/", serveSPA)
}

// healthCheck returns health status including module health when registry is present.
func (r *Router) healthCheck(c echo.Context) error {
	result := map[string]any{
		"status": "healthy",
	}

	if r.deps.Registry != nil {
		modules := make(map[string]string)
		for name, err := range r.deps.Registry.HealthAll(c.Request().Context()) {
			if err != nil {
				modules[name] = err.Error()
				result["status"] = "degraded"
			} else {
				modules[name] = "healthy"
			}
		}
		result["modules"] = modules
	}

	status := http.StatusOK
	if result["status"] == "degraded" {
		status = http.StatusServiceUnavailable
	}
	return c.JSON(status, result)
}
