package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

const (
	defaultTraceListLimit = 50
	maxTraceListLimit     = 200
	defaultListLookback   = 24 * time.Hour
	traceIDByteLength     = 16
)

// TracesAPIHandler serves the read-side trace APIs that #15's explorer
// UI consumes. Lives under /api/v1/traces with the same hybrid auth as
// the rest of the v1 API (session OR bearer token).
type TracesAPIHandler struct {
	repo   ports.SpanRepository
	logger *slog.Logger
}

// NewTracesAPIHandler constructs a TracesAPIHandler.
func NewTracesAPIHandler(repo ports.SpanRepository, logger *slog.Logger) *TracesAPIHandler {
	return &TracesAPIHandler{repo: repo, logger: logger}
}

// traceSummaryResponse is the wire-shape for a single row in the
// trace-list view. trace_id is hex-encoded so JSON consumers don't
// have to decode base64.
type traceSummaryResponse struct {
	TraceID     string    `json:"trace_id"`
	StartTime   time.Time `json:"start_time"`
	DurationNS  int64     `json:"duration_ns"`
	SpanCount   int       `json:"span_count"`
	HasError    bool      `json:"has_error"`
	ServiceName string    `json:"service_name,omitempty"`
	RootName    string    `json:"root_name,omitempty"`
}

// spanResponse is the wire-shape for a single span. JSONB columns are
// passed through as raw JSON so we don't pay an unmarshal/remarshal
// trip; trace/span/parent IDs are hex-encoded.
type spanResponse struct {
	TraceID                string          `json:"trace_id"`
	SpanID                 string          `json:"span_id"`
	ParentSpanID           string          `json:"parent_span_id,omitempty"`
	TraceState             string          `json:"trace_state,omitempty"`
	Flags                  uint32          `json:"flags"`
	Name                   string          `json:"name"`
	Kind                   int16           `json:"kind"`
	ServiceName            string          `json:"service_name"`
	StartTime              time.Time       `json:"start_time"`
	EndTime                time.Time       `json:"end_time"`
	DurationNS             int64           `json:"duration_ns"`
	StatusCode             int16           `json:"status_code"`
	StatusMessage          string          `json:"status_message,omitempty"`
	Attributes             json.RawMessage `json:"attributes,omitempty"`
	Resource               json.RawMessage `json:"resource,omitempty"`
	Events                 json.RawMessage `json:"events,omitempty"`
	DroppedAttributesCount uint32          `json:"dropped_attributes_count,omitempty"`
	DroppedEventsCount     uint32          `json:"dropped_events_count,omitempty"`
	DroppedLinksCount      uint32          `json:"dropped_links_count,omitempty"`
}

// ListTraces returns recent trace summaries scoped to the authenticated
// user's tenant.
// GET /api/v1/traces?service=&since=&limit=
func (h *TracesAPIHandler) ListTraces(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "authentication required")
	}

	limit, err := parseLimit(c.QueryParam("limit"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	since, err := parseSince(c.QueryParam("since"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	service := c.QueryParam("service")

	summaries, err := h.repo.ListRecentTraces(c.Request().Context(), userID, since, service, limit)
	if err != nil {
		h.logger.Error("traces api: ListRecentTraces failed", slog.String("error", err.Error()))
		return errJSON(c, http.StatusInternalServerError, "failed to list traces")
	}

	out := make([]traceSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, traceSummaryResponse{
			TraceID:     hex.EncodeToString(s.TraceID),
			StartTime:   s.StartTime,
			DurationNS:  s.DurationNS,
			SpanCount:   s.SpanCount,
			HasError:    s.HasError,
			ServiceName: s.ServiceName,
			RootName:    s.RootName,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

// GetTrace returns every span for a given trace_id (scoped to the
// authenticated user's tenant), ordered by start_time.
// GET /api/v1/traces/:trace_id
func (h *TracesAPIHandler) GetTrace(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "authentication required")
	}

	raw := c.Param("trace_id")
	traceID, err := hex.DecodeString(raw)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "trace_id must be hex-encoded")
	}
	if len(traceID) != traceIDByteLength {
		return errJSON(c, http.StatusBadRequest, "trace_id must decode to 16 bytes")
	}

	spans, err := h.repo.GetByTraceID(c.Request().Context(), userID, traceID)
	if err != nil {
		h.logger.Error("traces api: GetByTraceID failed", slog.String("error", err.Error()))
		return errJSON(c, http.StatusInternalServerError, "failed to fetch trace")
	}
	if len(spans) == 0 {
		return errJSON(c, http.StatusNotFound, "trace not found")
	}

	out := make([]spanResponse, 0, len(spans))
	for _, s := range spans {
		out = append(out, toSpanResponse(s))
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return defaultTraceListLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	if n > maxTraceListLimit {
		return maxTraceListLimit, nil
	}
	return n, nil
}

func parseSince(raw string) (time.Time, error) {
	if raw == "" {
		return time.Now().Add(-defaultListLookback), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.New("since must be RFC3339 timestamp")
	}
	return t, nil
}

func toSpanResponse(s *domain.Span) spanResponse {
	return spanResponse{
		TraceID:                hex.EncodeToString(s.TraceID),
		SpanID:                 hex.EncodeToString(s.SpanID),
		ParentSpanID:           hex.EncodeToString(s.ParentSpanID),
		TraceState:             s.TraceState,
		Flags:                  s.Flags,
		Name:                   s.Name,
		Kind:                   int16(s.Kind),
		ServiceName:            s.ServiceName,
		StartTime:              s.StartTime,
		EndTime:                s.EndTime,
		DurationNS:             s.DurationNS,
		StatusCode:             int16(s.StatusCode),
		StatusMessage:          s.StatusMessage,
		Attributes:             json.RawMessage(s.Attributes),
		Resource:               json.RawMessage(s.Resource),
		Events:                 json.RawMessage(s.Events),
		DroppedAttributesCount: s.DroppedAttributesCount,
		DroppedEventsCount:     s.DroppedEventsCount,
		DroppedLinksCount:      s.DroppedLinksCount,
	}
}
