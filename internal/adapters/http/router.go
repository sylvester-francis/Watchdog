package http

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	"github.com/sylvester/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester/watchdog/internal/adapters/http/view"
	"github.com/sylvester/watchdog/internal/core/ports"
	"github.com/sylvester/watchdog/internal/core/realtime"
)

// Dependencies holds all the dependencies required by the router.
type Dependencies struct {
	AuthService    ports.AuthService
	MonitorService ports.MonitorService
	IncidentService ports.IncidentService
	UserRepo       ports.UserRepository
	AgentRepo      ports.AgentRepository
	MonitorRepo    ports.MonitorRepository
	Hub            *realtime.Hub
	SessionSecret  string
	TemplatesDir   string
}

// Router handles HTTP routing and handler registration.
type Router struct {
	echo       *echo.Echo
	deps       Dependencies
	templates  *view.Templates

	// Handlers
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	monitorHandler   *handlers.MonitorHandler
	incidentHandler  *handlers.IncidentHandler
	sseHandler       *handlers.SSEHandler
}

// NewRouter creates a new Router instance.
func NewRouter(e *echo.Echo, deps Dependencies) (*Router, error) {
	// Initialize templates
	templates, err := view.NewTemplates(deps.TemplatesDir)
	if err != nil {
		return nil, err
	}

	// Set Echo's renderer
	e.Renderer = templates

	r := &Router{
		echo:      e,
		deps:      deps,
		templates: templates,
	}

	// Initialize handlers
	r.authHandler = handlers.NewAuthHandler(deps.AuthService, deps.UserRepo, templates)
	r.dashboardHandler = handlers.NewDashboardHandler(deps.AgentRepo, deps.MonitorRepo, deps.IncidentService, templates)
	r.monitorHandler = handlers.NewMonitorHandler(deps.MonitorService, deps.AgentRepo, templates)
	r.incidentHandler = handlers.NewIncidentHandler(deps.IncidentService, deps.MonitorRepo, templates)
	r.sseHandler = handlers.NewSSEHandler(deps.Hub, deps.AgentRepo, deps.IncidentService)

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
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	e.Use(middleware.SessionMiddleware(store))

	// Static files
	e.Static("/static", "web/static")

	// Public routes (no auth required)
	e.GET("/login", r.authHandler.LoginPage)
	e.POST("/login", r.authHandler.Login)
	e.GET("/register", r.authHandler.RegisterPage)
	e.POST("/register", r.authHandler.Register)
	e.POST("/logout", r.authHandler.Logout)

	// Health check (public)
	e.GET("/health", r.healthCheck)

	// Protected routes (auth required)
	protected := e.Group("")
	protected.Use(middleware.AuthRequired)

	// Root redirect
	e.GET("/", r.rootRedirect)

	// Dashboard
	protected.GET("/dashboard", r.dashboardHandler.Dashboard)

	// Agents API (for HTMX)
	protected.GET("/api/agents", r.dashboardHandler.AgentsJSON)
	protected.GET("/api/agents/:id", r.dashboardHandler.AgentJSON)

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

	// SSE for real-time updates
	protected.GET("/sse/events", r.sseHandler.Events)
}

// rootRedirect redirects to dashboard if logged in, otherwise to login.
func (r *Router) rootRedirect(c echo.Context) error {
	if !middleware.IsAuthenticated(c) {
		return c.Redirect(http.StatusFound, "/login")
	}
	return c.Redirect(http.StatusFound, "/dashboard")
}

// healthCheck returns health status.
func (r *Router) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
