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
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type fakeSpanRepo struct {
	mu       sync.Mutex
	inserted []*domain.Span
	err      error
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

func (f *fakeSpanRepo) GetByTraceID(context.Context, []byte) ([]*domain.Span, error) {
	return nil, nil
}
func (f *fakeSpanRepo) DeleteOlderThan(context.Context, time.Time) error {
	return nil
}

func newTracesTestServer(t *testing.T, repo *fakeSpanRepo) *echo.Echo {
	t.Helper()
	e := echo.New()
	h := NewTracesHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.POST("/v1/traces", h.Handle)
	return e
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
