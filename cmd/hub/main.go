package main

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

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	internalhttp "github.com/sylvester-francis/watchdog/internal/adapters/http"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/crypto"
	"github.com/sylvester-francis/watchdog/internal/defaults"
)

func main() {
	// Setup logger
	logger := slog.Default()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to database
	db, err := repository.NewDB(context.Background(), cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("connected to database")

	// Initialize crypto services
	hasher := crypto.NewPasswordHasher()
	encryptor, err := crypto.NewEncryptor(cfg.Crypto.EncryptionKey)
	if err != nil {
		logger.Error("failed to initialize encryptor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize repositories
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

	// Initialize notifiers from environment configuration
	var notifier notify.Notifier
	multi := notify.NewMultiNotifier()
	notifierCount := 0

	if cfg.Notify.SlackWebhookURL != "" {
		multi.AddNotifier(notify.NewSlackNotifier(cfg.Notify.SlackWebhookURL))
		logger.Info("slack notifier enabled")
		notifierCount++
	}
	if cfg.Notify.DiscordWebhookURL != "" {
		multi.AddNotifier(notify.NewDiscordNotifier(cfg.Notify.DiscordWebhookURL))
		logger.Info("discord notifier enabled")
		notifierCount++
	}
	if cfg.Notify.WebhookURL != "" {
		multi.AddNotifier(notify.NewWebhookNotifier(cfg.Notify.WebhookURL))
		logger.Info("webhook notifier enabled")
		notifierCount++
	}
	if cfg.Notify.SMTPHost != "" && cfg.Notify.SMTPFrom != "" && cfg.Notify.SMTPTo != "" {
		multi.AddNotifier(notify.NewEmailNotifier(notify.EmailConfig{
			Host:     cfg.Notify.SMTPHost,
			Port:     cfg.Notify.SMTPPort,
			Username: cfg.Notify.SMTPUsername,
			Password: cfg.Notify.SMTPPassword,
			From:     cfg.Notify.SMTPFrom,
			To:       cfg.Notify.SMTPTo,
		}))
		logger.Info("email notifier enabled")
		notifierCount++
	}
	if cfg.Notify.TelegramBotToken != "" && cfg.Notify.TelegramChatID != "" {
		multi.AddNotifier(notify.NewTelegramNotifier(cfg.Notify.TelegramBotToken, cfg.Notify.TelegramChatID))
		logger.Info("telegram notifier enabled")
		notifierCount++
	}
	if cfg.Notify.PagerDutyRoutingKey != "" {
		multi.AddNotifier(notify.NewPagerDutyNotifier(cfg.Notify.PagerDutyRoutingKey))
		logger.Info("pagerduty notifier enabled")
		notifierCount++
	}

	if notifierCount > 0 {
		notifier = multi
	} else {
		notifier = notify.NewNoOpNotifier()
		logger.Info("no notifiers configured, alerts disabled")
	}

	// Initialize services
	auditSvc := services.NewAuditService(auditLogRepo, logger)
	authSvc := services.NewAuthService(userRepo, agentRepo, usageEventRepo, hasher, encryptor, logger)
	notifierFactory := notify.NewChannelNotifierFactory()
	incidentSvc := services.NewIncidentService(incidentRepo, monitorRepo, agentRepo, alertChannelRepo, notifier, notifierFactory, db, logger)
	monitorSvc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, incidentSvc, userRepo, usageEventRepo, logger)

	// Initialize templates
	templates, err := view.NewTemplates("web/templates")
	if err != nil {
		logger.Error("failed to load templates", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize module registry
	reg := registry.New(logger)
	defaults.RegisterAll(reg, defaults.Deps{
		AuthService:    authSvc,
		AgentAuth:      authSvc,
		AgentRepo:      agentRepo,
		Notifier:       notifier,
		AuditService:   auditSvc,
		StatusPageRepo: statusPageRepo,
		DB:             db,
		Templates:      templates,
		Logger:         logger,
	})
	if err := reg.InitAll(context.Background()); err != nil {
		logger.Error("failed to initialize modules", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize WebSocket hub
	hub := realtime.NewHub(logger)
	go hub.Run()

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Global middleware
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

	// Initialize router with all dependencies
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
		Logger:           logger,
		SessionSecret:    cfg.Crypto.SessionSecret,
		TemplatesDir:     "web/templates",
		SecureCookies:    cfg.Server.SecureCookies,
		AllowedOrigins:   cfg.Server.AllowedOrigins,
	})
	if err != nil {
		logger.Error("failed to initialize router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Register routes
	router.RegisterRoutes()

	// Start server
	addr := cfg.Server.Address()
	go func() {
		logger.Info("starting server", slog.String("address", addr))
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	fmt.Printf("\n🐕 WatchDog Hub running on http://localhost:%d\n\n", cfg.Server.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hub.Stop()
	router.Stop()
	if err := reg.ShutdownAll(ctx); err != nil {
		logger.Error("module shutdown error", slog.String("error", err.Error()))
	}

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}
}
