package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newNDJSONServer(t *testing.T, repo *fakeLogRepo) *echo.Echo {
	t.Helper()
	e := echo.New()
	h := NewLogsNDJSONHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.POST("/v1/logs/raw", h.Handle)
	return e
}

func postNDJSON(t *testing.T, e *echo.Echo, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/logs/raw", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestLogsNDJSON_AcceptsValid(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"hello","service":"api"}` + "\n" +
		`{"timestamp":"2026-04-26T12:00:01Z","severity":"ERROR","body":"oops","service":"api","attributes":{"user":"u-1"}}` + "\n"

	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ndjsonResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Accepted)
	assert.Zero(t, resp.Rejected)

	require.Len(t, repo.inserted, 2)
	assert.Equal(t, "hello", repo.inserted[0].Body)
	assert.Equal(t, "INFO", repo.inserted[0].SeverityText)
	assert.Equal(t, "api", repo.inserted[0].ServiceName)
	assert.Equal(t, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC), repo.inserted[0].Timestamp.UTC())

	var attrs map[string]any
	require.NoError(t, json.Unmarshal(repo.inserted[1].Attributes, &attrs))
	assert.Equal(t, "u-1", attrs["user"])
}

func TestLogsNDJSON_DecodesHexTraceIDs(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	traceHex := strings.Repeat("ab", 16)
	spanHex := strings.Repeat("cd", 8)
	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"x","service":"api","trace_id":"` + traceHex + `","span_id":"` + spanHex + `"}` + "\n"

	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 1)

	want, _ := hex.DecodeString(traceHex)
	wantSpan, _ := hex.DecodeString(spanHex)
	assert.Equal(t, want, repo.inserted[0].TraceID)
	assert.Equal(t, wantSpan, repo.inserted[0].SpanID)
}

func TestLogsNDJSON_RejectsWrongContentType(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"x","service":"api"}` + "\n"
	rec := postNDJSON(t, e, body, "application/json")
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Empty(t, repo.inserted)
}

func TestLogsNDJSON_SkipsBlankLines(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	body := "\n" +
		`{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"x","service":"api"}` + "\n" +
		"\n" +
		`{"timestamp":"2026-04-26T12:00:01Z","severity":"INFO","body":"y","service":"api"}` + "\n"

	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.inserted, 2)
}

func TestLogsNDJSON_RejectsBadLineButContinues(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"good","service":"api"}` + "\n" +
		`{this is not json}` + "\n" +
		`{"timestamp":"2026-04-26T12:00:02Z","severity":"INFO","body":"good2","service":"api"}` + "\n"

	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ndjsonResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Accepted)
	assert.Equal(t, 1, resp.Rejected)
	assert.NotEmpty(t, resp.Errors)
	require.Len(t, repo.inserted, 2)
}

func TestLogsNDJSON_RejectsRecordMissingRequiredFields(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	// Missing service
	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"x"}` + "\n"
	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ndjsonResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Zero(t, resp.Accepted)
	assert.Equal(t, 1, resp.Rejected)
	assert.Empty(t, repo.inserted)
}

func TestLogsNDJSON_DropsOversizeRecords(t *testing.T) {
	repo := &fakeLogRepo{}
	e := newNDJSONServer(t, repo)

	bigBody := strings.Repeat("z", 70_000)
	line, _ := json.Marshal(map[string]any{
		"timestamp": "2026-04-26T12:00:00Z",
		"severity":  "INFO",
		"body":      bigBody,
		"service":   "api",
	})
	body := string(line) + "\n" +
		`{"timestamp":"2026-04-26T12:00:01Z","severity":"INFO","body":"ok","service":"api"}` + "\n"

	rec := postNDJSON(t, e, body, "application/x-ndjson")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ndjsonResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Accepted)
	assert.Equal(t, 1, resp.Rejected)
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, "ok", repo.inserted[0].Body)
}

func TestLogsNDJSON_RepoFailureReturns500(t *testing.T) {
	repo := &fakeLogRepo{err: errors.New("db down")}
	e := newNDJSONServer(t, repo)

	body := `{"timestamp":"2026-04-26T12:00:00Z","severity":"INFO","body":"x","service":"api"}` + "\n"
	rec := postNDJSON(t, e, body, "application/x-ndjson")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

