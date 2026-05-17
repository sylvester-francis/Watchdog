package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// heartbeatRangeReader is the minimal subset of HeartbeatRepository the
// service needs. Declared locally (not a port) to keep the surface tight.
type heartbeatRangeReader interface {
	GetByMonitorIDInRange(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Heartbeat, error)
}

// AnomalyService computes latency anomalies on demand from heartbeats. It is
// stateless: each call queries the repo, runs the detector, returns events.
type AnomalyService struct {
	heartbeats heartbeatRangeReader
}

// NewAnomalyService constructs the service.
func NewAnomalyService(heartbeats heartbeatRangeReader) *AnomalyService {
	return &AnomalyService{heartbeats: heartbeats}
}

// RecentAnomalies returns latency anomalies for the monitor over the given
// window (caller picks; default UI uses 24h). Empty slice is a valid result.
func (s *AnomalyService) RecentAnomalies(ctx context.Context, monitorID uuid.UUID, window time.Duration) ([]domain.LatencyAnomaly, error) {
	to := time.Now()
	from := to.Add(-window)
	hbs, err := s.heartbeats.GetByMonitorIDInRange(ctx, monitorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("anomaly service: %w", err)
	}
	return domain.DetectLatencyAnomalies(hbs), nil
}

// CountSince returns just the count, intended for the monitor-card badge.
// Caller typically passes a short window (e.g. 1h).
func (s *AnomalyService) CountSince(ctx context.Context, monitorID uuid.UUID, window time.Duration) (int, error) {
	out, err := s.RecentAnomalies(ctx, monitorID, window)
	if err != nil {
		return 0, err
	}
	return len(out), nil
}
