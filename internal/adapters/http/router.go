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
	Hub              *realtime.Hub
	Logger           *slog.Logger
	SessionSecret    string
	TemplatesDir     string
	SecureCookies    bool
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
	apiHandler       *handlers.APIHandler

	// Rate limiters (kept for graceful shutdown)
	authRateLimiter    *middleware.RateLimiter
	generalRateLimiter *middleware.RateLimiter
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

	r := &Router{
		echo:      e,
		deps:      deps,
		templates: templates,
	}

	// Initialize handlers
	r.authHandler = handlers.NewAuthHandler(deps.UserAuthService, deps.UserRepo, templates)
	r.dashboardHandler = handlers.NewDashboardHandler(deps.AgentRepo, deps.MonitorRepo, deps.HeartbeatRepo, deps.IncidentService, deps.UserRepo, templates)
	r.monitorHandler = handlers.NewMonitorHandler(deps.MonitorService, deps.AgentRepo, deps.HeartbeatRepo, templates)
	r.incidentHandler = handlers.NewIncidentHandler(deps.IncidentService, deps.MonitorRepo, templates)
	r.agentHandler = handlers.NewAgentHandler(deps.AgentAuthService, deps.AgentRepo, templates)
	r.adminHandler = handlers.NewAdminHandler(deps.UserRepo, deps.AgentRepo, deps.MonitorRepo, deps.UsageEventRepo, templates)
	r.landingHandler = handlers.NewLandingHandler(deps.WaitlistRepo, templates)
	r.sseHandler = handlers.NewSSEHandler(deps.Hub, deps.AgentRepo, deps.IncidentService)
	r.wsHandler = handlers.NewWSHandler(deps.AgentAuthService, deps.MonitorService, deps.AgentRepo, deps.Hub, logger)
	r.apiHandler = handlers.NewAPIHandler(deps.HeartbeatRepo, deps.MonitorRepo, deps.AgentRepo, deps.IncidentService)

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
	e.GET("/login", r.authHandler.LoginPage)
	e.POST("/login", r.authHandler.Login, authRL)
	e.GET("/register", r.authHandler.RegisterPage)
	e.POST("/register", r.authHandler.Register, authRL)
	e.POST("/logout", r.authHandler.Logout)
	e.POST("/waitlist", r.landingHandler.JoinWaitlist, authRL)

	// Health check (public)
	e.GET("/health", r.healthCheck)

	// WebSocket endpoint for agents (public - authenticated via API key in handshake)
	e.GET("/ws/agent", r.wsHandler.HandleConnection)

	// Protected routes (auth required)
	protected := e.Group("")
	protected.Use(middleware.AuthRequired)

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

	// Admin routes
	admin := protected.Group("/admin", middleware.AdminRequired(r.deps.UserRepo))
	admin.GET("", r.adminHandler.Dashboard)
}

// rootRedirect shows the landing page for unauthenticated users,
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
}

// healthCheck returns health status.
func (r *Router) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
