package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
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

// tableSizeInfo holds a table name and its formatted disk size.
type tableSizeInfo struct {
	Name string
	Size string
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

	// --- Operational Metrics ---

	// Heartbeat throughput: count in last hour + per-minute rate
	var hbLastHour int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM heartbeats WHERE time > NOW() - INTERVAL '1 hour'").Scan(&hbLastHour)
	hbPerMinute := float64(hbLastHour) / 60.0

	// Database size
	var dbSizeBytes int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&dbSizeBytes)

	// Top tables by size
	tableSizes := make([]tableSizeInfo, 0)
	tblRows, err := h.db.Pool.Query(ctx, `
		SELECT relname, pg_total_relation_size(quote_ident(relname))
		FROM pg_class
		WHERE relkind = 'r' AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		ORDER BY pg_total_relation_size(quote_ident(relname)) DESC
		LIMIT 5`)
	if err == nil {
		defer tblRows.Close()
		for tblRows.Next() {
			var name string
			var bytes int64
			if tblRows.Scan(&name, &bytes) == nil {
				tableSizes = append(tableSizes, tableSizeInfo{Name: name, Size: formatBytes(bytes)})
			}
		}
	}

	// Recent errors: heartbeats with status != 'up' in last hour
	var errorsLastHour int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM heartbeats WHERE time > NOW() - INTERVAL '1 hour' AND status != 'up'").Scan(&errorsLastHour)

	// Runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	goroutines := runtime.NumGoroutine()

	return c.Render(http.StatusOK, "system.html", map[string]interface{}{
		"Title":          "System",
		"DBHealthy":      dbHealthy,
		"PingLatency":    fmt.Sprintf("%.0fms", float64(pingLatency.Microseconds())/1000.0),
		"PoolAcquired":   poolStats.AcquiredConns(),
		"PoolIdle":       poolStats.IdleConns(),
		"PoolTotal":      poolStats.TotalConns(),
		"AgentCount":     agentCount,
		"Uptime":         formatUptime(uptime),
		"Migration":      migration,
		"HBPerMinute":    fmt.Sprintf("%.1f", hbPerMinute),
		"HBLastHour":     hbLastHour,
		"DBSize":         formatBytes(dbSizeBytes),
		"TableSizes":     tableSizes,
		"ErrorsLastHour": errorsLastHour,
		"Goroutines":     goroutines,
		"HeapMB":         fmt.Sprintf("%.1f", float64(memStats.HeapAlloc)/(1024*1024)),
		"StackMB":        fmt.Sprintf("%.1f", float64(memStats.StackInuse)/(1024*1024)),
		"GCPauseMs":      fmt.Sprintf("%.2f", float64(memStats.PauseNs[(memStats.NumGC+255)%256])/1e6),
		"AuditLogs":      auditViews,
	})
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
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
