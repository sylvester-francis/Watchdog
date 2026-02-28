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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/sylvester-francis/watchdog/core/registry"
	internalhttp "github.com/sylvester-francis/watchdog/internal/adapters/http"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/crypto"
	"github.com/sylvester-francis/watchdog/internal/defaults"
	"github.com/sylvester-francis/watchdog/internal/workflows"
)

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

	// Notifiers
	notifier := buildNotifier(cfg.Notify, logger)

	// Services
	auditSvc := services.NewAuditService(auditLogRepo, logger)
	authSvc := services.NewAuthService(userRepo, agentRepo, usageEventRepo, hasher, encryptor, logger)
	notifierFactory := notify.NewChannelNotifierFactory()
	incidentSvc := services.NewIncidentService(incidentRepo, monitorRepo, agentRepo, alertChannelRepo, notifier, notifierFactory, db, logger)
	monitorSvc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, incidentSvc, userRepo, usageEventRepo, logger)

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

	// Wire workflow engine for durable alert dispatch
	if wfEngine := reg.WorkflowEngine(); wfEngine != nil {
		workflows.RegisterAlertHandlers(
			wfEngine, notifier, notifierFactory,
			agentRepo, alertChannelRepo, incidentRepo, monitorRepo, logger,
		)
		incidentSvc.SetWorkflowEngine(wfEngine)
		logger.Info("durable alert dispatch enabled")
	}

	// WebSocket hub
	hub := realtime.NewHub(logger)

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

	// Router
	router, err := internalhttp.NewRouter(e, internalhttp.Dependencies{
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
		Hub:              hub,
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
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize router: %w", err)
	}

	return &Engine{
		reg:    reg,
		db:     db,
		echo:   e,
		logger: logger,
		router: router,
		hub:    hub,
		cfg:    cfg,
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

// Init initializes all registered modules and registers HTTP routes.
// Call this after registering any module overrides.
func (e *Engine) Init(ctx context.Context) error {
	if err := e.reg.InitAll(ctx); err != nil {
		return fmt.Errorf("initialize modules: %w", err)
	}

	go e.hub.Run()
	e.router.RegisterRoutes()

	return nil
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
	return nil
}

// buildNotifier creates the appropriate notifier based on configuration.
func buildNotifier(cfg config.NotifyConfig, logger *slog.Logger) notify.Notifier {
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
		multi.AddNotifier(notify.NewWebhookNotifier(cfg.WebhookURL))
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
