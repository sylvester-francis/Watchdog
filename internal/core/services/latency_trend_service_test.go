package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type fakeTrendRepo struct {
	pointsFn  func(ctx context.Context, monitorID uuid.UUID, from, to time.Time, bucket string) ([]domain.LatencyPercentilePoint, error)
	summaryFn func(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (domain.LatencyTrendSummary, error)
}

func (f *fakeTrendRepo) GetLatencyPercentiles(ctx context.Context, monitorID uuid.UUID, from, to time.Time, bucket string) ([]domain.LatencyPercentilePoint, error) {
	return f.pointsFn(ctx, monitorID, from, to, bucket)
}

func (f *fakeTrendRepo) GetLatencyPercentileSummary(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (domain.LatencyTrendSummary, error) {
	return f.summaryFn(ctx, monitorID, from, to)
}

func TestGetTrend_PullsCorrectWindowAndPriorPeriod(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	monitorID := uuid.New()

	var pointsCalls, summaryCalls int
	var seenFroms []time.Time
	repo := &fakeTrendRepo{
		pointsFn: func(_ context.Context, _ uuid.UUID, from, to time.Time, bucket string) ([]domain.LatencyPercentilePoint, error) {
			pointsCalls++
			assert.Equal(t, "1 hour", bucket)
			assert.Equal(t, now, to)
			seenFroms = append(seenFroms, from)
			return []domain.LatencyPercentilePoint{{Time: from, P50: 100, P95: 200, P99: 300, SampleCount: 50}}, nil
		},
		summaryFn: func(_ context.Context, _ uuid.UUID, from, _ time.Time) (domain.LatencyTrendSummary, error) {
			summaryCalls++
			seenFroms = append(seenFroms, from)
			return domain.LatencyTrendSummary{P50: 100, P95: 200, P99: 300, SampleCount: 1000}, nil
		},
	}

	trend, err := NewLatencyTrendService(repo).GetTrend(context.Background(), monitorID, domain.TrendWindow7d, now)

	require.NoError(t, err)
	require.NotNil(t, trend)
	assert.Equal(t, int((7 * 24 * time.Hour).Seconds()), trend.WindowSeconds)
	assert.Equal(t, "1 hour", trend.BucketInterval)
	assert.Len(t, trend.Points, 1)
	assert.Equal(t, 1, pointsCalls)
	assert.Equal(t, 2, summaryCalls) // current + previous

	// Three reads (1 points + 2 summaries) hit two distinct From timestamps:
	// currentFrom = now - 7d, previousFrom = now - 14d.
	assert.Contains(t, seenFroms, now.Add(-7*24*time.Hour))
	assert.Contains(t, seenFroms, now.Add(-14*24*time.Hour))
}

func TestGetTrend_PropagatesPointsError(t *testing.T) {
	now := time.Now()
	repo := &fakeTrendRepo{
		pointsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) ([]domain.LatencyPercentilePoint, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := NewLatencyTrendService(repo).GetTrend(context.Background(), uuid.New(), domain.TrendWindow7d, now)
	assert.ErrorContains(t, err, "boom")
}

func TestGetTrend_PropagatesCurrentSummaryError(t *testing.T) {
	now := time.Now()
	var summaryCalls int
	repo := &fakeTrendRepo{
		pointsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, _ string) ([]domain.LatencyPercentilePoint, error) {
			return nil, nil
		},
		summaryFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) (domain.LatencyTrendSummary, error) {
			summaryCalls++
			return domain.LatencyTrendSummary{}, errors.New("db down")
		},
	}
	_, err := NewLatencyTrendService(repo).GetTrend(context.Background(), uuid.New(), domain.TrendWindow7d, now)
	assert.ErrorContains(t, err, "db down")
	assert.Equal(t, 1, summaryCalls, "should stop after first summary failure")
}

func TestGetTrend_DefaultsToSevenDayWindow(t *testing.T) {
	now := time.Now()
	repo := &fakeTrendRepo{
		pointsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, bucket string) ([]domain.LatencyPercentilePoint, error) {
			assert.Equal(t, "1 hour", bucket)
			return nil, nil
		},
		summaryFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) (domain.LatencyTrendSummary, error) {
			return domain.LatencyTrendSummary{}, nil
		},
	}
	trend, err := NewLatencyTrendService(repo).GetTrend(context.Background(), uuid.New(), domain.ParseTrendWindow("bogus"), now)
	require.NoError(t, err)
	assert.Equal(t, int((7 * 24 * time.Hour).Seconds()), trend.WindowSeconds)
}

func TestGetTrend_ThirtyDayUsesSixHourBuckets(t *testing.T) {
	now := time.Now()
	repo := &fakeTrendRepo{
		pointsFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time, bucket string) ([]domain.LatencyPercentilePoint, error) {
			assert.Equal(t, "6 hours", bucket)
			return nil, nil
		},
		summaryFn: func(_ context.Context, _ uuid.UUID, _, _ time.Time) (domain.LatencyTrendSummary, error) {
			return domain.LatencyTrendSummary{}, nil
		},
	}
	trend, err := NewLatencyTrendService(repo).GetTrend(context.Background(), uuid.New(), domain.TrendWindow30d, now)
	require.NoError(t, err)
	assert.Equal(t, "6 hours", trend.BucketInterval)
}
