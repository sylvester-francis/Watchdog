package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

type fakeSpanRepo struct {
	mu       sync.Mutex
	inserted []*domain.Span
	err      error

	getByTraceID    func(ctx context.Context, userID uuid.UUID, traceID []byte) ([]*domain.Span, error)
	listRecentFn    func(ctx context.Context, userID uuid.UUID, since time.Time, service string, limit int) ([]*domain.TraceSummary, error)
}

func (f *fakeSpanRepo) InsertBatch(_ context.Context, spans []*domain.Span) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.inserted = append(f.inserted, spans...)
	return nil
}

func (f *fakeSpanRepo) GetByTraceID(ctx context.Context, userID uuid.UUID, traceID []byte) ([]*domain.Span, error) {
	if f.getByTraceID != nil {
		return f.getByTraceID(ctx, userID, traceID)
	}
	return nil, nil
}
func (f *fakeSpanRepo) DeleteOlderThan(context.Context, time.Time) error {
	return nil
}
func (f *fakeSpanRepo) ListRecentTraces(ctx context.Context, userID uuid.UUID, since time.Time, service string, limit int) ([]*domain.TraceSummary, error) {
	if f.listRecentFn != nil {
		return f.listRecentFn(ctx, userID, since, service, limit)
	}
	return nil, nil
}

// defaultTestUserID is used by newTracesTestServer for tests that don't
// care about scoping but still need auth context to pass the handler's
// authentication guard. Scoping-specific tests should use
// newScopedTracesTestServer with explicit values.
var defaultTestUserID = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")

func newTracesTestServer(t *testing.T, repo *fakeSpanRepo) *echo.Echo {
	t.Helper()
	return newScopedTracesTestServer(t, repo, defaultTestUserID.String(), "default")
}

func postOTLP(t *testing.T, e *echo.Echo, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestTracesHandler_AcceptsValidOTLP(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "GET /healthz",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
	}, "svc")
	body, err := proto.Marshal(req)
	require.NoError(t, err)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/x-protobuf", rec.Header().Get("Content-Type"))
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, "svc", repo.inserted[0].ServiceName)

	var resp coltracepb.ExportTraceServiceResponse
	require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Nil(t, resp.GetPartialSuccess(), "no partial success when nothing dropped")
}

func TestTracesHandler_AcceptsGzippedOTLP(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "GET /healthz",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
	}, "svc")
	body, err := proto.Marshal(req)
	require.NoError(t, err)

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err = gz.Write(body)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	httpReq := httptest.NewRequest(http.MethodPost, "/v1/traces", &compressed)
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httpReq)

	require.Equal(t, http.StatusOK, rec.Code, "gzip-encoded OTLP body should be accepted")
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, "svc", repo.inserted[0].ServiceName)
}

func TestTracesHandler_RejectsWrongContentType(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	req := requestWithSpan(&tracepb.Span{
		TraceId: bytes16(0x01), SpanId: bytes8(0x02),
		Name: "x", StartTimeUnixNano: 1, EndTimeUnixNano: 2,
	}, "svc")
	body, _ := proto.Marshal(req)

	rec := postOTLP(t, e, body, "application/json")
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestTracesHandler_RejectsMalformedProtobuf(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	rec := postOTLP(t, e, []byte("not a protobuf"), "application/x-protobuf")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestTracesHandler_AcceptsEmptyRequest(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	body, _ := proto.Marshal(&coltracepb.ExportTraceServiceRequest{})
	rec := postOTLP(t, e, body, "application/x-protobuf")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestTracesHandler_ReportsPartialSuccessWhenSpansDropped(t *testing.T) {
	repo := &fakeSpanRepo{}
	e := newTracesTestServer(t, repo)

	bigVal := strings.Repeat("y", 70_000)
	oversize := &tracepb.Span{
		TraceId: bytes16(0xFE), SpanId: bytes8(0xFE),
		Name:              "huge",
		StartTimeUnixNano: 1, EndTimeUnixNano: 2,
		Attributes: []*commonpb.KeyValue{kvString("blob", bigVal)},
	}
	keep := &tracepb.Span{
		TraceId: bytes16(0x01), SpanId: bytes8(0x01),
		Name:              "small",
		StartTimeUnixNano: 3, EndTimeUnixNano: 4,
	}
	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: resourceWithService("svc"),
				ScopeSpans: []*tracepb.ScopeSpans{
					{Spans: []*tracepb.Span{oversize, keep}},
				},
			},
		},
	}
	body, _ := proto.Marshal(req)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 1, "only the small span survives")

	var resp coltracepb.ExportTraceServiceResponse
	require.NoError(t, proto.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.GetPartialSuccess())
	assert.EqualValues(t, 1, resp.GetPartialSuccess().GetRejectedSpans())
	assert.NotEmpty(t, resp.GetPartialSuccess().GetErrorMessage())
}

func TestTracesHandler_ReturnsServerErrorOnRepoFailure(t *testing.T) {
	repo := &fakeSpanRepo{err: errors.New("db down")}
	e := newTracesTestServer(t, repo)

	req := requestWithSpan(&tracepb.Span{
		TraceId: bytes16(0x01), SpanId: bytes8(0x02),
		Name: "x", StartTimeUnixNano: 1, EndTimeUnixNano: 2,
	}, "svc")
	body, _ := proto.Marshal(req)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// withAuthCtx wraps the handler with a tiny middleware that mimics what
// APITokenAuth + TenantScope set up at runtime: a UserID in the echo
// context and a tenant_id in the request context.
func withAuthCtx(userID, tenantID string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if userID != "" {
				c.Set(middleware.UserIDKey, userID)
			}
			if tenantID != "" {
				ctx := repository.WithTenantID(c.Request().Context(), tenantID)
				c.SetRequest(c.Request().WithContext(ctx))
			}
			return next(c)
		}
	}
}

func newScopedTracesTestServer(t *testing.T, repo *fakeSpanRepo, userID, tenantID string) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Use(withAuthCtx(userID, tenantID))
	h := NewTracesHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.POST("/v1/traces", h.Handle)
	return e
}

func TestTracesHandler_StampsUserIDFromContextOntoEverySpan(t *testing.T) {
	repo := &fakeSpanRepo{}
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	e := newScopedTracesTestServer(t, repo, userID.String(), "default")

	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: resourceWithService("svc"),
				ScopeSpans: []*tracepb.ScopeSpans{
					{Spans: []*tracepb.Span{
						{TraceId: bytes16(0x01), SpanId: bytes8(0x01), Name: "a", StartTimeUnixNano: 1, EndTimeUnixNano: 2},
						{TraceId: bytes16(0x01), SpanId: bytes8(0x02), Name: "b", StartTimeUnixNano: 3, EndTimeUnixNano: 4},
					}},
				},
			},
		},
	}
	body, err := proto.Marshal(req)
	require.NoError(t, err)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 2)
	for i, s := range repo.inserted {
		assert.Equal(t, userID, s.UserID, "span %d should carry user_id from request context", i)
	}
}

func TestTracesHandler_StampsTenantIDFromContextOntoEverySpan(t *testing.T) {
	repo := &fakeSpanRepo{}
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	const tenantID = "acme-corp"
	e := newScopedTracesTestServer(t, repo, userID.String(), tenantID)

	req := requestWithSpan(&tracepb.Span{
		TraceId: bytes16(0xAB), SpanId: bytes8(0xCD),
		Name: "x", StartTimeUnixNano: 1, EndTimeUnixNano: 2,
	}, "svc")
	body, err := proto.Marshal(req)
	require.NoError(t, err)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, tenantID, repo.inserted[0].TenantID)
}

func TestTracesHandler_RejectsRequestWithoutUserID(t *testing.T) {
	repo := &fakeSpanRepo{}
	// No UserID set on context — simulates a misconfigured route mount
	// (auth middleware missing). Handler must fail loud, not silently
	// drop spans into the database with a zero UUID.
	e := newScopedTracesTestServer(t, repo, "", "default")

	req := requestWithSpan(&tracepb.Span{
		TraceId: bytes16(0x01), SpanId: bytes8(0x02),
		Name: "x", StartTimeUnixNano: 1, EndTimeUnixNano: 2,
	}, "svc")
	body, _ := proto.Marshal(req)

	rec := postOTLP(t, e, body, "application/x-protobuf")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, repo.inserted, "no spans should be persisted without an authenticated user")
}
