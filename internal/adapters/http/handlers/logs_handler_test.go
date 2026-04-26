package handlers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type fakeLogRepo struct {
	mu       sync.Mutex
	inserted []*domain.LogRecord
	err      error

	listRecentFn func(ctx context.Context, since time.Time, service, severity string, limit int) ([]*domain.LogRecord, error)
}

func (f *fakeLogRepo) InsertBatch(_ context.Context, records []*domain.LogRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.inserted = append(f.inserted, records...)
	return nil
}

func (f *fakeLogRepo) ListRecent(ctx context.Context, since time.Time, service, severity string, limit int) ([]*domain.LogRecord, error) {
	if f.listRecentFn != nil {
		return f.listRecentFn(ctx, since, service, severity, limit)
	}
	return nil, nil
}

func (f *fakeLogRepo) DeleteOlderThan(context.Context, time.Time) error {
	return nil
}

func newLogsTestServer(t *testing.T, repo *fakeLogRepo) *echo.Echo {
	t.Helper()
	e := echo.New()
	h := NewLogsHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.POST("/v1/logs", h.Handle)
	return e
}

func postLogsOTLP(t *testing.T, e *echo.Echo, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func requestWithLog(lr *logspb.LogRecord, service string) *collogspb.ExportLogsServiceRequest {
	return &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService(service),
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{lr},
			}},
		}},
	}
}

func TestLogsHandler_AcceptsValidOTLP(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsTestServer(t, repo)

	req := requestWithLog(&logspb.LogRecord{
		TimeUnixNano: 1, ObservedTimeUnixNano: 1,
		SeverityNumber: logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
		SeverityText:   "INFO",
		Body:           anyValueString("hello"),
	}, "svc")
	body, err := proto.Marshal(req)
	require.NoError(t, err)

	rec := postLogsOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/x-protobuf", rec.Header().Get("Content-Type"))
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, "svc", repo.inserted[0].ServiceName)
	assert.Equal(t, "hello", repo.inserted[0].Body)

	var resp collogspb.ExportLogsServiceResponse
	require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Nil(t, resp.GetPartialSuccess())
}

func TestLogsHandler_RejectsWrongContentType(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsTestServer(t, repo)

	body, _ := proto.Marshal(requestWithLog(&logspb.LogRecord{
		TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: anyValueString("x"),
	}, "svc"))

	rec := postLogsOTLP(t, e, body, "application/json")
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestLogsHandler_RejectsMalformedProtobuf(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsTestServer(t, repo)

	rec := postLogsOTLP(t, e, []byte("not a protobuf"), "application/x-protobuf")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestLogsHandler_AcceptsEmptyRequest(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsTestServer(t, repo)

	body, _ := proto.Marshal(&collogspb.ExportLogsServiceRequest{})
	rec := postLogsOTLP(t, e, body, "application/x-protobuf")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestLogsHandler_ReportsPartialSuccessWhenRecordsDropped(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newLogsTestServer(t, repo)

	bigBody := strings.Repeat("y", 70_000)
	oversize := &logspb.LogRecord{
		TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: anyValueString(bigBody),
	}
	keep := &logspb.LogRecord{
		TimeUnixNano: 2, ObservedTimeUnixNano: 2, Body: anyValueString("ok"),
	}
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService("svc"),
			ScopeLogs: []*logspb.ScopeLogs{{LogRecords: []*logspb.LogRecord{oversize, keep}}},
		}},
	}
	body, _ := proto.Marshal(req)

	rec := postLogsOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 1)

	var resp collogspb.ExportLogsServiceResponse
	require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.GetPartialSuccess())
	assert.EqualValues(t, 1, resp.GetPartialSuccess().GetRejectedLogRecords())
	assert.NotEmpty(t, resp.GetPartialSuccess().GetErrorMessage())
}

func TestLogsHandler_ReturnsServerErrorOnRepoFailure(t *testing.T) {
	repo := &fakeLogRepo{err: errors.New("db down")}
	e := newLogsTestServer(t, repo)

	req := requestWithLog(&logspb.LogRecord{
		TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: anyValueString("x"),
	}, "svc")
	body, _ := proto.Marshal(req)

	rec := postLogsOTLP(t, e, body, "application/x-protobuf")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ensure commonpb import used
var _ = commonpb.AnyValue{}
