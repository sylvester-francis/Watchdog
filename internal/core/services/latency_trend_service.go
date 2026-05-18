// Package services holds the application services. LatencyTrendService is
// the one this file adds — pulls bucketed p50/p95/p99 plus a current-vs-
// previous period summary so the frontend can render a chart and a delta
// callout without doing math.
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type latencyTrendReader interface {
	GetLatencyPercentiles(ctx context.Context, monitorID uuid.UUID, from, to time.Time, bucketInterval string) ([]domain.LatencyPercentilePoint, error)
	GetLatencyPercentileSummary(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (domain.LatencyTrendSummary, error)
}

type LatencyTrendService struct {
	heartbeats latencyTrendReader
}

func NewLatencyTrendService(heartbeats latencyTrendReader) *LatencyTrendService {
	return &LatencyTrendService{heartbeats: heartbeats}
}

// GetTrend returns the bucketed latency series for the requested window plus
// the current-period summary and the prior-period summary (for the delta
// callout). Time anchor is `now`, set by the handler for testability.
func (s *LatencyTrendService) GetTrend(ctx context.Context, monitorID uuid.UUID, window domain.TrendWindow, now time.Time) (*domain.LatencyTrend, error) {
	duration := window.Duration()
	bucket := window.BucketIntervalFor()
	currentFrom := now.Add(-duration)
	previousFrom := currentFrom.Add(-duration)

	points, err := s.heartbeats.GetLatencyPercentiles(ctx, monitorID, currentFrom, now, bucket)
	if err != nil {
		return nil, fmt.Errorf("latency trend points: %w", err)
	}
	current, err := s.heartbeats.GetLatencyPercentileSummary(ctx, monitorID, currentFrom, now)
	if err != nil {
		return nil, fmt.Errorf("latency trend current summary: %w", err)
	}
	previous, err := s.heartbeats.GetLatencyPercentileSummary(ctx, monitorID, previousFrom, currentFrom)
	if err != nil {
		return nil, fmt.Errorf("latency trend previous summary: %w", err)
	}

	return &domain.LatencyTrend{
		WindowSeconds:  int(duration.Seconds()),
		BucketInterval: bucket,
		Points:         points,
		Current:        current,
		Previous:       previous,
	}, nil
}
