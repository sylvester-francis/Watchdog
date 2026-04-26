package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"

	"github.com/sylvester-francis/watchdog/core/ports"
)

// Per-record body+attributes JSONB cap. Records exceeding this footprint
// are dropped at decode time and reported via PartialSuccess so noisy
// senders can't bloat the log_records hypertable indefinitely.
const maxLogRecordBytes = 64 * 1024

// Hard cap on the request body before protobuf decode. Aligned with the
// 1 MB global BodyLimit so we don't accept anything Echo would have
// already rejected.
const maxLogRequestBytes = 1 * 1024 * 1024

// LogsHandler accepts OTLP/HTTP protobuf log exports at /v1/logs and
// writes records through to the LogRecordRepository.
type LogsHandler struct {
	repo   ports.LogRecordRepository
	logger *slog.Logger
}

// NewLogsHandler constructs a LogsHandler.
func NewLogsHandler(repo ports.LogRecordRepository, logger *slog.Logger) *LogsHandler {
	return &LogsHandler{repo: repo, logger: logger}
}

// Handle serves POST /v1/logs.
func (h *LogsHandler) Handle(c echo.Context) error {
	if ct := c.Request().Header.Get(echo.HeaderContentType); ct != "application/x-protobuf" {
		return c.JSON(http.StatusUnsupportedMediaType, map[string]string{
			"error": "Content-Type must be application/x-protobuf",
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

	var req collogspb.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid OTLP protobuf"})
	}

	records, dropped, err := decodeLogs(&req, maxLogRecordBytes)
	if err != nil {
		h.logger.Error("logs: decode failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "decode log records"})
	}

	if err := h.repo.InsertBatch(c.Request().Context(), records); err != nil {
		h.logger.Error("logs: insert failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "store log records"})
	}

	resp := &collogspb.ExportLogsServiceResponse{}
	if dropped > 0 {
		resp.PartialSuccess = &collogspb.ExportLogsPartialSuccess{
			RejectedLogRecords: int64(dropped),
			ErrorMessage:       fmt.Sprintf("%d log record(s) dropped: per-record footprint exceeded %d bytes", dropped, maxLogRecordBytes),
		}
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "marshal response"})
	}
	return c.Blob(http.StatusOK, "application/x-protobuf", out)
}
