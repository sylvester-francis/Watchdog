package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sylvester-francis/watchdog/core/ports"
)

// TraceRetentionSettingKey is the system_settings row that holds the
// retention window in days. Seeded by migration 100 to "7".
const TraceRetentionSettingKey = "trace_retention_days"

// DefaultTraceRetentionDays is the fallback when the setting is missing,
// unparseable, or non-positive — a 0-day retention would wipe live data,
// so we hard-clamp to the safe default in those cases.
const DefaultTraceRetentionDays = 7

// TraceRetentionTickInterval is how often the worker rechecks the setting
// and prunes. Hourly is plenty for day-granularity retention.
const TraceRetentionTickInterval = 1 * time.Hour

// TraceRetention prunes old spans according to system_settings.trace_retention_days.
// The retention is enforced application-side via SpanRepository.DeleteOlderThan;
// this lets admins change the window at runtime without a redeploy.
type TraceRetention struct {
	spans    ports.SpanRepository
	settings ports.SystemSettingsRepository
	logger   *slog.Logger
}

// NewTraceRetention builds a TraceRetention worker.
func NewTraceRetention(spans ports.SpanRepository, settings ports.SystemSettingsRepository, logger *slog.Logger) *TraceRetention {
	if logger == nil {
		logger = slog.Default()
	}
	return &TraceRetention{spans: spans, settings: settings, logger: logger}
}

// Start launches the periodic loop and returns immediately. The loop
// exits when ctx is cancelled.
func (r *TraceRetention) Start(ctx context.Context) {
	go r.loop(ctx)
}

func (r *TraceRetention) loop(ctx context.Context) {
	if err := r.RunOnce(ctx, time.Now()); err != nil {
		r.logger.Error("trace retention initial run failed", slog.String("error", err.Error()))
	}
	t := time.NewTicker(TraceRetentionTickInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.RunOnce(ctx, time.Now()); err != nil {
				r.logger.Error("trace retention run failed", slog.String("error", err.Error()))
			}
		}
	}
}

// RunOnce executes a single prune cycle using the supplied wall time
// as the reference point. Exposed (rather than just Start) so tests can
// drive it deterministically without spinning a goroutine.
func (r *TraceRetention) RunOnce(ctx context.Context, now time.Time) error {
	days := r.readRetentionDays(ctx)
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	if err := r.spans.DeleteOlderThan(ctx, cutoff); err != nil {
		return fmt.Errorf("delete spans older than %s: %w", cutoff.Format(time.RFC3339), err)
	}
	r.logger.Info("trace retention run",
		slog.Int("days", days),
		slog.Time("cutoff", cutoff),
	)
	return nil
}

func (r *TraceRetention) readRetentionDays(ctx context.Context) int {
	raw, err := r.settings.Get(ctx, TraceRetentionSettingKey)
	if err != nil {
		// Missing or transient repo error — operate on the default.
		// Logged at debug to avoid noise on cold-start before the
		// setting is seeded.
		r.logger.Debug("trace retention: using default days",
			slog.String("reason", err.Error()),
		)
		return DefaultTraceRetentionDays
	}

	var days int
	if err := json.Unmarshal(raw, &days); err != nil {
		r.logger.Warn("trace retention: setting is not a JSON number, using default",
			slog.String("raw", string(raw)),
		)
		return DefaultTraceRetentionDays
	}
	if days <= 0 {
		r.logger.Warn("trace retention: non-positive days rejected, using default",
			slog.Int("got", days),
		)
		return DefaultTraceRetentionDays
	}
	return days
}

// Sentinel for callers that want to distinguish "no setting yet" from
// other errors. Currently unused inside TraceRetention itself but
// exported because admin handlers (PR E) may want to surface it.
var ErrNoRetentionSetting = errors.New("trace retention setting not configured")
