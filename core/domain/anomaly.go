package domain

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// LatencyAnomalyConfig holds detection thresholds. Defaults are not user-tunable in v1.
type LatencyAnomalyConfig struct {
	MinSamples      int
	ZScoreThreshold float64
	IQRMultiplier   float64
}

// DefaultLatencyAnomalyConfig returns the v1 defaults: 24h window assumed by
// the caller, Z-score >= 3 (≈ 99.7% range), IQR 1.5× (classic Tukey).
func DefaultLatencyAnomalyConfig() LatencyAnomalyConfig {
	return LatencyAnomalyConfig{
		MinSamples:      30,
		ZScoreThreshold: 3.0,
		IQRMultiplier:   1.5,
	}
}

// LatencyAnomaly is a single anomalous heartbeat detected post-hoc.
type LatencyAnomaly struct {
	MonitorID uuid.UUID `json:"monitor_id"`
	AgentID   uuid.UUID `json:"agent_id"`
	Time      time.Time `json:"time"`
	LatencyMs int       `json:"latency_ms"`
	ZScore    float64   `json:"z_score"`
	Method    string    `json:"method"` // "zscore", "iqr", or "both"
}

// DetectLatencyAnomalies returns heartbeats that fail either the Z-score test
// or the IQR/Tukey-fence test, using default config. Only HeartbeatStatusUp
// heartbeats with non-nil positive latency contribute to the statistics and
// candidate set. Order of returned anomalies is unspecified.
func DetectLatencyAnomalies(hbs []*Heartbeat) []LatencyAnomaly {
	return DetectLatencyAnomaliesWithConfig(hbs, DefaultLatencyAnomalyConfig())
}

// DetectLatencyAnomaliesWithConfig is the configurable variant. Exposed for
// tests; not currently wired to user config.
func DetectLatencyAnomaliesWithConfig(hbs []*Heartbeat, cfg LatencyAnomalyConfig) []LatencyAnomaly {
	type sample struct {
		hb      *Heartbeat
		latency float64
	}
	samples := make([]sample, 0, len(hbs))
	for _, h := range hbs {
		if h == nil || h.Status != HeartbeatStatusUp || h.LatencyMs == nil || *h.LatencyMs <= 0 {
			continue
		}
		samples = append(samples, sample{hb: h, latency: float64(*h.LatencyMs)})
	}
	if len(samples) < cfg.MinSamples {
		return nil
	}

	latencies := make([]float64, len(samples))
	for i, s := range samples {
		latencies[i] = s.latency
	}

	m := mean(latencies)
	sd := stdev(latencies)
	q1 := percentile(latencies, 25)
	q3 := percentile(latencies, 75)
	iqr := q3 - q1
	iqrLo := q1 - cfg.IQRMultiplier*iqr
	iqrHi := q3 + cfg.IQRMultiplier*iqr

	out := make([]LatencyAnomaly, 0)
	for _, s := range samples {
		var z float64
		if sd > 0 {
			z = (s.latency - m) / sd
		}
		zFlagged := math.Abs(z) >= cfg.ZScoreThreshold
		iqrFlagged := s.latency < iqrLo || s.latency > iqrHi
		if !zFlagged && !iqrFlagged {
			continue
		}
		method := "both"
		switch {
		case zFlagged && !iqrFlagged:
			method = "zscore"
		case !zFlagged && iqrFlagged:
			method = "iqr"
		}
		out = append(out, LatencyAnomaly{
			MonitorID: s.hb.MonitorID,
			AgentID:   s.hb.AgentID,
			Time:      s.hb.Time,
			LatencyMs: int(s.latency),
			ZScore:    z,
			Method:    method,
		})
	}
	return out
}

// --- internal helpers ---

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

func stdev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	sq := 0.0
	for _, x := range xs {
		d := x - m
		sq += d * d
	}
	return math.Sqrt(sq / float64(len(xs)))
}

// percentile returns the p-th percentile (0-100) using linear interpolation
// between the two nearest ranks (same default algorithm numpy uses).
func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := (p / 100.0) * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}
