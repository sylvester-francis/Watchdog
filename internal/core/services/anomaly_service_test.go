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

type fakeHBRepo struct {
	heartbeats []*domain.Heartbeat
	err        error
}

func (f *fakeHBRepo) GetByMonitorIDInRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*domain.Heartbeat, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.heartbeats, nil
}

func intPtr(n int) *int { return &n }

func newHBs(monitorID uuid.UUID, n int, latency int, spikeAt int, spikeMs int) []*domain.Heartbeat {
	hbs := make([]*domain.Heartbeat, n)
	for i := range hbs {
		l := latency
		if i == spikeAt {
			l = spikeMs
		}
		hbs[i] = &domain.Heartbeat{
			MonitorID: monitorID,
			Status:    domain.HeartbeatStatusUp,
			LatencyMs: intPtr(l),
			Time:      time.Now().Add(-time.Duration(n-i) * time.Minute),
		}
	}
	return hbs
}

func TestAnomalyService_RecentAnomalies(t *testing.T) {
	mID := uuid.New()
	svc := NewAnomalyService(&fakeHBRepo{heartbeats: newHBs(mID, 60, 100, 30, 10000)})
	got, err := svc.RecentAnomalies(context.Background(), mID, 24*time.Hour)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 10000, got[0].LatencyMs)
}

func TestAnomalyService_CountSince(t *testing.T) {
	mID := uuid.New()
	hbs := newHBs(mID, 60, 100, 30, 10000)
	hbs[31].LatencyMs = intPtr(12000)
	svc := NewAnomalyService(&fakeHBRepo{heartbeats: hbs})
	count, err := svc.CountSince(context.Background(), mID, time.Hour)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}

func TestAnomalyService_BubblesRepoError(t *testing.T) {
	svc := NewAnomalyService(&fakeHBRepo{err: errors.New("db down")})
	_, err := svc.RecentAnomalies(context.Background(), uuid.New(), time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anomaly service")
}

func TestAnomalyService_EmptyWhenNoData(t *testing.T) {
	svc := NewAnomalyService(&fakeHBRepo{heartbeats: nil})
	got, err := svc.RecentAnomalies(context.Background(), uuid.New(), time.Hour)
	require.NoError(t, err)
	assert.Empty(t, got)
}
