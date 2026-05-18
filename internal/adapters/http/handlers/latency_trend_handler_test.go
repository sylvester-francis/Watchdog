package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type fakeTrendService struct {
	trend *domain.LatencyTrend
	err   error
	seen  struct {
		window domain.TrendWindow
		now    time.Time
	}
}

func (f *fakeTrendService) GetTrend(_ context.Context, _ uuid.UUID, w domain.TrendWindow, now time.Time) (*domain.LatencyTrend, error) {
	f.seen.window, f.seen.now = w, now
	return f.trend, f.err
}

func newHandler(svc *fakeTrendService, fixedNow time.Time) *LatencyTrendHandler {
	return &LatencyTrendHandler{svc: svc, now: func() time.Time { return fixedNow }}
}

func TestGetLatencyTrend_OK(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	svc := &fakeTrendService{
		trend: &domain.LatencyTrend{
			WindowSeconds:  7 * 24 * 3600,
			BucketInterval: "1 hour",
			Points:         []domain.LatencyPercentilePoint{{Time: now, P50: 100, P95: 200, P99: 300, SampleCount: 10}},
			Current:        domain.LatencyTrendSummary{P50: 100, P95: 200, P99: 300, SampleCount: 100},
			Previous:       domain.LatencyTrendSummary{P50: 90, P95: 180, P99: 270, SampleCount: 100},
		},
	}
	h := newHandler(svc, now)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?window=7d", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	require.NoError(t, h.GetLatencyTrend(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, domain.TrendWindow7d, svc.seen.window)

	var out domain.LatencyTrend
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, 100, out.Current.P50)
	assert.Equal(t, 200, out.Current.P95)
}

func TestGetLatencyTrend_DefaultsTo7d(t *testing.T) {
	svc := &fakeTrendService{trend: &domain.LatencyTrend{}}
	h := newHandler(svc, time.Now())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	require.NoError(t, h.GetLatencyTrend(c))
	assert.Equal(t, domain.TrendWindow7d, svc.seen.window)
}

func TestGetLatencyTrend_InvalidIDIs400(t *testing.T) {
	h := newHandler(&fakeTrendService{}, time.Now())
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	err := h.GetLatencyTrend(c)
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}

func TestGetLatencyTrend_ServiceErrorIs500(t *testing.T) {
	h := newHandler(&fakeTrendService{err: errors.New("db down")}, time.Now())
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	err := h.GetLatencyTrend(c)
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
}
