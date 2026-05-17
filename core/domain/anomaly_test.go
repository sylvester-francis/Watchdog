package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func intPtr(n int) *int { return &n }

func TestMean(t *testing.T) {
	assert.InDelta(t, 0.0, mean(nil), 1e-9)
	assert.InDelta(t, 5.0, mean([]float64{5}), 1e-9)
	assert.InDelta(t, 3.0, mean([]float64{1, 2, 3, 4, 5}), 1e-9)
}

func TestStdev(t *testing.T) {
	// Population stdev of {1,2,3,4,5} = sqrt(2) ≈ 1.4142
	assert.InDelta(t, 1.4142, stdev([]float64{1, 2, 3, 4, 5}), 1e-3)
	assert.InDelta(t, 0.0, stdev([]float64{42}), 1e-9, "single value: stdev is 0")
	assert.InDelta(t, 0.0, stdev(nil), 1e-9)
}

func TestPercentile(t *testing.T) {
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.InDelta(t, 3.25, percentile(xs, 25), 1e-9)
	assert.InDelta(t, 5.5, percentile(xs, 50), 1e-9)
	assert.InDelta(t, 7.75, percentile(xs, 75), 1e-9)
}

func TestDetectLatencyAnomalies_SkipsWhenInsufficient(t *testing.T) {
	hbs := make([]*Heartbeat, 29)
	for i := range hbs {
		hbs[i] = &Heartbeat{Status: HeartbeatStatusUp, LatencyMs: intPtr(100), Time: time.Now()}
	}
	got := DetectLatencyAnomalies(hbs)
	assert.Empty(t, got, "fewer than 30 samples → no anomalies returned")
}

func TestDetectLatencyAnomalies_FlagsZScoreOutlier(t *testing.T) {
	hbs := make([]*Heartbeat, 100)
	mID := uuid.New()
	for i := range hbs {
		hbs[i] = &Heartbeat{
			MonitorID: mID,
			Status:    HeartbeatStatusUp,
			LatencyMs: intPtr(100),
			Time:      time.Now().Add(-time.Duration(100-i) * time.Minute),
		}
	}
	hbs[50].LatencyMs = intPtr(10000) // huge spike

	got := DetectLatencyAnomalies(hbs)
	assert.Len(t, got, 1)
	assert.Equal(t, mID, got[0].MonitorID)
	assert.Equal(t, 10000, got[0].LatencyMs)
	assert.Contains(t, []string{"zscore", "iqr", "both"}, got[0].Method)
}

func TestDetectLatencyAnomalies_FlagsIQROutlier(t *testing.T) {
	// Skewed distribution: 50 samples at 100ms, 50 at 200ms, one at 1000ms.
	// IQR-based detection should flag the 1000ms point.
	hbs := make([]*Heartbeat, 101)
	for i := 0; i < 50; i++ {
		hbs[i] = &Heartbeat{
			Status:    HeartbeatStatusUp,
			LatencyMs: intPtr(100),
			Time:      time.Now().Add(-time.Duration(101-i) * time.Minute),
		}
	}
	for i := 50; i < 100; i++ {
		hbs[i] = &Heartbeat{
			Status:    HeartbeatStatusUp,
			LatencyMs: intPtr(200),
			Time:      time.Now().Add(-time.Duration(101-i) * time.Minute),
		}
	}
	hbs[100] = &Heartbeat{Status: HeartbeatStatusUp, LatencyMs: intPtr(1000), Time: time.Now()}

	got := DetectLatencyAnomalies(hbs)
	assert.NotEmpty(t, got)
	// The 1000ms point must be among the anomalies (there may be a few more
	// near the edges of the bimodal distribution, but the spike is certain).
	foundSpike := false
	for _, a := range got {
		if a.LatencyMs == 1000 {
			foundSpike = true
		}
	}
	assert.True(t, foundSpike, "1000ms spike must be flagged")
}

func TestDetectLatencyAnomalies_IgnoresDownHeartbeats(t *testing.T) {
	hbs := make([]*Heartbeat, 50)
	for i := range hbs {
		hbs[i] = &Heartbeat{Status: HeartbeatStatusDown, LatencyMs: intPtr(10000), Time: time.Now()}
	}
	got := DetectLatencyAnomalies(hbs)
	assert.Empty(t, got, "down heartbeats are skipped — latency is meaningless")
}

func TestDetectLatencyAnomalies_IgnoresNilLatency(t *testing.T) {
	hbs := make([]*Heartbeat, 50)
	for i := range hbs {
		hbs[i] = &Heartbeat{Status: HeartbeatStatusUp, LatencyMs: nil, Time: time.Now()}
	}
	got := DetectLatencyAnomalies(hbs)
	assert.Empty(t, got, "nil-latency heartbeats are skipped")
}
