package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// AdminHandler handles the system dashboard page.
type AdminHandler struct {
	auditLogRepo ports.AuditLogRepository
	userRepo     ports.UserRepository
	hub          *realtime.Hub
	db           *repository.DB
	cfg          *config.Config
	startTime    time.Time
	templates    *view.Templates
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	auditLogRepo ports.AuditLogRepository,
	userRepo ports.UserRepository,
	hub *realtime.Hub,
	db *repository.DB,
	cfg *config.Config,
	startTime time.Time,
	templates *view.Templates,
) *AdminHandler {
	return &AdminHandler{
		auditLogRepo: auditLogRepo,
		userRepo:     userRepo,
		hub:          hub,
		db:           db,
		cfg:          cfg,
		startTime:    startTime,
		templates:    templates,
	}
}

// migrationInfo holds schema migration status.
type migrationInfo struct {
	Version int
	Dirty   bool
}

// configEntry holds a redacted config key-value for display.
type configEntry struct {
	Key   string
	Value string
}

// configSection groups config entries under a heading.
type configSection struct {
	Name    string
	Entries []configEntry
}

// auditLogView holds an audit log entry enriched with user email.
type auditLogView struct {
	*domain.AuditLog
	Email string
}

// Dashboard renders the system dashboard page.
func (h *AdminHandler) Dashboard(c echo.Context) error {
	ctx := c.Request().Context()

	// Audit log: recent 50 entries
	logs, err := h.auditLogRepo.GetRecent(ctx, 50)
	if err != nil {
		slog.Error("admin: failed to fetch audit logs", "error", err)
		logs = nil
	}

	// Resolve user IDs to emails
	emailMap := make(map[uuid.UUID]string)
	if logs != nil {
		userIDs := make(map[uuid.UUID]bool)
		for _, l := range logs {
			if l.UserID != nil {
				userIDs[*l.UserID] = true
			}
		}
		for uid := range userIDs {
			user, err := h.userRepo.GetByID(ctx, uid)
			if err == nil && user != nil {
				emailMap[uid] = user.Email
			}
		}
	}

	auditViews := make([]auditLogView, 0, len(logs))
	for _, l := range logs {
		v := auditLogView{AuditLog: l}
		if l.UserID != nil {
			v.Email = emailMap[*l.UserID]
		}
		auditViews = append(auditViews, v)
	}

	// DB health + latency
	dbHealthy := true
	pingStart := time.Now()
	if err := h.db.Health(ctx); err != nil {
		dbHealthy = false
	}
	pingLatency := time.Since(pingStart)

	// Pool stats
	poolStats := h.db.Stats()

	// Connected agents
	agentCount := h.hub.ClientCount()

	// Uptime
	uptime := time.Since(h.startTime)

	// Migration status
	migration := migrationInfo{}
	row := h.db.Pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1")
	if err := row.Scan(&migration.Version, &migration.Dirty); err != nil {
		slog.Error("admin: failed to fetch migration status", "error", err)
	}

	// Count total available migrations
	totalMigrations := 0
	migRows, err := h.db.Pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err == nil {
		defer migRows.Close()
		for migRows.Next() {
			totalMigrations++
		}
	}

	// Config overview (redacted)
	configSections := h.buildConfigOverview()

	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"Title":           "System",
		"DBHealthy":       dbHealthy,
		"PingLatency":     fmt.Sprintf("%.0fms", float64(pingLatency.Microseconds())/1000.0),
		"PoolAcquired":    poolStats.AcquiredConns(),
		"PoolIdle":        poolStats.IdleConns(),
		"PoolTotal":       poolStats.TotalConns(),
		"AgentCount":      agentCount,
		"Uptime":          formatUptime(uptime),
		"Migration":       migration,
		"TotalMigrations": totalMigrations,
		"ConfigSections":  configSections,
		"AuditLogs":       auditViews,
	})
}

// buildConfigOverview returns config sections with secrets redacted.
func (h *AdminHandler) buildConfigOverview() []configSection {
	cfg := h.cfg

	server := configSection{
		Name: "Server",
		Entries: []configEntry{
			{Key: "Host", Value: cfg.Server.Host},
			{Key: "Port", Value: fmt.Sprintf("%d", cfg.Server.Port)},
			{Key: "Secure Cookies", Value: fmt.Sprintf("%v", cfg.Server.SecureCookies)},
			{Key: "Allowed Origins", Value: strings.Join(cfg.Server.AllowedOrigins, ", ")},
		},
	}
	if server.Entries[3].Value == "" {
		server.Entries[3].Value = "(default)"
	}

	database := configSection{
		Name: "Database",
		Entries: []configEntry{
			{Key: "URL", Value: "********"},
			{Key: "Max Connections", Value: fmt.Sprintf("%d", cfg.Database.MaxConns)},
			{Key: "Min Connections", Value: fmt.Sprintf("%d", cfg.Database.MinConns)},
			{Key: "Max Conn Lifetime", Value: cfg.Database.MaxConnLifetime.String()},
			{Key: "Max Conn Idle Time", Value: cfg.Database.MaxConnIdleTime.String()},
		},
	}

	notifiers := configSection{
		Name: "Notifiers",
		Entries: []configEntry{
			{Key: "Slack", Value: boolIndicator(cfg.Notify.SlackWebhookURL != "")},
			{Key: "Discord", Value: boolIndicator(cfg.Notify.DiscordWebhookURL != "")},
			{Key: "Webhook", Value: boolIndicator(cfg.Notify.WebhookURL != "")},
			{Key: "Email (SMTP)", Value: boolIndicator(cfg.Notify.SMTPHost != "")},
			{Key: "Telegram", Value: boolIndicator(cfg.Notify.TelegramBotToken != "")},
			{Key: "PagerDuty", Value: boolIndicator(cfg.Notify.PagerDutyRoutingKey != "")},
		},
	}

	security := configSection{
		Name: "Security",
		Entries: []configEntry{
			{Key: "Encryption Key", Value: "********"},
			{Key: "Session Secret", Value: "********"},
		},
	}

	return []configSection{server, database, notifiers, security}
}

// boolIndicator returns a check or x string.
func boolIndicator(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

// formatUptime formats a duration as a human-readable uptime string.
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
