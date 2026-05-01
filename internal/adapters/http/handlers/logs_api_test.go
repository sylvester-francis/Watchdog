package handlers

import (
	"context"
	"encoding/hex"
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
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

func newLogsAPIServer(repo *fakeLogRepo) *echo.Echo {
	return newScopedLogsAPIServer(repo, defaultTestUserID.String(), "default")
}

func newScopedLogsAPIServer(repo *fakeLogRepo, userID, tenantID string) *echo.Echo {
	e := echo.New()
	e.Use(withAuthCtx(userID, tenantID))
	h := NewLogsAPIHandler(repo)
	e.GET("/api/v1/logs", h.ListLogs)
	return e
}

func TestLogsAPI_ListLogs_EmptyResult(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logRecordResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Empty(t, body.Data)
}

func TestLogsAPI_ListLogs_HexEncodesTraceAndSpanIDs(t *testing.T) {
	traceID := bytes16(0xAB)
	spanID := bytes8(0xCD)
	repo := &fakeLogRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ string, _ int) ([]*domain.LogRecord, error) {
			return []*domain.LogRecord{{
				Timestamp:      time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
				TraceID:        traceID,
				SpanID:         spanID,
				SeverityNumber: domain.SeverityInfo,
				SeverityText:   "INFO",
				Body:           "hello",
				ServiceName:    "api",
				Attributes:     []byte(`{"user":"u-1"}`),
			}}, nil
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data []logRecordResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, hex.EncodeToString(traceID), body.Data[0].TraceID)
	assert.Equal(t, hex.EncodeToString(spanID), body.Data[0].SpanID)
	assert.Equal(t, "hello", body.Data[0].Body)
	assert.Equal(t, "INFO", body.Data[0].SeverityText)
	assert.JSONEq(t, `{"user":"u-1"}`, string(body.Data[0].Attributes))
}

func TestLogsAPI_ListLogs_OmitsEmptyTraceID(t *testing.T) {
	repo := &fakeLogRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ string, _ int) ([]*domain.LogRecord, error) {
			return []*domain.LogRecord{{
				Timestamp:   time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
				Body:        "no-trace",
				ServiceName: "api",
			}}, nil
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), `"trace_id"`,
		"empty trace_id is omitted, not rendered as empty string")
}

func TestLogsAPI_ListLogs_PassesServiceSeverityAndLimit(t *testing.T) {
	var captured struct {
		service, severity string
		limit             int
	}
	repo := &fakeLogRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, service, severity string, limit int) ([]*domain.LogRecord, error) {
			captured.service = service
			captured.severity = severity
			captured.limit = limit
			return nil, nil
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs?service=api&severity=ERROR&limit=25", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "api", captured.service)
	assert.Equal(t, "ERROR", captured.severity)
	assert.Equal(t, 25, captured.limit)
}

func TestLogsAPI_ListLogs_ClampsLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeLogRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ string, limit int) ([]*domain.LogRecord, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=999999", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 500, capturedLimit, "huge limit clamps to max 500")
}

func TestLogsAPI_ListLogs_DefaultsLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeLogRepo{
		listRecentFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ string, limit int) ([]*domain.LogRecord, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 100, capturedLimit, "default limit is 100")
}

func TestLogsAPI_ListLogs_RejectsBadLimit(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=oops", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogsAPI_ListLogs_RejectsBadSince(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs?since=not-a-time", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogsAPI_ListLogs_RepoErrorReturns500(t *testing.T) {
	repo := &fakeLogRepo{
		listRecentFn: func(context.Context, uuid.UUID, time.Time, string, string, int) ([]*domain.LogRecord, error) {
			return nil, assertErr{}
		},
	}
	e := newLogsAPIServer(repo)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

func TestLogsAPI_ListLogs_PassesUserAndTenantToRepo(t *testing.T) {
	userID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	const tenantID = "acme-corp"

	var captured struct {
		userID   uuid.UUID
		tenantID string
	}
	repo := &fakeLogRepo{
		listRecentFn: func(ctx context.Context, uid uuid.UUID, _ time.Time, _, _ string, _ int) ([]*domain.LogRecord, error) {
			captured.userID = uid
			captured.tenantID = repository.TenantIDFromContext(ctx)
			return nil, nil
		},
	}
	e := newScopedLogsAPIServer(repo, userID.String(), tenantID)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, captured.userID)
	assert.Equal(t, tenantID, captured.tenantID)
}

func TestLogsAPI_ListLogs_RejectsRequestWithoutUserID(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newScopedLogsAPIServer(repo, "", "default")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
