package domain

import "time"

// LatencyPercentilePoint is one time-bucket of latency stats. p50/p95/p99 are
// percentile_cont aggregates over successful checks (status=up) in the bucket.
type LatencyPercentilePoint struct {
	Time        time.Time `json:"time"`
	P50         int       `json:"p50"`
	P95         int       `json:"p95"`
	P99         int       `json:"p99"`
	SampleCount int       `json:"sample_count"`
}

// LatencyTrendSummary is the aggregate over an entire period — used to render
// the "current vs previous period" delta callout above the chart.
type LatencyTrendSummary struct {
	P50         int `json:"p50"`
	P95         int `json:"p95"`
	P99         int `json:"p99"`
	SampleCount int `json:"sample_count"`
}

// LatencyTrend is the full payload the frontend renders: time-series points
// for the chart, plus the period summaries (current + previous) used to
// compute the "p95 +26%" callout. Zero SampleCount means no data — UI hides
// the delta and shows an empty-state.
type LatencyTrend struct {
	WindowSeconds int                      `json:"window_seconds"`
	BucketInterval string                  `json:"bucket_interval"`
	Points         []LatencyPercentilePoint `json:"points"`
	Current        LatencyTrendSummary     `json:"current"`
	Previous       LatencyTrendSummary     `json:"previous"`
}

// TrendWindow names the supported look-back windows. Tied to bucket sizing
// to keep the rendered chart at ~100-200 points.
type TrendWindow string

const (
	TrendWindow7d  TrendWindow = "7d"
	TrendWindow30d TrendWindow = "30d"
	TrendWindow90d TrendWindow = "90d"
)

// BucketIntervalFor returns the time_bucket() interval string to use with this
// window. Sized so each window renders ~100-200 buckets.
func (w TrendWindow) BucketIntervalFor() string {
	switch w {
	case TrendWindow30d:
		return "6 hours"
	case TrendWindow90d:
		return "1 day"
	default:
		return "1 hour"
	}
}

// Duration returns the look-back window as a time.Duration.
func (w TrendWindow) Duration() time.Duration {
	switch w {
	case TrendWindow30d:
		return 30 * 24 * time.Hour
	case TrendWindow90d:
		return 90 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

// ParseTrendWindow accepts user input from the query string. Unknown values
// fall back to the default 7d window.
func ParseTrendWindow(raw string) TrendWindow {
	switch TrendWindow(raw) {
	case TrendWindow7d, TrendWindow30d, TrendWindow90d:
		return TrendWindow(raw)
	default:
		return TrendWindow7d
	}
}
