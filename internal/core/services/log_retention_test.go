package services_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

type stubLogs struct {
	cutoffs []time.Time
	err     error
}

func (s *stubLogs) InsertBatch(context.Context, []*domain.LogRecord) error { return nil }
func (s *stubLogs) ListRecent(context.Context, uuid.UUID, time.Time, string, string, []byte, []byte, int) ([]*domain.LogRecord, error) {
	return nil, nil
}
func (s *stubLogs) DeleteOlderThan(_ context.Context, cutoff time.Time) error {
	s.cutoffs = append(s.cutoffs, cutoff)
	return s.err
}

func newLogRetention(logs *stubLogs, settings *stubSettings) *services.LogRetention {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return services.NewLogRetention(logs, settings, logger)
}

func TestLogRetention_UsesConfiguredDays(t *testing.T) {
	logs := &stubLogs{}
	settings := &stubSettings{value: []byte(`30`)}

	r := newLogRetention(logs, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, logs.cutoffs, 1)
	assert.Equal(t, now.Add(-30*24*time.Hour), logs.cutoffs[0])
}

func TestLogRetention_FallsBackToDefaultWhenSettingMissing(t *testing.T) {
	logs := &stubLogs{}
	settings := &stubSettings{err: errors.New("not found")}

	r := newLogRetention(logs, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, logs.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), logs.cutoffs[0],
		"missing setting falls back to 7-day default")
}

func TestLogRetention_FallsBackOnInvalidJSON(t *testing.T) {
	logs := &stubLogs{}
	settings := &stubSettings{value: []byte(`"not-a-number"`)}

	r := newLogRetention(logs, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, logs.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), logs.cutoffs[0])
}

func TestLogRetention_RejectsNonPositiveDays(t *testing.T) {
	logs := &stubLogs{}
	settings := &stubSettings{value: []byte(`0`)}

	r := newLogRetention(logs, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, logs.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), logs.cutoffs[0],
		"a 0-day retention would wipe live data; coerce to default")
}

func TestLogRetention_PropagatesDeleteError(t *testing.T) {
	logs := &stubLogs{err: errors.New("db down")}
	settings := &stubSettings{value: []byte(`7`)}

	r := newLogRetention(logs, settings)
	err := r.RunOnce(context.Background(), time.Now())
	require.Error(t, err)
}
