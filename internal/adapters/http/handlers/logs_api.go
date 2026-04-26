package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

const (
	defaultLogListLimit  = 100
	maxLogListLimit      = 500
	defaultLogLookback   = 1 * time.Hour
)

// LogsAPIHandler serves the read-side log API the explorer UI consumes.
// Lives under /api/v1/logs with the same hybrid auth as the rest of the
// v1 API (session OR bearer token).
type LogsAPIHandler struct {
	repo ports.LogRecordRepository
}

// NewLogsAPIHandler constructs a LogsAPIHandler.
func NewLogsAPIHandler(repo ports.LogRecordRepository) *LogsAPIHandler {
	return &LogsAPIHandler{repo: repo}
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

// ListLogs returns recent log records.
// GET /api/v1/logs?service=&severity=&since=&limit=
func (h *LogsAPIHandler) ListLogs(c echo.Context) error {
	limit, err := parseLogLimit(c.QueryParam("limit"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	since, err := parseLogSince(c.QueryParam("since"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	records, err := h.repo.ListRecent(c.Request().Context(), since,
		c.QueryParam("service"), c.QueryParam("severity"), limit)
	if err != nil {
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
