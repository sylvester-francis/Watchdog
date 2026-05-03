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
	defaultLogListLimit = 100
	maxLogListLimit     = 500
	defaultLogLookback  = 1 * time.Hour
	spanIDByteLength    = 8
)

// LogsAPIHandler serves the read-side log API the explorer UI consumes.
// Lives under /api/v1/logs with the same hybrid auth as the rest of the
// v1 API (session OR bearer token).
type LogsAPIHandler struct {
	repo   ports.LogRecordRepository
	logger *slog.Logger
}

// NewLogsAPIHandler constructs a LogsAPIHandler.
func NewLogsAPIHandler(repo ports.LogRecordRepository, logger *slog.Logger) *LogsAPIHandler {
	return &LogsAPIHandler{repo: repo, logger: logger}
}

// logRecordResponse is the wire-shape for a single log record. JSONB
// columns are passed through as raw JSON; trace/span IDs are hex-encoded
// and omitted when empty.
type logRecordResponse struct {
	Timestamp              time.Time       `json:"timestamp"`
	ObservedTimestamp      time.Time       `json:"observed_timestamp"`
	TraceID                string          `json:"trace_id,omitempty"`
	SpanID                 string          `json:"span_id,omitempty"`
	SeverityNumber         int16           `json:"severity_number"`
	SeverityText           string          `json:"severity_text,omitempty"`
	Body                   string          `json:"body"`
	ServiceName            string          `json:"service_name"`
	Resource               json.RawMessage `json:"resource,omitempty"`
	Attributes             json.RawMessage `json:"attributes,omitempty"`
	DroppedAttributesCount uint32          `json:"dropped_attributes_count,omitempty"`
	Flags                  uint32          `json:"flags,omitempty"`
}

// ListLogs returns recent log records scoped to the authenticated
// user's tenant. trace_id and span_id correlate to OTLP IDs and are
// optional; when present they must decode to exactly 16 / 8 bytes.
// `before` is an optional RFC3339 cursor for keyset pagination —
// returns log records strictly older than that timestamp.
// GET /api/v1/logs?service=&severity=&since=&trace_id=&span_id=&before=&limit=
func (h *LogsAPIHandler) ListLogs(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "authentication required")
	}

	limit, err := parseLogLimit(c.QueryParam("limit"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	since, err := parseLogSince(c.QueryParam("since"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	before, err := parseBefore(c.QueryParam("before"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	traceID, err := parseHexID(c.QueryParam("trace_id"), traceIDByteLength, "trace_id")
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	spanID, err := parseHexID(c.QueryParam("span_id"), spanIDByteLength, "span_id")
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	records, err := h.repo.ListRecent(c.Request().Context(), userID, since,
		c.QueryParam("service"), c.QueryParam("severity"), traceID, spanID, before, limit)
	if err != nil {
		h.logger.Error("logs api: ListRecent failed", slog.String("error", err.Error()))
		return errJSON(c, http.StatusInternalServerError, "failed to list log records")
	}

	out := make([]logRecordResponse, 0, len(records))
	for _, r := range records {
		out = append(out, toLogRecordResponse(r))
	}
	return c.JSON(http.StatusOK, map[string]any{"data": out})
}

func parseLogLimit(raw string) (int, error) {
	if raw == "" {
		return defaultLogListLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	if n > maxLogListLimit {
		return maxLogListLimit, nil
	}
	return n, nil
}

func parseLogSince(raw string) (time.Time, error) {
	if raw == "" {
		return time.Now().Add(-defaultLogLookback), nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.New("since must be RFC3339 timestamp")
	}
	return t, nil
}

// parseHexID decodes a hex-encoded ID and verifies its length. Returns
// nil with no error when the input is empty (filter not requested).
func parseHexID(raw string, wantBytes int, name string) ([]byte, error) {
	if raw == "" {
		return nil, nil
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, errors.New(name + " must be hex-encoded")
	}
	if len(decoded) != wantBytes {
		return nil, errors.New(name + " must decode to " + strconv.Itoa(wantBytes) + " bytes")
	}
	return decoded, nil
}

func toLogRecordResponse(r *domain.LogRecord) logRecordResponse {
	return logRecordResponse{
		Timestamp:              r.Timestamp,
		ObservedTimestamp:      r.ObservedTimestamp,
		TraceID:                hexOrEmpty(r.TraceID),
		SpanID:                 hexOrEmpty(r.SpanID),
		SeverityNumber:         int16(r.SeverityNumber),
		SeverityText:           r.SeverityText,
		Body:                   r.Body,
		ServiceName:            r.ServiceName,
		Resource:               json.RawMessage(r.Resource),
		Attributes:             json.RawMessage(r.Attributes),
		DroppedAttributesCount: r.DroppedAttributesCount,
		Flags:                  r.Flags,
	}
}

func hexOrEmpty(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}
