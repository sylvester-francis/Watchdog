package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

func newTracesAPIServer(repo *fakeSpanRepo) *echo.Echo {
	return newScopedTracesAPIServer(repo, defaultTestUserID.String(), "default")
}

func newScopedTracesAPIServer(repo *fakeSpanRepo, userID, tenantID string) *echo.Echo {
	e := echo.New()
	e.Use(withAuthCtx(userID, tenantID))
	h := NewTracesAPIHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.GET("/api/v1/traces", h.ListTraces)
	e.GET("/api/v1/traces/:trace_id", h.GetTrace)
	return e
}

func TestTracesAPI_ListTraces_EmptyResult(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []traceSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Empty(t, body.Data)
}

func TestTracesAPI_ListTraces_HexEncodesTraceID(t *testing.T) {
	traceID := bytes16(0xAB)
	repo := &fakeSpanRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string, _ int) ([]*domain.TraceSummary, error) {
			return []*domain.TraceSummary{{
				TraceID:    traceID,
				StartTime:  time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
				DurationNS: 1_500_000,
				SpanCount:  4,
				HasError:   true,
			}}, nil
		},
	}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []traceSummaryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, hex.EncodeToString(traceID), body.Data[0].TraceID)
	assert.True(t, body.Data[0].HasError)
	assert.EqualValues(t, 4, body.Data[0].SpanCount)
}

func TestTracesAPI_ListTraces_PassesServiceAndLimit(t *testing.T) {
	var captured struct {
		service string
		limit   int
	}
	repo := &fakeSpanRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, service string, limit int) ([]*domain.TraceSummary, error) {
			captured.service = service
			captured.limit = limit
			return nil, nil
		},
	}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces?service=api&limit=25", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "api", captured.service)
	assert.Equal(t, 25, captured.limit)
}

func TestTracesAPI_ListTraces_ClampsLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeSpanRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string, limit int) ([]*domain.TraceSummary, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces?limit=999999", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 200, capturedLimit, "huge limit clamps to max 200")
}

func TestTracesAPI_ListTraces_DefaultsLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeSpanRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string, limit int) ([]*domain.TraceSummary, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 50, capturedLimit, "default limit is 50")
}

func TestTracesAPI_ListTraces_RejectsBadLimit(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces?limit=oops", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTracesAPI_GetTrace_RejectsBadHex(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/not-hex", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTracesAPI_GetTrace_RejectsWrongLengthTraceID(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/abcdef", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "trace_id must decode to exactly 16 bytes")
}

func TestTracesAPI_GetTrace_NotFound(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/"+strings.Repeat("ab", 16), nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTracesAPI_GetTrace_ReturnsHexEncodedSpans(t *testing.T) {
	traceID := bytes16(0xAB)
	spanID := bytes8(0x01)
	parent := bytes8(0x02)
	repo := &fakeSpanRepo{
		getByTraceID: func(_ context.Context, _ uuid.UUID, _ []byte) ([]*domain.Span, error) {
			return []*domain.Span{
				{
					TraceID:      traceID,
					SpanID:       spanID,
					ParentSpanID: parent,
					Name:         "GET /healthz",
					ServiceName:  "api",
					StartTime:    time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
					EndTime:      time.Date(2026, 4, 26, 12, 0, 0, 500_000_000, time.UTC),
					DurationNS:   500_000_000,
					Attributes:   []byte(`{"http.method":"GET"}`),
				},
			}, nil
		},
	}
	e := newTracesAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/"+hex.EncodeToString(traceID), nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []spanResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, hex.EncodeToString(traceID), body.Data[0].TraceID)
	assert.Equal(t, hex.EncodeToString(spanID), body.Data[0].SpanID)
	assert.Equal(t, hex.EncodeToString(parent), body.Data[0].ParentSpanID)
	assert.Equal(t, "GET /healthz", body.Data[0].Name)
	assert.Equal(t, "api", body.Data[0].ServiceName)
	assert.JSONEq(t, `{"http.method":"GET"}`, string(body.Data[0].Attributes))
}

func TestTracesAPI_ListTraces_PassesUserAndTenantToRepo(t *testing.T) {
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	const tenantID = "acme-corp"

	var captured struct {
		userID   uuid.UUID
		tenantID string
	}
	repo := &fakeSpanRepo{
		listRecentFn: func(ctx context.Context, uid uuid.UUID, _ time.Time, _ string, _ int) ([]*domain.TraceSummary, error) {
			captured.userID = uid
			captured.tenantID = repository.TenantIDFromContext(ctx)
			return nil, nil
		},
	}
	e := newScopedTracesAPIServer(repo, userID.String(), tenantID)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, captured.userID)
	assert.Equal(t, tenantID, captured.tenantID)
}

func TestTracesAPI_ListTraces_RejectsRequestWithoutUserID(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newScopedTracesAPIServer(repo, "", "default")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTracesAPI_GetTrace_PassesUserAndTenantToRepo(t *testing.T) {
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	const tenantID = "acme-corp"
	traceID := bytes16(0xAB)

	var captured struct {
		userID   uuid.UUID
		tenantID string
	}
	repo := &fakeSpanRepo{
		getByTraceID: func(ctx context.Context, uid uuid.UUID, _ []byte) ([]*domain.Span, error) {
			captured.userID = uid
			captured.tenantID = repository.TenantIDFromContext(ctx)
			return []*domain.Span{{TraceID: traceID, SpanID: bytes8(0x01), StartTime: time.Now(), EndTime: time.Now()}}, nil
		},
	}
	e := newScopedTracesAPIServer(repo, userID.String(), tenantID)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/"+hex.EncodeToString(traceID), nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, captured.userID)
	assert.Equal(t, tenantID, captured.tenantID)
}

func TestTracesAPI_GetTrace_RejectsRequestWithoutUserID(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newScopedTracesAPIServer(repo, "", "default")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/traces/"+strings.Repeat("ab", 16), nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
