package services_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

type stubSettings struct {
	value []byte
	err   error
	calls atomic.Int32
}

func (s *stubSettings) Get(_ context.Context, _ string) ([]byte, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return s.value, nil
}
func (s *stubSettings) Set(context.Context, string, []byte, uuid.UUID) error { return nil }

type stubSpans struct {
	cutoffs []time.Time
	err     error
}

func (s *stubSpans) InsertBatch(context.Context, []*domain.Span) error  { return nil }
func (s *stubSpans) GetByTraceID(context.Context, []byte) ([]*domain.Span, error) { return nil, nil }
func (s *stubSpans) ListRecentTraces(context.Context, time.Time, string, int) ([]*domain.TraceSummary, error) {
	return nil, nil
}
func (s *stubSpans) DeleteOlderThan(_ context.Context, cutoff time.Time) error {
	s.cutoffs = append(s.cutoffs, cutoff)
	return s.err
}

func newRetention(spans *stubSpans, settings *stubSettings) *services.TraceRetention {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	return services.NewTraceRetention(spans, settings, logger)
}

func TestTraceRetention_UsesConfiguredDays(t *testing.T) {
	spans := &stubSpans{}
	settings := &stubSettings{value: []byte(`14`)}

	r := newRetention(spans, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, spans.cutoffs, 1)
	assert.Equal(t, now.Add(-14*24*time.Hour), spans.cutoffs[0])
}

func TestTraceRetention_FallsBackToDefaultWhenSettingMissing(t *testing.T) {
	spans := &stubSpans{}
	settings := &stubSettings{err: errors.New("not found")}

	r := newRetention(spans, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, spans.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), spans.cutoffs[0],
		"missing setting falls back to 7-day default")
}

func TestTraceRetention_FallsBackOnInvalidJSON(t *testing.T) {
	spans := &stubSpans{}
	settings := &stubSettings{value: []byte(`"not-a-number"`)}

	r := newRetention(spans, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, spans.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), spans.cutoffs[0])
}

func TestTraceRetention_RejectsNonPositiveDays(t *testing.T) {
	spans := &stubSpans{}
	settings := &stubSettings{value: []byte(`0`)}

	r := newRetention(spans, settings)
	now := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	require.NoError(t, r.RunOnce(context.Background(), now))

	require.Len(t, spans.cutoffs, 1)
	assert.Equal(t, now.Add(-7*24*time.Hour), spans.cutoffs[0],
		"a 0-day retention would wipe live data; coerce to default")
}

func TestTraceRetention_PropagatesDeleteError(t *testing.T) {
	spans := &stubSpans{err: errors.New("db down")}
	settings := &stubSettings{value: []byte(`7`)}

	r := newRetention(spans, settings)
	err := r.RunOnce(context.Background(), time.Now())
	require.Error(t, err)
}
