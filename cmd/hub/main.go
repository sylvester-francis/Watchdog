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

	internalhttp "github.com/sylvester/watchdog/internal/adapters/http"
	"github.com/sylvester/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester/watchdog/internal/adapters/notify"
	"github.com/sylvester/watchdog/internal/adapters/repository"
	"github.com/sylvester/watchdog/internal/config"
	"github.com/sylvester/watchdog/internal/core/realtime"
	"github.com/sylvester/watchdog/internal/core/services"
	"github.com/sylvester/watchdog/internal/crypto"
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

	// Initialize notifiers
	notifier := notify.NewNoOpNotifier() // Replace with real notifier in production

	// Initialize services
	authSvc := services.NewAuthService(userRepo, agentRepo, hasher, encryptor)
	incidentSvc := services.NewIncidentService(incidentRepo, monitorRepo, notifier, db, logger)
	monitorSvc := services.NewMonitorService(monitorRepo, heartbeatRepo, incidentRepo, incidentSvc, logger)

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
		LogValuesFunc: func(_ echo.Context, v echomw.RequestLoggerValues) error {
			logger.Info("request",
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
				slog.String("remote_ip", v.RemoteIP),
			)
			return nil
		},
	}))
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(middleware.SecureHeaders())

	// Initialize router with all dependencies
	router, err := internalhttp.NewRouter(e, internalhttp.Dependencies{
		AuthService:     authSvc,
		MonitorService:  monitorSvc,
		IncidentService: incidentSvc,
		UserRepo:        userRepo,
		AgentRepo:       agentRepo,
		MonitorRepo:     monitorRepo,
		Hub:             hub,
		SessionSecret:   cfg.Crypto.SessionSecret,
		TemplatesDir:    "web/templates",
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

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}
}
