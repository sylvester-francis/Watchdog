package metrics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const (
	historySize      = 360 // 1 hour at 10s intervals
	snapshotInterval = 10 * time.Second
)

// MetricsSnapshot holds a point-in-time snapshot of key hub metrics.
type MetricsSnapshot struct {
	Timestamp       int64   `json:"timestamp"`
	WSConnections   float64 `json:"ws_connections"`
	DBPoolActive    float64 `json:"db_pool_active"`
	IncidentsOpen   float64 `json:"incidents_open"`
	IncidentsAcked  float64 `json:"incidents_acked"`
	HTTPLatencyP50  float64 `json:"http_latency_p50"`
	HTTPLatencyP95  float64 `json:"http_latency_p95"`
	HTTPLatencyP99  float64 `json:"http_latency_p99"`
	HeartbeatP50    float64 `json:"heartbeat_p50"`
	HeartbeatP95    float64 `json:"heartbeat_p95"`
	HTTPRequestRate float64 `json:"http_request_rate"`
}

// MetricsHistory maintains a ring buffer of metric snapshots.
type MetricsHistory struct {
	mu            sync.RWMutex
	buf           [historySize]MetricsSnapshot
	pos           int
	count         int
	prevHTTPCount float64
	prevTime      time.Time
}

// NewMetricsHistory creates a new MetricsHistory.
func NewMetricsHistory() *MetricsHistory {
	return &MetricsHistory{}
}

// Start begins the background snapshot loop. It stops when ctx is cancelled.
func (h *MetricsHistory) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(snapshotInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.snapshot()
			}
		}
	}()
}

// Current returns the most recent snapshot, or a zero snapshot if none.
func (h *MetricsHistory) Current() MetricsSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.count == 0 {
		return MetricsSnapshot{Timestamp: time.Now().Unix()}
	}
	idx := (h.pos - 1 + historySize) % historySize
	return h.buf[idx]
}

// History returns all snapshots in chronological order.
func (h *MetricsHistory) History() []MetricsSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.count == 0 {
		return nil
	}
	result := make([]MetricsSnapshot, h.count)
	start := 0
	if h.count == historySize {
		start = h.pos
	}
	for i := 0; i < h.count; i++ {
		result[i] = h.buf[(start+i)%historySize]
	}
	return result
}

func (h *MetricsHistory) snapshot() {
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return
	}

	now := time.Now()
	snap := MetricsSnapshot{Timestamp: now.Unix()}

	var totalHTTPCount float64

	for _, fam := range families {
		name := fam.GetName()
		switch name {
		case "watchdog_ws_connections_active":
			snap.WSConnections = gaugeValue(fam)
		case "watchdog_db_pool_active_connections":
			snap.DBPoolActive = gaugeValue(fam)
		case "watchdog_incidents_active":
			for _, m := range fam.GetMetric() {
				for _, lp := range m.GetLabel() {
					if lp.GetName() == "status" {
						switch lp.GetValue() {
						case "open":
							snap.IncidentsOpen = m.GetGauge().GetValue()
						case "acknowledged":
							snap.IncidentsAcked = m.GetGauge().GetValue()
						}
					}
				}
			}
		case "watchdog_http_request_duration_seconds":
			snap.HTTPLatencyP50 = histogramQuantile(fam, 0.50)
			snap.HTTPLatencyP95 = histogramQuantile(fam, 0.95)
			snap.HTTPLatencyP99 = histogramQuantile(fam, 0.99)
			totalHTTPCount = histogramTotalCount(fam)
		case "watchdog_heartbeat_processing_seconds":
			snap.HeartbeatP50 = histogramQuantile(fam, 0.50)
			snap.HeartbeatP95 = histogramQuantile(fam, 0.95)
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.prevTime.IsZero() {
		dt := now.Sub(h.prevTime).Seconds()
		if dt > 0 {
			snap.HTTPRequestRate = roundN((totalHTTPCount-h.prevHTTPCount)/dt, 1)
		}
	}
	h.prevHTTPCount = totalHTTPCount
	h.prevTime = now

	snap.HTTPLatencyP50 = roundN(snap.HTTPLatencyP50*1000, 2)
	snap.HTTPLatencyP95 = roundN(snap.HTTPLatencyP95*1000, 2)
	snap.HTTPLatencyP99 = roundN(snap.HTTPLatencyP99*1000, 2)
	snap.HeartbeatP50 = roundN(snap.HeartbeatP50*1000, 2)
	snap.HeartbeatP95 = roundN(snap.HeartbeatP95*1000, 2)

	h.buf[h.pos] = snap
	h.pos = (h.pos + 1) % historySize
	if h.count < historySize {
		h.count++
	}
}

func gaugeValue(fam *dto.MetricFamily) float64 {
	metrics := fam.GetMetric()
	if len(metrics) == 0 {
		return 0
	}
	return metrics[0].GetGauge().GetValue()
}

type histBucket struct {
	upperBound float64
	cumCount   float64
}

func histogramQuantile(fam *dto.MetricFamily, q float64) float64 {
	bucketMap := make(map[float64]float64)
	var totalCount float64

	for _, m := range fam.GetMetric() {
		h := m.GetHistogram()
		if h == nil {
			continue
		}
		totalCount += float64(h.GetSampleCount())
		for _, b := range h.GetBucket() {
			bucketMap[b.GetUpperBound()] += float64(b.GetCumulativeCount())
		}
	}

	if totalCount == 0 {
		return 0
	}

	buckets := make([]histBucket, 0, len(bucketMap))
	for ub, cc := range bucketMap {
		buckets = append(buckets, histBucket{upperBound: ub, cumCount: cc})
	}
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].upperBound < buckets[j].upperBound
	})

	rank := q * totalCount
	var prevBound, prevCount float64

	for _, b := range buckets {
		if math.IsInf(b.upperBound, 1) {
			break
		}
		if b.cumCount >= rank {
			bucketCount := b.cumCount - prevCount
			if bucketCount == 0 {
				return prevBound
			}
			fraction := (rank - prevCount) / bucketCount
			return prevBound + fraction*(b.upperBound-prevBound)
		}
		prevBound = b.upperBound
		prevCount = b.cumCount
	}

	for i := len(buckets) - 1; i >= 0; i-- {
		if !math.IsInf(buckets[i].upperBound, 1) {
			return buckets[i].upperBound
		}
	}
	return 0
}

func histogramTotalCount(fam *dto.MetricFamily) float64 {
	var total float64
	for _, m := range fam.GetMetric() {
		h := m.GetHistogram()
		if h != nil {
			total += float64(h.GetSampleCount())
		}
	}
	return total
}

func roundN(val float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}
