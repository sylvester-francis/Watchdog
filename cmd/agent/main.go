package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Flags
	hubURL := flag.String("hub", "ws://localhost:8080/ws/agent", "Hub WebSocket URL")
	apiKey := flag.String("api-key", "", "Agent API key")
	version := flag.Bool("version", false, "Print version and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *version {
		fmt.Printf("WatchDog Agent %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Setup logger
	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// API key from flag or environment
	key := *apiKey
	if key == "" {
		key = os.Getenv("WATCHDOG_API_KEY")
	}
	if key == "" {
		logger.Error("API key required. Use -api-key flag or WATCHDOG_API_KEY env var")
		os.Exit(1)
	}

	logger.Info("WatchDog Agent starting",
		slog.String("version", Version),
		slog.String("hub_url", *hubURL),
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start agent
	agent := NewAgent(AgentConfig{
		HubURL:  *hubURL,
		APIKey:  key,
		Version: Version,
		Logger:  logger,
	})

	// Start agent in background
	go func() {
		if err := agent.Run(ctx); err != nil {
			logger.Error("agent error", slog.String("error", err.Error()))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down agent...")

	// Give agent time to clean up
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	cancel() // Cancel the main context

	// Wait for shutdown or timeout
	select {
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timed out")
	case <-time.After(2 * time.Second):
		logger.Info("agent stopped gracefully")
	}
}

// AgentConfig holds agent configuration.
type AgentConfig struct {
	HubURL  string
	APIKey  string
	Version string
	Logger  *slog.Logger
}

// Agent represents the monitoring agent.
type Agent struct {
	config AgentConfig
	conn   *Connection
	tasks  map[string]*Task
	logger *slog.Logger
	stopCh chan struct{}
}

// NewAgent creates a new agent instance.
func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		config: config,
		tasks:  make(map[string]*Task),
		logger: config.Logger,
		stopCh: make(chan struct{}),
	}
}

// Run starts the agent and maintains connection to the hub.
func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			a.stopAllTasks()
			return ctx.Err()
		default:
			if err := a.connectAndRun(ctx); err != nil {
				a.logger.Error("connection error", slog.String("error", err.Error()))
			}

			// Wait before reconnecting
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(a.getReconnectDelay()):
				a.logger.Info("reconnecting to hub...")
			}
		}
	}
}

func (a *Agent) connectAndRun(ctx context.Context) error {
	conn, err := NewConnection(a.config.HubURL, a.config.APIKey, a.config.Version, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}
	a.conn = conn

	// Authenticate
	if err := conn.Authenticate(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("authentication failed: %w", err)
	}

	a.logger.Info("connected and authenticated")

	// Run message loop
	return conn.Run(ctx, a.handleMessage)
}

func (a *Agent) handleMessage(msg *Message) {
	switch msg.Type {
	case MsgTypeTask:
		a.handleTask(msg)
	case MsgTypePing:
		if err := a.conn.SendPong(); err != nil {
			a.logger.Error("failed to send pong", slog.String("error", err.Error()))
		}
	default:
		a.logger.Debug("received message", slog.String("type", msg.Type))
	}
}

func (a *Agent) handleTask(msg *Message) {
	var payload TaskPayload
	if err := msg.ParsePayload(&payload); err != nil {
		a.logger.Error("failed to parse task", slog.String("error", err.Error()))
		return
	}

	// Stop existing task for this monitor if running
	if existing, ok := a.tasks[payload.MonitorID]; ok {
		existing.Stop()
	}

	// Create and start new task
	task := NewTask(payload, a.conn, a.logger)
	a.tasks[payload.MonitorID] = task
	go task.Run()

	a.logger.Info("task started",
		slog.String("monitor_id", payload.MonitorID),
		slog.String("type", payload.Type),
		slog.String("target", payload.Target),
	)
}

func (a *Agent) stopAllTasks() {
	for _, task := range a.tasks {
		task.Stop()
	}
	a.tasks = make(map[string]*Task)
}

func (a *Agent) getReconnectDelay() time.Duration {
	// Simple fixed delay; could implement exponential backoff
	return 5 * time.Second
}
