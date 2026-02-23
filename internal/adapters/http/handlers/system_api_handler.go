package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
	"github.com/sylvester-francis/watchdog/internal/config"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// SystemAPIHandler serves the system dashboard data as JSON (admin-only).
type SystemAPIHandler struct {
	db           *repository.DB
	hub          *realtime.Hub
	cfg          *config.Config
	auditLogRepo ports.AuditLogRepository
	userRepo     ports.UserRepository
	startTime    time.Time
}

// NewSystemAPIHandler creates a new SystemAPIHandler.
func NewSystemAPIHandler(
	db *repository.DB,
	hub *realtime.Hub,
	cfg *config.Config,
	auditLogRepo ports.AuditLogRepository,
	userRepo ports.UserRepository,
	startTime time.Time,
) *SystemAPIHandler {
	return &SystemAPIHandler{
		db:           db,
		hub:          hub,
		cfg:          cfg,
		auditLogRepo: auditLogRepo,
		userRepo:     userRepo,
		startTime:    startTime,
	}
}

// dbResponse holds database health and pool information.
type dbResponse struct {
	Healthy    bool              `json:"healthy"`
	PingMs     float64           `json:"ping_ms"`
	Pool       poolResponse      `json:"pool"`
	Size       string            `json:"size"`
	TableSizes []tableSizeEntry  `json:"table_sizes"`
	Migration  migrationResponse `json:"migration"`
}

// poolResponse holds connection pool statistics.
type poolResponse struct {
	Acquired int32 `json:"acquired"`
	Idle     int32 `json:"idle"`
	Total    int32 `json:"total"`
	Max      int32 `json:"max"`
}

// tableSizeEntry holds a table name and its formatted disk size.
type tableSizeEntry struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

// migrationResponse holds schema migration status.
type migrationResponse struct {
	Version int  `json:"version"`
	Dirty   bool `json:"dirty"`
}

// runtimeResponse holds Go runtime statistics.
type runtimeResponse struct {
	UptimeSeconds   int64   `json:"uptime_seconds"`
	UptimeFormatted string  `json:"uptime_formatted"`
	Goroutines      int     `json:"goroutines"`
	HeapMB          float64 `json:"heap_mb"`
	StackMB         float64 `json:"stack_mb"`
	GCPauseMs       float64 `json:"gc_pause_ms"`
}

// heartbeatResponse holds heartbeat throughput metrics.
type heartbeatResponse struct {
	TotalLastHour  int64   `json:"total_last_hour"`
	PerMinute      float64 `json:"per_minute"`
	ErrorsLastHour int64   `json:"errors_last_hour"`
}

// auditLogEntry holds a single audit log entry for JSON output.
type auditLogEntry struct {
	ID        uuid.UUID         `json:"id"`
	Action    string            `json:"action"`
	UserEmail string            `json:"user_email"`
	IPAddress string            `json:"ip_address"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt string            `json:"created_at"`
}

// systemInfoResponse is the top-level JSON response for GET /api/v1/system.
type systemInfoResponse struct {
	DB              dbResponse      `json:"db"`
	Runtime         runtimeResponse `json:"runtime"`
	AgentsConnected int             `json:"agents_connected"`
	Heartbeats      heartbeatResponse `json:"heartbeats"`
	AuditLogs       []auditLogEntry `json:"audit_logs"`
}

// GetSystemInfo returns system dashboard data as JSON.
// GET /api/v1/system (admin-only)
func (h *SystemAPIHandler) GetSystemInfo(c echo.Context) error {
	ctx := c.Request().Context()

	// Admin check
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil || !user.IsAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "admin access required"})
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

	// Migration status
	var migVersion int
	var migDirty bool
	row := h.db.Pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1")
	if err := row.Scan(&migVersion, &migDirty); err != nil {
		slog.Error("system_api: failed to fetch migration status", "error", err)
	}

	// Heartbeat throughput: count in last hour + per-minute rate
	var hbLastHour int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM heartbeats WHERE time > NOW() - INTERVAL '1 hour'").Scan(&hbLastHour)
	hbPerMinute := float64(hbLastHour) / 60.0

	// Errors in last hour
	var errorsLastHour int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM heartbeats WHERE time > NOW() - INTERVAL '1 hour' AND status != 'up'").Scan(&errorsLastHour)

	// Database size
	var dbSizeBytes int64
	_ = h.db.Pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&dbSizeBytes)

	// Top tables by size
	tableSizes := make([]tableSizeEntry, 0)
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
				tableSizes = append(tableSizes, tableSizeEntry{Name: name, Size: formatBytes(bytes)})
			}
		}
	}

	// Runtime stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	goroutines := runtime.NumGoroutine()

	// Uptime
	uptime := time.Since(h.startTime)

	// Connected agents
	agentCount := h.hub.ClientCount()

	// Audit logs: recent 50 entries (all users)
	logs, err := h.auditLogRepo.GetRecent(ctx, 50)
	if err != nil {
		slog.Error("system_api: failed to fetch audit logs", "error", err)
		logs = nil
	}

	// Resolve user emails with cache to avoid N+1
	emailCache := make(map[uuid.UUID]string)
	for _, log := range logs {
		if log.UserID != nil {
			if _, ok := emailCache[*log.UserID]; !ok {
				if u, err := h.userRepo.GetByID(ctx, *log.UserID); err == nil && u != nil {
					emailCache[*log.UserID] = u.Email
				}
			}
		}
	}

	// Build audit log entries
	auditEntries := make([]auditLogEntry, 0, len(logs))
	for _, log := range logs {
		entry := auditLogEntry{
			ID:        log.ID,
			Action:    string(log.Action),
			IPAddress: log.IPAddress,
			Metadata:  log.Metadata,
			CreatedAt: log.CreatedAt.Format(time.RFC3339),
		}
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]string)
		}
		if log.UserID != nil {
			entry.UserEmail = emailCache[*log.UserID]
		}
		auditEntries = append(auditEntries, entry)
	}

	resp := systemInfoResponse{
		DB: dbResponse{
			Healthy: dbHealthy,
			PingMs:  float64(pingLatency.Microseconds()) / 1000.0,
			Pool: poolResponse{
				Acquired: poolStats.AcquiredConns(),
				Idle:     poolStats.IdleConns(),
				Total:    poolStats.TotalConns(),
				Max:      h.db.Pool.Config().MaxConns,
			},
			Size:       formatBytes(dbSizeBytes),
			TableSizes: tableSizes,
			Migration: migrationResponse{
				Version: migVersion,
				Dirty:   migDirty,
			},
		},
		Runtime: runtimeResponse{
			UptimeSeconds:   int64(uptime.Seconds()),
			UptimeFormatted: formatUptime(uptime),
			Goroutines:      goroutines,
			HeapMB:          roundTo(float64(memStats.HeapAlloc)/(1024*1024), 1),
			StackMB:         roundTo(float64(memStats.StackInuse)/(1024*1024), 1),
			GCPauseMs:       roundTo(float64(memStats.PauseNs[(memStats.NumGC+255)%256])/1e6, 2),
		},
		AgentsConnected: agentCount,
		Heartbeats: heartbeatResponse{
			TotalLastHour:  hbLastHour,
			PerMinute:      roundTo(hbPerMinute, 1),
			ErrorsLastHour: errorsLastHour,
		},
		AuditLogs: auditEntries,
	}

	return c.JSON(http.StatusOK, resp)
}

// roundTo rounds a float to the given number of decimal places.
func roundTo(val float64, decimals int) float64 {
	format := fmt.Sprintf("%%.%df", decimals)
	var result float64
	fmt.Sscanf(fmt.Sprintf(format, val), "%f", &result)
	return result
}
