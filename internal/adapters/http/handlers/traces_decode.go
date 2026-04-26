package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/sylvester-francis/watchdog/core/domain"
)

const unknownServiceName = "unknown"

// decodeTraces walks an OTLP ExportTraceServiceRequest and produces
// one domain.Span per OTLP span. Spans whose serialized footprint
// (name + service + JSONB blobs + status fields) exceeds maxSpanBytes
// are dropped; the second return is the dropped count for the
// PartialSuccess response.
//
// The function is deliberately tolerant: malformed values inside an
// individual span don't fail the whole batch, they just yield empty
// JSONB or default fields. The caller is the boundary that already
// validated the protobuf decode succeeded.
func decodeTraces(req *coltracepb.ExportTraceServiceRequest, maxSpanBytes int) ([]*domain.Span, int, error) {
	if req == nil {
		return nil, 0, nil
	}

	var out []*domain.Span
	dropped := 0

	for _, rs := range req.GetResourceSpans() {
		serviceName := extractServiceName(rs.GetResource())
		resourceJSON, err := marshalAttrs(rs.GetResource().GetAttributes())
		if err != nil {
			return nil, 0, fmt.Errorf("marshal resource attributes: %w", err)
		}

		for _, ss := range rs.GetScopeSpans() {
			for _, sp := range ss.GetSpans() {
				ds, derr := buildSpan(sp, serviceName, resourceJSON)
				if derr != nil {
					return nil, 0, derr
				}
				if spanFootprint(ds) > maxSpanBytes {
					dropped++
					continue
				}
				out = append(out, ds)
			}
		}
	}

	return out, dropped, nil
}

func buildSpan(sp *tracepb.Span, serviceName string, resourceJSON []byte) (*domain.Span, error) {
	attrJSON, err := marshalAttrs(sp.GetAttributes())
	if err != nil {
		return nil, fmt.Errorf("marshal span attributes: %w", err)
	}
	eventsJSON, err := marshalEvents(sp.GetEvents())
	if err != nil {
		return nil, fmt.Errorf("marshal span events: %w", err)
	}

	start := time.Unix(0, int64(sp.GetStartTimeUnixNano())).UTC()
	end := time.Unix(0, int64(sp.GetEndTimeUnixNano())).UTC()

	status := sp.GetStatus()
	var statusCode domain.SpanStatusCode
	var statusMessage string
	if status != nil {
		statusCode = domain.SpanStatusCode(status.GetCode())
		statusMessage = status.GetMessage()
	}

	return &domain.Span{
		TraceID:                sp.GetTraceId(),
		SpanID:                 sp.GetSpanId(),
		ParentSpanID:           sp.GetParentSpanId(),
		TraceState:             sp.GetTraceState(),
		Flags:                  sp.GetFlags(),
		Name:                   sp.GetName(),
		Kind:                   domain.SpanKind(sp.GetKind()),
		ServiceName:            serviceName,
		StartTime:              start,
		EndTime:                end,
		DurationNS:             int64(sp.GetEndTimeUnixNano()) - int64(sp.GetStartTimeUnixNano()),
		StatusCode:             statusCode,
		StatusMessage:          statusMessage,
		Attributes:             attrJSON,
		Resource:               resourceJSON,
		Events:                 eventsJSON,
		DroppedAttributesCount: sp.GetDroppedAttributesCount(),
		DroppedEventsCount:     sp.GetDroppedEventsCount(),
		DroppedLinksCount:      sp.GetDroppedLinksCount(),
	}, nil
}

func spanFootprint(s *domain.Span) int {
	return len(s.Name) + len(s.ServiceName) + len(s.TraceState) + len(s.StatusMessage) +
		len(s.Attributes) + len(s.Resource) + len(s.Events)
}

func extractServiceName(r *resourcepb.Resource) string {
	for _, kv := range r.GetAttributes() {
		if kv.GetKey() == "service.name" {
			if sv, ok := kv.GetValue().GetValue().(*commonpb.AnyValue_StringValue); ok && sv.StringValue != "" {
				return sv.StringValue
			}
		}
	}
	return unknownServiceName
}

func marshalAttrs(kvs []*commonpb.KeyValue) ([]byte, error) {
	if len(kvs) == 0 {
		return nil, nil
	}
	return json.Marshal(kvListToMap(kvs))
}

func marshalEvents(events []*tracepb.Span_Event) ([]byte, error) {
	if len(events) == 0 {
		return nil, nil
	}
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		entry := map[string]any{
			"name":           e.GetName(),
			"time_unix_nano": e.GetTimeUnixNano(),
		}
		if attrs := e.GetAttributes(); len(attrs) > 0 {
			entry["attributes"] = kvListToMap(attrs)
		}
		out = append(out, entry)
	}
	return json.Marshal(out)
}

func kvListToMap(kvs []*commonpb.KeyValue) map[string]any {
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		out[kv.GetKey()] = anyValue(kv.GetValue())
	}
	return out
}

func anyValue(v *commonpb.AnyValue) any {
	if v == nil {
		return nil
	}
	switch x := v.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return x.StringValue
	case *commonpb.AnyValue_BoolValue:
		return x.BoolValue
	case *commonpb.AnyValue_IntValue:
		return x.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return x.DoubleValue
	case *commonpb.AnyValue_BytesValue:
		return x.BytesValue
	case *commonpb.AnyValue_ArrayValue:
		arr := x.ArrayValue.GetValues()
		out := make([]any, len(arr))
		for i, item := range arr {
			out[i] = anyValue(item)
		}
		return out
	case *commonpb.AnyValue_KvlistValue:
		return kvListToMap(x.KvlistValue.GetValues())
	}
	return nil
}
