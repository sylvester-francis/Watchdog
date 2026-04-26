package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
	internalhttp "github.com/sylvester-francis/watchdog/internal/adapters/http"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	prommetrics "github.com/sylvester-francis/watchdog/internal/adapters/metrics"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/adapters/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/crypto"
	"github.com/sylvester-francis/watchdog/internal/defaults"
	"github.com/sylvester-francis/watchdog/internal/workflows"
)

// HeartbeatHook is called after each heartbeat is processed.
// Extensions can use this to add custom post-processing.
type HeartbeatHook func(ctx context.Context, agentID, monitorID uuid.UUID, payload *protocol.HeartbeatPayload)

// MaintenanceExpiredHook is called when a maintenance window expires with an offline agent.
// Extensions (e.g. EE) can use this for audit logging.
type MaintenanceExpiredHook func(ctx context.Context, mw *domain.MaintenanceWindow)

// MaintenanceTenantProvider returns all tenant IDs that have maintenance windows.
// Used by the background ticker to iterate over tenants for RLS-safe queries.
// If nil, only the "default" tenant is processed.
type MaintenanceTenantProvider func(ctx context.Context) []string

// TenantIDFromContext extracts the tenant ID from a request context.
// Returns "default" if no tenant ID is set.
func TenantIDFromContext(ctx context.Context) string {
	return repository.TenantIDFromContext(ctx)
}

// Engine wraps all application components and manages the lifecycle.
// Usage: New() -> (optional Registry().Register overrides) -> Init() -> Run()
type Engine struct {
	reg    *registry.Registry
	db     *repository.DB
	echo   *echo.Echo
	logger *slog.Logger
	router *internalhttp.Router
	hub    *realtime.Hub
	cfg    *config.Config

	metrics *prommetrics.Metrics

	// Exposed for extensions via accessor methods.
	agentRepo          ports.AgentRepository
	monitorRepo        ports.MonitorRepository
	heartbeatRepo      ports.HeartbeatRepository
	incidentRepo       ports.IncidentRepository
	certDetailsRepo    ports.CertDetailsRepository
	statusPageRepo     ports.StatusPageRepository
	incidentSvc        ports.IncidentService
	monitorSvc         ports.MonitorService
	investigationSvc   ports.InvestigationService
	concreteMonitorSvc *services.MonitorService
	agentAuthSvc       ports.AgentAuthService
	auditSvc           ports.AuditService
	mwRepo             ports.MaintenanceWindowRepository
	traceRetentionSvc  *services.TraceRetention
	logRetentionSvc    *services.LogRetention

	// Maintenance window background processing hooks.
	mwExpiredHooks    []MaintenanceExpiredHook
	mwTenantProvider  MaintenanceTenantProvider

	// rateLimitOnReject is forwarded to the router during Init() so the
	// general rate limiter can invoke it whenever a request is denied.
	rateLimitOnReject func(ip string)

	// telemetryShutdown flushes pending OTel spans and log records
	// during graceful shutdown. Always non-nil; a no-op when telemetry
	// is disabled or no OTLP endpoint is configured.
	telemetryShutdown func(context.Context) error
}

// SetRateLimitOnReject sets the callback that the general rate limiter will
// invoke whenever a request is denied for exceeding the burst limit.
// The callback receives the client IP string. Pass nil to clear.
//
// Must be called before Init() — the router constructs the limiter during
// Init and the callback value is captured at that point.
//
// Use case: extension modules wire this to persistent block-list trackers
// (e.g. write to a blocked_ips table) without modifying the rate-limiter
// or middleware code paths.
func (e *Engine) SetRateLimitOnReject(fn func(ip string)) {
	e.rateLimitOnReject = fn
}

// New creates a new Engine, loading config, connecting to the database,
// creating all repositories, services, notifiers, and registering default
// modules. It does NOT call InitAll or start the server, allowing callers
// to register module overrides before initialization.
func New(ctx context.Context) (*Engine, error) {
	logger := slog.Default()

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// OpenTelemetry SDK init runs BEFORE any subsystem captures `logger`
	// so the slog tee handler covers the whole engine lifecycle. Both
	// providers no-op when WATCHDOG_OTEL_ENABLED=false or when no OTLP
	// endpoint is configured — see internal/adapters/telemetry.
	tracerProvider, traceShutdown, err := telemetry.NewTracerProvider(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("initialize tracer provider: %w", err)
	}
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	loggerProvider, logShutdown, err := telemetry.NewLoggerProvider(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("initialize logger provider: %w", err)
	}
	logger = slog.New(telemetry.NewSlogHandler(logger.Handler(), loggerProvider, cfg.Telemetry.ServiceName))
	slog.SetDefault(logger)

	meterProvider, meterShutdown, err := telemetry.NewMeterProvider(ctx, cfg.Telemetry)
	if err != nil {
		return nil, fmt.Errorf("initialize meter provider: %w", err)
	}
	otel.SetMeterProvider(meterProvider)

	telemetryShutdown := func(ctx context.Context) error {
		traceErr := traceShutdown(ctx)
		logErr := logShutdown(ctx)
		meterErr := meterShutdown(ctx)
		if traceErr != nil {
			return traceErr
		}
		if logErr != nil {
			return logErr
		}
		return meterErr
	}

	db, err := repository.NewDB(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	logger.Info("connected to database")

	hasher := crypto.NewPasswordHasher()
	encryptor, err := crypto.NewEncryptor(cfg.Crypto.EncryptionKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize encryptor: %w", err)
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	agentRepo := repository.NewAgentRepository(db)
	monitorRepo := repository.NewMonitorRepository(db)
	heartbeatRepo := repository.NewHeartbeatRepository(db)
	incidentRepo := repository.NewIncidentRepository(db)
	usageEventRepo := repository.NewUsageEventRepository(db)
	waitlistRepo := repository.NewWaitlistRepository(db)
	apiTokenRepo := repository.NewAPITokenRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	statusPageRepo := repository.NewStatusPageRepository(db)
	alertChannelRepo := repository.NewAlertChannelRepository(db, encryptor)
	certDetailsRepo := repository.NewCertDetailsRepository(db)
	spanRepo := repository.NewSpanRepository(db)
	logRecordRepo := repository.NewLogRecordRepository(db)
	systemSettingsRepo := repository.NewSystemSettingsRepository(db)

	// Notifiers
	notifier := buildNotifier(cfg.Notify, logger)

	// Services
	auditSvc := services.NewAuditService(auditLogRepo, logger)
	authSvc := services.NewAuthService(userRepo, agentRepo, usageEventRepo, hasher, encryptor, logger)
	notifierFactory := notify.NewChannelNotifierFactory()
	incidentSvc := services.NewIncidentService(incidentRepo, monitorRepo, agentRepo, heartbeatRepo, alertChannelRepo, notifier, notifierFactory, db, logger)
	monitorSvc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, incidentSvc, userRepo, usageEventRepo, logger)
	investigationSvc := services.NewInvestigationService(incidentRepo, monitorRepo, agentRepo, heartbeatRepo, certDetailsRepo, logger)
	traceRetentionSvc := services.NewTraceRetention(spanRepo, systemSettingsRepo, logger)
	logRetentionSvc := services.NewLogRetention(logRecordRepo, systemSettingsRepo, logger)

	// Module registry with defaults
	reg := registry.New(logger)
	defaults.RegisterAll(reg, defaults.Deps{
		AuthService:    authSvc,
		AgentAuth:      authSvc,
		AgentRepo:      agentRepo,
		Notifier:       notifier,
		AuditService:   auditSvc,
		StatusPageRepo: statusPageRepo,
		DB:             db,
		Pool:           db.Pool,
		DurableAlerts:  cfg.Feature.DurableAlerts,
		Logger:         logger,
	})

	// WebSocket hub (created before workflow wiring so discovery handlers can reference it)
	hub := realtime.NewHub(logger)

	// Wire workflow engine for durable alert dispatch + discovery
	if wfEngine := reg.WorkflowEngine(); wfEngine != nil {
		workflows.RegisterAlertHandlers(
			wfEngine, notifier, notifierFactory,
			agentRepo, heartbeatRepo, alertChannelRepo, incidentRepo, monitorRepo, logger,
		)
		incidentSvc.SetWorkflowEngine(wfEngine)
		logger.Info("durable alert dispatch enabled")
	}

	// Echo + middleware
	e := echo.New()
	e.HideBanner = true

	e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogError:    true,
		LogValuesFunc: func(_ echo.Context, v echomw.RequestLoggerValues) error {
			if v.Error != nil {
				logger.Error("request",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("remote_ip", v.RemoteIP),
					slog.String("error", v.Error.Error()),
				)
			} else {
				logger.Info("request",
					slog.String("method", v.Method),
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.String("remote_ip", v.RemoteIP),
				)
			}
			return nil
		},
	}))
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(middleware.SecureHeaders(cfg.Server.SecureCookies))

	// OTel HTTP server-span middleware. The TracerProvider was set as
	// the global earlier in New() (right after config load) so otelecho
	// picks it up automatically.
	e.Use(otelecho.Middleware(cfg.Telemetry.ServiceName))

	// Prometheus metrics. The OTel meter is the single source of truth;
	// values flow out via the otelprom reader (default Prom registerer)
	// to /metrics, plus the OTLP push reader when telemetry is enabled.
	meter := otel.Meter(cfg.Telemetry.ServiceName)
	prom, err := prommetrics.New(meter, hub, db.Pool)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}
	e.Use(prom.HTTPMiddleware())
	// /metrics requires admin session — protects Prometheus scrape data from public access
	e.GET("/metrics", func(c echo.Context) error {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			return c.JSON(401, map[string]string{"error": "unauthorized"})
		}
		caller, err := userRepo.GetByID(c.Request().Context(), userID)
		if err != nil || caller == nil || !caller.IsAdmin {
			return c.JSON(403, map[string]string{"error": "admin access required"})
		}
		return prommetrics.Handler()(c)
	})

	// Maintenance windows — create and wire into monitor service
	mwRepo := repository.NewMaintenanceWindowRepository(db)
	monitorSvc.SetMaintenanceWindowRepo(mwRepo)
	monitorSvc.SetAuditService(auditSvc)
	monitorSvc.SetTransactor(db) // RLS-safe maintenance window checks

	// Router
	routerDeps := internalhttp.Dependencies{
		UserAuthService:  authSvc,
		AgentAuthService: authSvc,
		MonitorService:   monitorSvc,
		IncidentService:  incidentSvc,
		UserRepo:         userRepo,
		AgentRepo:        agentRepo,
		MonitorRepo:      monitorRepo,
		HeartbeatRepo:    heartbeatRepo,
		UsageEventRepo:   usageEventRepo,
		WaitlistRepo:     waitlistRepo,
		APITokenRepo:     apiTokenRepo,
		StatusPageRepo:   statusPageRepo,
		AlertChannelRepo: alertChannelRepo,
		SpanRepo:         spanRepo,
		LogRecordRepo:    logRecordRepo,
		CertDetailsRepo:       certDetailsRepo,
		MaintenanceWindowRepo: mwRepo,
		Hub:                   hub,
		Hasher:           hasher,
		AuditService:     auditSvc,
		AuditLogRepo:     auditLogRepo,
		DB:               db,
		Config:           cfg,
		StartTime:        time.Now(),
		Logger:           logger,
		SessionSecret:    cfg.Crypto.SessionSecret,
		SecureCookies:    cfg.Server.SecureCookies,
		AllowedOrigins:   cfg.Server.AllowedOrigins,
		Registry:         reg,
	}
	router, err := internalhttp.NewRouter(e, routerDeps)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize router: %w", err)
	}

	// Wire investigation service into the API handler
	router.APIV1Handler().SetInvestigationService(investigationSvc)

	// Wire discovery service
	discoveryRepo := repository.NewDiscoveryRepository(db)
	discoverySvc := services.NewDiscoveryService(discoveryRepo, agentRepo, monitorSvc, hub, logger)

	// Wire workflow engine for durable discovery scans
	if wfEngine := reg.WorkflowEngine(); wfEngine != nil {
		workflows.RegisterDiscoveryHandlers(wfEngine, hub, discoveryRepo, logger)
		discoverySvc.SetWorkflowEngine(wfEngine)

		// On agent disconnect, fail any waiting discovery workflows for that agent
		hub.OnDisconnect(func(agentID uuid.UUID) {
			scans, scanErr := discoveryRepo.GetActiveScansByAgentID(context.Background(), agentID)
			if scanErr != nil {
				logger.Error("disconnect: failed to query scans", slog.String("error", scanErr.Error()))
				return
			}
			for _, scan := range scans {
				corrKey := workflows.CorrelationKeyForScan(scan.ID)
				resumeErr := wfEngine.ResumeStep(
					context.Background(), corrKey, nil,
					fmt.Errorf("agent %s disconnected", agentID),
				)
				if resumeErr != nil {
					// Not an error if the scan wasn't using the workflow engine
					logger.Debug("disconnect: no waiting workflow for scan",
						slog.String("scan_id", scan.ID.String()),
					)
				} else {
					logger.Info("disconnect: failed waiting discovery workflow",
						slog.String("scan_id", scan.ID.String()),
						slog.String("agent_id", agentID.String()),
					)
				}
			}
		})
		logger.Info("durable discovery scans enabled")
	}

	discoveryHandler := handlers.NewDiscoveryHandler(discoverySvc, agentRepo)
	router.SetDiscoveryHandler(discoveryHandler)

	router.WSHandler().SetDiscoveryHook(func(ctx context.Context, payload *protocol.DiscoveryResultPayload) {
		if err := discoverySvc.ProcessResult(ctx, payload); err != nil {
			logger.Error("failed to process discovery result", slog.String("error", err.Error()))
		}
	})

	// Wire agent auto-update service if manifest URL is configured.
	if cfg.Feature.AgentUpdateManifestURL != "" {
		updateSvc := services.NewUpdateService(cfg.Feature.AgentUpdateManifestURL, logger)
		updateSvc.Start(ctx)
		router.WSHandler().SetUpdateService(updateSvc)
		router.APIV1Handler().SetUpdateService(updateSvc)
		logger.Info("agent auto-update enabled",
			slog.String("manifest_url", cfg.Feature.AgentUpdateManifestURL),
		)
	}

	// Wire Prometheus heartbeat latency observer
	router.WSHandler().SetHeartbeatTimer(prom.ObserveHeartbeat)

	// Wire metrics history for in-app dashboard
	router.SystemAPIHandler().SetMetricsHistory(prom.History())

	return &Engine{
		reg:     reg,
		db:      db,
		echo:    e,
		logger:  logger,
		router:  router,
		hub:     hub,
		cfg:     cfg,
		metrics: prom,

		agentRepo:          agentRepo,
		monitorRepo:        monitorRepo,
		heartbeatRepo:      heartbeatRepo,
		incidentRepo:       incidentRepo,
		certDetailsRepo:    certDetailsRepo,
		statusPageRepo:     statusPageRepo,
		incidentSvc:        incidentSvc,
		monitorSvc:         monitorSvc,
		investigationSvc:   investigationSvc,
		concreteMonitorSvc: monitorSvc,
		agentAuthSvc:       authSvc,
		auditSvc:           auditSvc,
		mwRepo:             mwRepo,
		traceRetentionSvc:  traceRetentionSvc,
		logRetentionSvc:    logRetentionSvc,

		telemetryShutdown: telemetryShutdown,
	}, nil
}

// Registry returns the module registry for registering overrides.
func (e *Engine) Registry() *registry.Registry {
	return e.reg
}

// Pool returns the underlying database connection pool.
func (e *Engine) Pool() *pgxpool.Pool {
	return e.db.Pool
}

// Echo returns the underlying Echo instance for route extensions.
func (e *Engine) Echo() *echo.Echo {
	return e.echo
}

// Logger returns the configured logger.
func (e *Engine) Logger() *slog.Logger {
	return e.logger
}

// AuthMiddleware returns an Echo middleware that authenticates requests via
// session cookie and sets "user_id" in the Echo context. This is exposed so
// that external route groups (which cannot import internal packages) can reuse the
// same session-based auth that CE uses. Returns JSON 401 on failure.
func (e *Engine) AuthMiddleware() echo.MiddlewareFunc {
	return middleware.AuthRequiredAPI
}

// TenantMiddleware returns the tenant-scoping middleware that injects
// tenant_id into the request context. External route groups need this to ensure
// repository queries are correctly scoped by tenant.
func (e *Engine) TenantMiddleware() echo.MiddlewareFunc {
	return e.router.TenantMiddleware()
}

// AgentRepo returns the agent repository for extensions.
func (e *Engine) AgentRepo() ports.AgentRepository { return e.agentRepo }

// MonitorRepo returns the monitor repository for extensions.
func (e *Engine) MonitorRepo() ports.MonitorRepository { return e.monitorRepo }

// HeartbeatRepo returns the heartbeat repository for extensions.
func (e *Engine) HeartbeatRepo() ports.HeartbeatRepository { return e.heartbeatRepo }

// CertDetailsRepo returns the certificate details repository for extensions.
func (e *Engine) CertDetailsRepo() ports.CertDetailsRepository { return e.certDetailsRepo }

// StatusPageRepo returns the status page repository for extensions.
func (e *Engine) StatusPageRepo() ports.StatusPageRepository { return e.statusPageRepo }

// IncidentService returns the incident service for extensions.
func (e *Engine) IncidentService() ports.IncidentService { return e.incidentSvc }

// MonitorService returns the monitor service for extensions.
func (e *Engine) MonitorService() ports.MonitorService { return e.monitorSvc }

// AgentAuthService returns the agent auth service for extensions.
func (e *Engine) AgentAuthService() ports.AgentAuthService { return e.agentAuthSvc }

// AuditService returns the audit service for extensions.
func (e *Engine) AuditService() ports.AuditService { return e.auditSvc }

// InvestigationService returns the investigation service for extensions.
func (e *Engine) InvestigationService() ports.InvestigationService { return e.investigationSvc }

// IncidentRepo returns the incident repository for extensions.
func (e *Engine) IncidentRepo() ports.IncidentRepository { return e.incidentRepo }

// NewMaintenanceWindowRepo returns the maintenance window repository.
// The repo is already wired into MonitorService during New().
// Kept for compatibility.
func (e *Engine) NewMaintenanceWindowRepo() ports.MaintenanceWindowRepository {
	return e.mwRepo
}

// AgentMessenger returns the WebSocket hub as an AgentMessenger for extensions.
func (e *Engine) AgentMessenger() ports.AgentMessenger { return e.hub }

// SetIncidents updates the Prometheus incidents_active gauge.
func (e *Engine) SetIncidents(open, acknowledged int) {
	if e.metrics != nil {
		e.metrics.SetIncidents(open, acknowledged)
	}
}

// SetTenantValidator sets the hook for tenant-scoped registration.
// When set, registration requires a valid tenant_slug and creates users in
// the specified tenant rather than the default.
func (e *Engine) SetTenantValidator(v handlers.TenantValidator) {
	e.router.AuthAPIHandler().SetTenantValidator(v)
}

// SetPostRegisterHook sets the hook for post-registration actions.
func (e *Engine) SetPostRegisterHook(hook handlers.PostRegisterHook) {
	e.router.AuthAPIHandler().SetPostRegisterHook(hook)
}

// AddHeartbeatHook registers a hook to be called after each heartbeat is processed.
// Extensions can use this for post-processing heartbeat data.
func (e *Engine) AddHeartbeatHook(hook HeartbeatHook) {
	e.router.WSHandler().AddHeartbeatHook(handlers.HeartbeatHook(hook))
}

// OnMaintenanceExpired registers a hook called when a maintenance window expires
// with an offline agent. EE uses this for audit logging.
func (e *Engine) OnMaintenanceExpired(hook MaintenanceExpiredHook) {
	e.mwExpiredHooks = append(e.mwExpiredHooks, hook)
}

// SetMaintenanceTenantProvider sets the provider for listing tenant IDs.
// When set, the background maintenance ticker iterates over all tenants
// instead of only the "default" tenant. EE sets this from the tenants table.
func (e *Engine) SetMaintenanceTenantProvider(p MaintenanceTenantProvider) {
	e.mwTenantProvider = p
}

// Init initializes all registered modules and registers HTTP routes.
// Call this after registering any module overrides.
func (e *Engine) Init(ctx context.Context) error {
	if err := e.reg.InitAll(ctx); err != nil {
		return fmt.Errorf("initialize modules: %w", err)
	}

	go e.hub.Run()

	// Start metrics history snapshots.
	e.metrics.History().Start(ctx)

	// Forward any rate-limit OnReject callback set via SetRateLimitOnReject
	// before the router constructs its limiters in RegisterRoutes.
	e.router.SetRateLimitOnReject(e.rateLimitOnReject)

	e.router.RegisterRoutes()

	// Background maintenance window processor (60s tick).
	if e.mwRepo != nil {
		go e.runMaintenanceTicker(ctx)
	}

	// Background trace retention worker (hourly tick) — prunes old
	// spans according to system_settings.trace_retention_days.
	e.traceRetentionSvc.Start(ctx)

	// Background log retention worker (hourly tick) — prunes old
	// log records according to system_settings.log_retention_days.
	e.logRetentionSvc.Start(ctx)

	return nil
}

// runMaintenanceTicker processes expired maintenance windows every 60 seconds.
func (e *Engine) runMaintenanceTicker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.processMaintenanceWindows(ctx)
		}
	}
}

// processMaintenanceWindows handles expired windows, recurring regeneration, and cleanup.
// Iterates over all tenants (via provider if set, otherwise just "default") so that
// RLS-protected queries return the correct data for each tenant.
func (e *Engine) processMaintenanceWindows(ctx context.Context) {
	tenants := []string{"default"}
	if e.mwTenantProvider != nil {
		tenants = e.mwTenantProvider(ctx)
	}

	for _, tenantID := range tenants {
		tCtx := repository.WithTenantID(ctx, tenantID)
		e.processMaintenanceWindowsForTenant(tCtx)
	}
}

// processMaintenanceWindowsForTenant runs maintenance window processing for a single tenant.
func (e *Engine) processMaintenanceWindowsForTenant(ctx context.Context) {
	// 1. Expired windows with offline agents — mark monitors down.
	var expired []*domain.MaintenanceWindow
	if txErr := e.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var err error
		expired, err = e.mwRepo.GetExpiredWithOfflineAgents(txCtx)
		return err
	}); txErr != nil {
		e.logger.Error("maintenance: failed to query expired windows",
			slog.String("error", txErr.Error()),
		)
	}

	for _, mw := range expired {
		// Use the window's own tenant context for downstream operations.
		mwCtx := repository.WithTenantID(ctx, mw.TenantID)
		if err := e.monitorSvc.MarkAgentMonitorsDown(mwCtx, mw.AgentID); err != nil {
			e.logger.Error("maintenance: failed to mark monitors down",
				slog.String("agent_id", mw.AgentID.String()),
				slog.String("error", err.Error()),
			)
		}
		// Fire expired hooks (e.g. EE audit logging).
		for _, hook := range e.mwExpiredHooks {
			hook(mwCtx, mw)
		}
	}

	// 2. Expired recurring windows — advance to next occurrence in-place.
	var recurring []*domain.MaintenanceWindow
	if txErr := e.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var err error
		recurring, err = e.mwRepo.GetExpiredRecurring(txCtx)
		return err
	}); txErr != nil {
		e.logger.Error("maintenance: failed to query expired recurring windows",
			slog.String("error", txErr.Error()),
		)
	}

	for _, mw := range recurring {
		if mw.AdvanceToNext() {
			if err := e.db.WithTransaction(ctx, func(txCtx context.Context) error {
				return e.mwRepo.AdvanceRecurringWindow(txCtx, mw)
			}); err != nil {
				e.logger.Error("maintenance: failed to advance recurring window",
					slog.String("window_id", mw.ID.String()),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	// 3. Cleanup windows older than 30 days.
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	if err := e.db.WithTransaction(ctx, func(txCtx context.Context) error {
		return e.mwRepo.DeleteExpired(txCtx, cutoff)
	}); err != nil {
		e.logger.Error("maintenance: failed to cleanup old windows",
			slog.String("error", err.Error()),
		)
	}
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM.
func (e *Engine) Run(ctx context.Context) error {
	addr := e.cfg.Server.Address()

	go func() {
		e.logger.Info("starting server", slog.String("address", addr))
		if err := e.echo.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	fmt.Printf("\n\U0001F415 WatchDog Hub running on http://localhost:%d\n\n", e.cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return e.Shutdown(shutdownCtx)
}

// Shutdown performs graceful shutdown of all components.
func (e *Engine) Shutdown(ctx context.Context) error {
	e.hub.Stop()
	e.router.Stop()

	if err := e.reg.ShutdownAll(ctx); err != nil {
		e.logger.Error("module shutdown error", slog.String("error", err.Error()))
	}

	e.db.Close()

	if err := e.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("echo shutdown: %w", err)
	}

	// Flush any batched OTel spans last so spans from the in-flight
	// requests Echo just drained get exported before the process exits.
	if err := e.telemetryShutdown(ctx); err != nil {
		e.logger.Error("telemetry shutdown error", slog.String("error", err.Error()))
	}
	return nil
}

// buildNotifier creates the appropriate notifier based on configuration.
func buildNotifier(cfg config.NotifyConfig, logger *slog.Logger) notify.Notifier {
	notify.SetBrandName(cfg.BrandName)

	multi := notify.NewMultiNotifier()
	count := 0

	if cfg.SlackWebhookURL != "" {
		multi.AddNotifier(notify.NewSlackNotifier(cfg.SlackWebhookURL))
		logger.Info("slack notifier enabled")
		count++
	}
	if cfg.DiscordWebhookURL != "" {
		multi.AddNotifier(notify.NewDiscordNotifier(cfg.DiscordWebhookURL))
		logger.Info("discord notifier enabled")
		count++
	}
	if cfg.WebhookURL != "" {
		multi.AddNotifier(notify.NewWebhookNotifier(cfg.WebhookURL, ""))
		logger.Info("webhook notifier enabled")
		count++
	}
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" && cfg.SMTPTo != "" {
		multi.AddNotifier(notify.NewEmailNotifier(notify.EmailConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			To:       cfg.SMTPTo,
		}))
		logger.Info("email notifier enabled")
		count++
	}
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		multi.AddNotifier(notify.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID))
		logger.Info("telegram notifier enabled")
		count++
	}
	if cfg.PagerDutyRoutingKey != "" {
		multi.AddNotifier(notify.NewPagerDutyNotifier(cfg.PagerDutyRoutingKey))
		logger.Info("pagerduty notifier enabled")
		count++
	}

	if count > 0 {
		return multi
	}

	logger.Info("no notifiers configured, alerts disabled")
	return notify.NewNoOpNotifier()
}
