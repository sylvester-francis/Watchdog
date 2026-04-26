package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// decodeLogs walks an OTLP ExportLogsServiceRequest and produces one
// domain.LogRecord per OTLP LogRecord. Records whose serialized
// footprint (body + JSONB blobs + severity text) exceeds maxRecordBytes
// are dropped; the second return is the dropped count for the
// PartialSuccess response.
//
// Body is rendered to a string: AnyValue_StringValue passes through
// unchanged; structured bodies are JSON-marshalled.
func decodeLogs(req *collogspb.ExportLogsServiceRequest, maxRecordBytes int) ([]*domain.LogRecord, int, error) {
	if req == nil {
		return nil, 0, nil
	}

	var out []*domain.LogRecord
	dropped := 0

	for _, rl := range req.GetResourceLogs() {
		serviceName := extractServiceName(rl.GetResource())
		resourceJSON, err := marshalAttrs(rl.GetResource().GetAttributes())
		if err != nil {
			return nil, 0, fmt.Errorf("marshal resource attributes: %w", err)
		}

		for _, sl := range rl.GetScopeLogs() {
			for _, lr := range sl.GetLogRecords() {
				dr, derr := buildLogRecord(lr, serviceName, resourceJSON)
				if derr != nil {
					return nil, 0, derr
				}
				if logRecordFootprint(dr) > maxRecordBytes {
					dropped++
					continue
				}
				out = append(out, dr)
			}
		}
	}

	return out, dropped, nil
}

func buildLogRecord(lr *logspb.LogRecord, serviceName string, resourceJSON []byte) (*domain.LogRecord, error) {
	attrJSON, err := marshalAttrs(lr.GetAttributes())
	if err != nil {
		return nil, fmt.Errorf("marshal log attributes: %w", err)
	}

	body, err := renderBody(lr.GetBody())
	if err != nil {
		return nil, fmt.Errorf("render log body: %w", err)
	}

	observed := time.Unix(0, int64(lr.GetObservedTimeUnixNano())).UTC()
	timestamp := observed
	if lr.GetTimeUnixNano() > 0 {
		timestamp = time.Unix(0, int64(lr.GetTimeUnixNano())).UTC()
	}

	return &domain.LogRecord{
		Timestamp:              timestamp,
		ObservedTimestamp:      observed,
		TraceID:                lr.GetTraceId(),
		SpanID:                 lr.GetSpanId(),
		SeverityNumber:         domain.SeverityNumber(lr.GetSeverityNumber()),
		SeverityText:           lr.GetSeverityText(),
		Body:                   body,
		ServiceName:            serviceName,
		Resource:               resourceJSON,
		Attributes:             attrJSON,
		DroppedAttributesCount: lr.GetDroppedAttributesCount(),
		Flags:                  lr.GetFlags(),
	}, nil
}

func renderBody(v *commonpb.AnyValue) (string, error) {
	if v == nil {
		return "", nil
	}
	if sv, ok := v.GetValue().(*commonpb.AnyValue_StringValue); ok {
		return sv.StringValue, nil
	}
	rendered := anyValue(v)
	if rendered == nil {
		return "", nil
	}
	out, err := json.Marshal(rendered)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func logRecordFootprint(r *domain.LogRecord) int {
	return len(r.Body) + len(r.SeverityText) + len(r.ServiceName) +
		len(r.Attributes) + len(r.Resource)
}
