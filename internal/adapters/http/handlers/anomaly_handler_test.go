package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
)

type fakeAnomalyHB struct{ list []*domain.Heartbeat }

func (f *fakeAnomalyHB) GetByMonitorIDInRange(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*domain.Heartbeat, error) {
	return f.list, nil
}

func intPtrA(n int) *int { return &n }

func anomalyTestHeartbeats(monitorID uuid.UUID) []*domain.Heartbeat {
	hbs := make([]*domain.Heartbeat, 60)
	for i := range hbs {
		hbs[i] = &domain.Heartbeat{
			MonitorID: monitorID,
			Status:    domain.HeartbeatStatusUp,
			LatencyMs: intPtrA(100),
			Time:      time.Now().Add(-time.Duration(60-i) * time.Minute),
		}
	}
	hbs[30].LatencyMs = intPtrA(10000)
	return hbs
}

func buildAnomalyHandler(hbs []*domain.Heartbeat) *AnomalyHandler {
	svc := services.NewAnomalyService(&fakeAnomalyHB{list: hbs})
	return NewAnomalyHandler(svc)
}

func TestAnomalyHandler_ListAnomalies(t *testing.T) {
	mID := uuid.New()
	h := buildAnomalyHandler(anomalyTestHeartbeats(mID))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+mID.String()+"/anomalies", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(mID.String())

	require.NoError(t, h.ListAnomalies(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []struct {
			Time      time.Time `json:"time"`
			LatencyMs int       `json:"latency_ms"`
			Method    string    `json:"method"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, 10000, resp.Data[0].LatencyMs)
}

func TestAnomalyHandler_CountAnomalies(t *testing.T) {
	mID := uuid.New()
	h := buildAnomalyHandler(anomalyTestHeartbeats(mID))
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+mID.String()+"/anomalies/count", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(mID.String())

	require.NoError(t, h.CountAnomalies(c))
	var resp struct {
		Count int `json:"count"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Count)
}

func TestAnomalyHandler_InvalidMonitorID(t *testing.T) {
	h := buildAnomalyHandler(nil)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/garbage/anomalies", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("garbage")
	require.NoError(t, h.ListAnomalies(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAnomalyHandler_EmptyReturnsEmptyArray(t *testing.T) {
	// When the service returns no anomalies, the JSON `data` field must be
	// an empty array `[]`, not `null` — frontend consumers expect array.
	h := buildAnomalyHandler(nil)
	e := echo.New()
	mID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/"+mID.String()+"/anomalies", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(mID.String())

	require.NoError(t, h.ListAnomalies(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestParseAnomalyWindow(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"", time.Hour},                       // empty → default
		{"30m", 30 * time.Minute},             // duration string
		{"3600", time.Hour},                   // integer seconds
		{"garbage", time.Hour},                // unparseable → default
		{"8760h", time.Hour},                  // exceeds max → default
		{"-5m", time.Hour},                    // negative → default
	}
	for _, c := range cases {
		got := parseAnomalyWindow(c.raw, time.Hour, 7*24*time.Hour)
		assert.Equal(t, c.want, got, "input=%q", c.raw)
	}
}
