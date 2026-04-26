package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sylvester-francis/watchdog/core/ports"
)

// LogRetentionSettingKey is the system_settings row that holds the
// retention window in days. Seeded by migration 101 to "7".
const LogRetentionSettingKey = "log_retention_days"

// DefaultLogRetentionDays is the fallback when the setting is missing,
// unparseable, or non-positive — a 0-day retention would wipe live data,
// so we hard-clamp to the safe default in those cases.
const DefaultLogRetentionDays = 7

// LogRetentionTickInterval is how often the worker rechecks the setting
// and prunes. Hourly matches trace retention.
const LogRetentionTickInterval = 1 * time.Hour

// LogRetention prunes old log records according to system_settings.log_retention_days.
// The retention is enforced application-side via LogRecordRepository.DeleteOlderThan;
// this lets admins change the window at runtime without a redeploy.
type LogRetention struct {
	logs     ports.LogRecordRepository
	settings ports.SystemSettingsRepository
	logger   *slog.Logger
}

// NewLogRetention builds a LogRetention worker.
func NewLogRetention(logs ports.LogRecordRepository, settings ports.SystemSettingsRepository, logger *slog.Logger) *LogRetention {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogRetention{logs: logs, settings: settings, logger: logger}
}

// Start launches the periodic loop and returns immediately. The loop
// exits when ctx is cancelled.
func (r *LogRetention) Start(ctx context.Context) {
	go r.loop(ctx)
}

func (r *LogRetention) loop(ctx context.Context) {
	if err := r.RunOnce(ctx, time.Now()); err != nil {
		r.logger.Error("log retention initial run failed", slog.String("error", err.Error()))
	}
	t := time.NewTicker(LogRetentionTickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.RunOnce(ctx, time.Now()); err != nil {
				r.logger.Error("log retention run failed", slog.String("error", err.Error()))
			}
		}
	}
}

// RunOnce executes a single prune cycle using the supplied wall time
// as the reference point. Exposed (rather than just Start) so tests can
// drive it deterministically without spinning a goroutine.
func (r *LogRetention) RunOnce(ctx context.Context, now time.Time) error {
	days := r.readRetentionDays(ctx)
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	if err := r.logs.DeleteOlderThan(ctx, cutoff); err != nil {
		return fmt.Errorf("delete log records older than %s: %w", cutoff.Format(time.RFC3339), err)
	}
	r.logger.Info("log retention run",
		slog.Int("days", days),
		slog.Time("cutoff", cutoff),
	)
	return nil
}

func (r *LogRetention) readRetentionDays(ctx context.Context) int {
	raw, err := r.settings.Get(ctx, LogRetentionSettingKey)
	if err != nil {
		r.logger.Debug("log retention: using default days",
			slog.String("reason", err.Error()),
		)
		return DefaultLogRetentionDays
	}

	var days int
	if err := json.Unmarshal(raw, &days); err != nil {
		r.logger.Warn("log retention: setting is not a JSON number, using default",
			slog.String("raw", string(raw)),
		)
		return DefaultLogRetentionDays
	}
	if days <= 0 {
		r.logger.Warn("log retention: non-positive days rejected, using default",
			slog.Int("got", days),
		)
		return DefaultLogRetentionDays
	}
	return days
}
