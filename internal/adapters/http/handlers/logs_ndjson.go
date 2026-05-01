package handlers

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

// Max bytes for a single NDJSON line. Aligned with the OTLP per-record
// cap so the two ingestion paths have the same noisy-sender envelope.
const maxNDJSONLineBytes = 64 * 1024

// LogsNDJSONHandler accepts a newline-delimited JSON stream at
// /v1/logs/raw — the legacy ingest path for senders that don't speak
// OTLP. One JSON object per line; bad lines are reported in the
// response without failing the whole batch.
type LogsNDJSONHandler struct {
	repo   ports.LogRecordRepository
	logger *slog.Logger
}

// NewLogsNDJSONHandler constructs a LogsNDJSONHandler.
func NewLogsNDJSONHandler(repo ports.LogRecordRepository, logger *slog.Logger) *LogsNDJSONHandler {
	return &LogsNDJSONHandler{repo: repo, logger: logger}
}

// ndjsonRecord is the wire shape of a single line. timestamp + severity
// + body + service are required; trace_id / span_id are optional hex
// strings; attributes is a free-form JSON object.
type ndjsonRecord struct {
	Timestamp  time.Time       `json:"timestamp"`
	Severity   string          `json:"severity"`
	Body       string          `json:"body"`
	Service    string          `json:"service"`
	TraceID    string          `json:"trace_id,omitempty"`
	SpanID     string          `json:"span_id,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// ndjsonResponse is the JSON envelope returned by /v1/logs/raw.
type ndjsonResponse struct {
	Accepted int      `json:"accepted"`
	Rejected int      `json:"rejected"`
	Errors   []string `json:"errors,omitempty"`
}

// Handle serves POST /v1/logs/raw.
func (h *LogsNDJSONHandler) Handle(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}
	tenantID := repository.TenantIDFromContext(c.Request().Context())

	if ct := c.Request().Header.Get(echo.HeaderContentType); ct != "application/x-ndjson" {
		return c.JSON(http.StatusUnsupportedMediaType, map[string]string{
			"error": "Content-Type must be application/x-ndjson",
		})
	}

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxLogRequestBytes+1))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "read body"})
	}
	if len(body) > maxLogRequestBytes {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
			"error": fmt.Sprintf("log export exceeds %d-byte limit", maxLogRequestBytes),
		})
	}

	records, errs := parseNDJSON(body)

	for _, r := range records {
		r.UserID = userID
		r.TenantID = tenantID
	}

	if err := h.repo.InsertBatch(c.Request().Context(), records); err != nil {
		h.logger.Error("logs/raw: insert failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "store log records"})
	}

	return c.JSON(http.StatusOK, ndjsonResponse{
		Accepted: len(records),
		Rejected: len(errs),
		Errors:   errs,
	})
}

func parseNDJSON(body []byte) ([]*domain.LogRecord, []string) {
	var records []*domain.LogRecord
	var errs []string

	scanner := bufio.NewScanner(bytes.NewReader(body))
	// Allow the scanner to read lines up to the full request cap so a
	// single oversize line doesn't abort the whole batch. Per-line size
	// is enforced below against maxNDJSONLineBytes.
	scanner.Buffer(make([]byte, 0, 4096), maxLogRequestBytes)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if len(line) > maxNDJSONLineBytes {
			errs = append(errs, fmt.Sprintf("line %d: exceeds %d bytes", lineNum, maxNDJSONLineBytes))
			continue
		}

		rec, err := decodeNDJSONLine(line)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %s", lineNum, err.Error()))
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Sprintf("scanner: %s", err.Error()))
	}

	return records, errs
}

func decodeNDJSONLine(line []byte) (*domain.LogRecord, error) {
	var raw ndjsonRecord
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if raw.Timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp is required")
	}
	if raw.Service == "" {
		return nil, fmt.Errorf("service is required")
	}
	if raw.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	rec := &domain.LogRecord{
		Timestamp:         raw.Timestamp.UTC(),
		ObservedTimestamp: raw.Timestamp.UTC(),
		SeverityText:      raw.Severity,
		SeverityNumber:    severityFromText(raw.Severity),
		Body:              raw.Body,
		ServiceName:       raw.Service,
		Attributes:        raw.Attributes,
	}

	if raw.TraceID != "" {
		tid, err := hex.DecodeString(raw.TraceID)
		if err != nil {
			return nil, fmt.Errorf("trace_id: %w", err)
		}
		rec.TraceID = tid
	}
	if raw.SpanID != "" {
		sid, err := hex.DecodeString(raw.SpanID)
		if err != nil {
			return nil, fmt.Errorf("span_id: %w", err)
		}
		rec.SpanID = sid
	}

	if logRecordFootprint(rec) > maxNDJSONLineBytes {
		return nil, fmt.Errorf("record footprint exceeds %d bytes", maxNDJSONLineBytes)
	}

	return rec, nil
}

// severityFromText maps the most common log-level strings to OTLP
// SeverityNumber. Unrecognized values stay at SeverityUnspecified —
// SeverityText is preserved verbatim so callers don't lose information.
func severityFromText(s string) domain.SeverityNumber {
	switch s {
	case "TRACE":
		return domain.SeverityTrace
	case "DEBUG":
		return domain.SeverityDebug
	case "INFO":
		return domain.SeverityInfo
	case "WARN", "WARNING":
		return domain.SeverityWarn
	case "ERROR":
		return domain.SeverityError
	case "FATAL", "CRITICAL":
		return domain.SeverityFatal
	}
	return domain.SeverityUnspecified
}
