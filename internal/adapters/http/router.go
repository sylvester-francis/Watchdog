package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
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
	Hub              *realtime.Hub
	Hasher           *crypto.PasswordHasher
	AuditService     ports.AuditService
	Logger           *slog.Logger
	SessionSecret    string
	TemplatesDir     string
	SecureCookies    bool
	AllowedOrigins   []string
}

// Router handles HTTP routing and handler registration.
type Router struct {
	echo      *echo.Echo
	deps      Dependencies
	templates *view.Templates

	// Handlers
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	monitorHandler   *handlers.MonitorHandler
	incidentHandler  *handlers.IncidentHandler
	agentHandler     *handlers.AgentHandler
	adminHandler     *handlers.AdminHandler
	landingHandler   *handlers.LandingHandler
	sseHandler       *handlers.SSEHandler
	wsHandler        *handlers.WSHandler
	apiHandler        *handlers.APIHandler
	apiTokenHandler   *handlers.APITokenHandler
	apiV1Handler      *handlers.APIV1Handler
	statusPageHandler    *handlers.StatusPageHandler
	alertChannelHandler *handlers.AlertChannelHandler

	// Rate limiters (kept for graceful shutdown)
	authRateLimiter    *middleware.RateLimiter
	generalRateLimiter *middleware.RateLimiter
	loginLimiter       *middleware.LoginLimiter
}

// NewRouter creates a new Router instance.
func NewRouter(e *echo.Echo, deps Dependencies) (*Router, error) {
	templates, err := view.NewTemplates(deps.TemplatesDir)
	if err != nil {
		return nil, err
	}

	e.Renderer = templates

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	loginLimiter := middleware.NewLoginLimiter()

	r := &Router{
		echo:         e,
		deps:         deps,
		templates:    templates,
		loginLimiter: loginLimiter,
	}

	// Initialize handlers
	r.authHandler = handlers.NewAuthHandler(deps.UserAuthService, deps.UserRepo, templates, loginLimiter, deps.AuditService)
	r.dashboardHandler = handlers.NewDashboardHandler(deps.AgentRepo, deps.MonitorRepo, deps.HeartbeatRepo, deps.IncidentService, deps.UserRepo, templates)
	r.monitorHandler = handlers.NewMonitorHandler(deps.MonitorService, deps.AgentRepo, deps.HeartbeatRepo, templates, deps.Hub)
	r.incidentHandler = handlers.NewIncidentHandler(deps.IncidentService, deps.MonitorRepo, templates)
	r.agentHandler = handlers.NewAgentHandler(deps.AgentAuthService, deps.AgentRepo, templates)
	r.adminHandler = handlers.NewAdminHandler(deps.UserRepo, deps.AgentRepo, deps.MonitorRepo, deps.UsageEventRepo, deps.Hasher, templates)
	r.landingHandler = handlers.NewLandingHandler(deps.WaitlistRepo, templates)
	r.sseHandler = handlers.NewSSEHandler(deps.Hub, deps.AgentRepo, deps.IncidentService)
	r.wsHandler = handlers.NewWSHandler(deps.AgentAuthService, deps.MonitorService, deps.AgentRepo, deps.Hub, logger, deps.AllowedOrigins)
	r.apiHandler = handlers.NewAPIHandler(deps.HeartbeatRepo, deps.MonitorRepo, deps.AgentRepo, deps.IncidentService)
	r.apiTokenHandler = handlers.NewAPITokenHandler(deps.APITokenRepo, deps.AlertChannelRepo, deps.UserRepo, templates)
	r.apiV1Handler = handlers.NewAPIV1Handler(deps.AgentRepo, deps.MonitorRepo, deps.HeartbeatRepo, deps.IncidentService, deps.MonitorService, deps.AgentAuthService, deps.Hub)
	r.statusPageHandler = handlers.NewStatusPageHandler(deps.StatusPageRepo, deps.MonitorRepo, deps.AgentRepo, deps.HeartbeatRepo, deps.UserRepo, templates)
	r.alertChannelHandler = handlers.NewAlertChannelHandler(deps.AlertChannelRepo, templates)

	return r, nil
}

// RegisterRoutes registers all HTTP routes.
func (r *Router) RegisterRoutes() {
	e := r.echo

	// Session middleware
	store := sessions.NewCookieStore([]byte(r.deps.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   r.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	e.Use(middleware.SessionMiddleware(store))

	// CSRF protection for HTML form submissions
	e.Use(middleware.CSRFMiddleware(r.deps.SecureCookies))

	// Rate limiters
	r.authRateLimiter = middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            5,
		Burst:           10,
		CleanupInterval: 5 * time.Minute,
	})
	r.generalRateLimiter = middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())
	e.Use(r.generalRateLimiter.Middleware())

	// Static files
	e.Static("/static", "web/static")

	// Public routes (no auth required)
	authRL := r.authRateLimiter.Middleware()
	loginLL := r.loginLimiter.Middleware()
	e.GET("/login", r.authHandler.LoginPage)
	e.POST("/login", r.authHandler.Login, authRL, loginLL)
	e.GET("/register", r.authHandler.RegisterPage)
	e.POST("/register", r.authHandler.Register, authRL)
	e.POST("/logout", r.authHandler.Logout)
	e.GET("/setup", r.authHandler.SetupPage)
	e.POST("/setup", r.authHandler.Setup, authRL)
	e.POST("/waitlist", r.landingHandler.JoinWaitlist, authRL)

	// Health check (public)
	e.GET("/health", r.healthCheck)

	// API docs (public)
	e.GET("/docs", r.apiDocs)

	// Legal pages (public)
	e.GET("/terms", r.termsPage)
	e.GET("/privacy", r.privacyPage)

	// Agent install script (public)
	e.GET("/install", r.installScript)

	// Public status pages (no auth required)
	e.GET("/status/@:username/:slug", r.statusPageHandler.PublicView)

	// WebSocket endpoint for agents (public - authenticated via API key in handshake)
	e.GET("/ws/agent", r.wsHandler.HandleConnection)

	// Protected routes (auth required)
	protected := e.Group("")
	protected.Use(middleware.NoCacheHeaders)
	protected.Use(middleware.AuthRequired)
	protected.Use(middleware.UserContext(r.deps.UserRepo))

	// Root redirect
	e.GET("/", r.rootRedirect)

	// Dashboard
	protected.GET("/dashboard", r.dashboardHandler.Dashboard)

	// Agents
	protected.GET("/api/agents", r.dashboardHandler.AgentsJSON)
	protected.GET("/api/agents/:id", r.dashboardHandler.AgentJSON)
	protected.POST("/agents", r.agentHandler.Create)
	protected.DELETE("/agents/:id", r.agentHandler.Delete)

	// Monitors
	protected.GET("/monitors", r.monitorHandler.List)
	protected.GET("/monitors/new", r.monitorHandler.NewForm)
	protected.POST("/monitors", r.monitorHandler.Create)
	protected.GET("/monitors/:id", r.monitorHandler.Detail)
	protected.GET("/monitors/:id/edit", r.monitorHandler.EditForm)
	protected.POST("/monitors/:id", r.monitorHandler.Update)
	protected.DELETE("/monitors/:id", r.monitorHandler.Delete)

	// Incidents
	protected.GET("/incidents", r.incidentHandler.List)
	protected.GET("/incidents/:id", r.incidentHandler.Detail)
	protected.POST("/incidents/:id/ack", r.incidentHandler.Acknowledge)
	protected.POST("/incidents/:id/resolve", r.incidentHandler.Resolve)

	// API endpoints for chart data
	protected.GET("/api/monitors/:id/heartbeats", r.apiHandler.MonitorHeartbeats)
	protected.GET("/api/dashboard/stats", r.apiHandler.DashboardStats)
	protected.GET("/api/monitors/summary", r.apiHandler.MonitorsSummary)

	// SSE for real-time updates
	protected.GET("/sse/events", r.sseHandler.Events)

	// Status pages
	protected.GET("/status-pages", r.statusPageHandler.List)
	protected.POST("/status-pages", r.statusPageHandler.Create)
	protected.GET("/status-pages/:id/edit", r.statusPageHandler.Edit)
	protected.POST("/status-pages/:id", r.statusPageHandler.Update)
	protected.DELETE("/status-pages/:id", r.statusPageHandler.Delete)

	// Settings (API tokens + alert channels)
	protected.GET("/settings", r.apiTokenHandler.List)
	protected.POST("/settings/tokens", r.apiTokenHandler.Create)
	protected.DELETE("/settings/tokens/:id", r.apiTokenHandler.Delete)
	protected.POST("/settings/alerts", r.alertChannelHandler.Create)
	protected.DELETE("/settings/alerts/:id", r.alertChannelHandler.Delete)
	protected.POST("/settings/alerts/:id/toggle", r.alertChannelHandler.Toggle)
	protected.POST("/settings/alerts/:id/test", r.alertChannelHandler.TestChannel)
	protected.POST("/settings/username", r.apiTokenHandler.UpdateUsername)

	// Admin routes
	admin := protected.Group("/admin", middleware.AdminRequired(r.deps.UserRepo))
	admin.GET("", r.adminHandler.Dashboard)
	admin.POST("/users", r.adminHandler.CreateUser)
	admin.POST("/users/:id", r.adminHandler.UpdateUser)
	admin.DELETE("/users/:id", r.adminHandler.DeleteUser)

	// Public API v1 (token-authenticated)
	v1 := e.Group("/api/v1")
	v1.Use(middleware.APITokenAuth(r.deps.APITokenRepo, r.deps.UserRepo))
	v1.GET("/monitors", r.apiV1Handler.ListMonitors)
	v1.GET("/monitors/:id", r.apiV1Handler.GetMonitor)
	v1.POST("/monitors", r.apiV1Handler.CreateMonitor)
	v1.PUT("/monitors/:id", r.apiV1Handler.UpdateMonitor)
	v1.DELETE("/monitors/:id", r.apiV1Handler.DeleteMonitor)
	v1.GET("/agents", r.apiV1Handler.ListAgents)
	v1.POST("/agents", r.apiV1Handler.CreateAgent)
	v1.DELETE("/agents/:id", r.apiV1Handler.DeleteAgent)
	v1.GET("/incidents", r.apiV1Handler.ListIncidents)
	v1.POST("/incidents/:id/acknowledge", r.apiV1Handler.AcknowledgeIncident)
	v1.POST("/incidents/:id/resolve", r.apiV1Handler.ResolveIncident)
	v1.GET("/dashboard/stats", r.apiV1Handler.DashboardStats)
}

// rootRedirect shows the setup wizard when no users exist,
// the landing page for unauthenticated users,
// or redirects to dashboard for authenticated users.
func (r *Router) rootRedirect(c echo.Context) error {
	if middleware.IsAuthenticated(c) {
		return c.Redirect(http.StatusFound, "/dashboard")
	}
	return r.landingHandler.Page(c)
}

// Stop cleans up router resources (rate limiters).
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
}

// apiDocs renders the Swagger UI page.
func (r *Router) apiDocs(c echo.Context) error {
	return c.Render(http.StatusOK, "api_docs.html", nil)
}

// installScript serves the agent install script for curl-pipe-sh installs.
func (r *Router) installScript(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", installScriptContent)
}

// termsPage renders the Terms of Service page.
func (r *Router) termsPage(c echo.Context) error {
	return c.Render(http.StatusOK, "terms.html", map[string]interface{}{
		"Title": "Terms of Service",
	})
}

// privacyPage renders the Privacy Policy page.
func (r *Router) privacyPage(c echo.Context) error {
	return c.Render(http.StatusOK, "privacy.html", map[string]interface{}{
		"Title": "Privacy Policy",
	})
}

// healthCheck returns health status.
func (r *Router) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
