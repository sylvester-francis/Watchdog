package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/repository"
)

// Per-span attribute+events JSONB cap. Spans exceeding this footprint
// are dropped at decode time and reported via PartialSuccess so noisy
// senders can't bloat the spans hypertable indefinitely.
const maxSpanBytes = 64 * 1024

// Hard cap on the request body before protobuf decode. Aligned with the
// 1 MB global BodyLimit so we don't accept anything Echo would have
// already rejected. Plenty of headroom for typical trace exports
// (50-100 KB / batch); senders with larger batches should split.
const maxTraceRequestBytes = 1 * 1024 * 1024

// TracesHandler accepts OTLP/HTTP protobuf trace exports at /v1/traces
// and writes spans through to the SpanRepository.
type TracesHandler struct {
	repo   ports.SpanRepository
	logger *slog.Logger
}

// NewTracesHandler constructs a TracesHandler.
func NewTracesHandler(repo ports.SpanRepository, logger *slog.Logger) *TracesHandler {
	return &TracesHandler{repo: repo, logger: logger}
}

// Handle serves POST /v1/traces.
func (h *TracesHandler) Handle(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}
	tenantID := repository.TenantIDFromContext(c.Request().Context())

	if ct := c.Request().Header.Get(echo.HeaderContentType); ct != "application/x-protobuf" {
		return c.JSON(http.StatusUnsupportedMediaType, map[string]string{
			"error": "Content-Type must be application/x-protobuf",
		})
	}

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxTraceRequestBytes+1))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "read body"})
	}
	if len(body) > maxTraceRequestBytes {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
			"error": fmt.Sprintf("trace export exceeds %d-byte limit", maxTraceRequestBytes),
		})
	}

	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid OTLP protobuf"})
	}

	spans, dropped, err := decodeTraces(&req, maxSpanBytes)
	if err != nil {
		h.logger.Error("traces: decode failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "decode spans"})
	}

	for _, s := range spans {
		s.UserID = userID
		s.TenantID = tenantID
	}

	if err := h.repo.InsertBatch(c.Request().Context(), spans); err != nil {
		h.logger.Error("traces: insert failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "store spans"})
	}

	resp := &coltracepb.ExportTraceServiceResponse{}
	if dropped > 0 {
		resp.PartialSuccess = &coltracepb.ExportTracePartialSuccess{
			RejectedSpans: int64(dropped),
			ErrorMessage:  fmt.Sprintf("%d span(s) dropped: per-span footprint exceeded %d bytes", dropped, maxSpanBytes),
		}
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "marshal response"})
	}
	return c.Blob(http.StatusOK, "application/x-protobuf", out)
}
