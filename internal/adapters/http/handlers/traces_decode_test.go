package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

func TestDecodeTraces_EmptyRequest(t *testing.T) {
	spans, dropped, err := decodeTraces(&coltracepb.ExportTraceServiceRequest{}, 64*1024)
	require.NoError(t, err)
	assert.Empty(t, spans)
	assert.Zero(t, dropped)
}

func TestDecodeTraces_SingleRootSpan(t *testing.T) {
	traceID := bytes16(0x01)
	spanID := bytes8(0x02)
	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: resourceWithService("checkout-api"),
				ScopeSpans: []*tracepb.ScopeSpans{
					{
						Spans: []*tracepb.Span{
							{
								TraceId:           traceID,
								SpanId:            spanID,
								Name:              "GET /healthz",
								Kind:              tracepb.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: 1_700_000_000_000_000_000,
								EndTimeUnixNano:   1_700_000_000_500_000_000,
							},
						},
					},
				},
			},
		},
	}

	spans, dropped, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	assert.Zero(t, dropped)

	got := spans[0]
	assert.Equal(t, traceID, got.TraceID)
	assert.Equal(t, spanID, got.SpanID)
	assert.Empty(t, got.ParentSpanID, "root span has no parent")
	assert.Equal(t, "GET /healthz", got.Name)
	assert.Equal(t, "checkout-api", got.ServiceName)
	assert.EqualValues(t, 2, got.Kind)
	assert.EqualValues(t, 500_000_000, got.DurationNS)
}

func TestDecodeTraces_PreservesParentSpanID(t *testing.T) {
	parent := bytes8(0xAA)
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		ParentSpanId:      parent,
		Name:              "child",
		StartTimeUnixNano: 1_700_000_000_000_000_000,
		EndTimeUnixNano:   1_700_000_000_001_000_000,
	}, "svc")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	assert.Equal(t, parent, spans[0].ParentSpanID)
}

func TestDecodeTraces_FallsBackToUnknownServiceName(t *testing.T) {
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "no-resource",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
	}, "")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	assert.Equal(t, "unknown", spans[0].ServiceName)
}

func TestDecodeTraces_SerializesScalarAttributesToJSON(t *testing.T) {
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "with-attrs",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
		Attributes: []*commonpb.KeyValue{
			kvString("http.method", "GET"),
			kvInt("http.status_code", 200),
			kvBool("retry", true),
			kvDouble("latency_ms", 12.5),
		},
	}, "svc")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal(spans[0].Attributes, &attrs))
	assert.Equal(t, "GET", attrs["http.method"])
	assert.EqualValues(t, 200, attrs["http.status_code"])
	assert.Equal(t, true, attrs["retry"])
	assert.InDelta(t, 12.5, attrs["latency_ms"], 0.0001)
}

func TestDecodeTraces_SerializesNestedAttributes(t *testing.T) {
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "nested",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
		Attributes: []*commonpb.KeyValue{
			{
				Key: "nested",
				Value: &commonpb.AnyValue{
					Value: &commonpb.AnyValue_KvlistValue{
						KvlistValue: &commonpb.KeyValueList{
							Values: []*commonpb.KeyValue{
								kvString("inner", "value"),
							},
						},
					},
				},
			},
			{
				Key: "list",
				Value: &commonpb.AnyValue{
					Value: &commonpb.AnyValue_ArrayValue{
						ArrayValue: &commonpb.ArrayValue{
							Values: []*commonpb.AnyValue{
								{Value: &commonpb.AnyValue_StringValue{StringValue: "a"}},
								{Value: &commonpb.AnyValue_StringValue{StringValue: "b"}},
							},
						},
					},
				},
			},
		},
	}, "svc")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal(spans[0].Attributes, &attrs))
	assert.Equal(t, map[string]any{"inner": "value"}, attrs["nested"])
	assert.Equal(t, []any{"a", "b"}, attrs["list"])
}

func TestDecodeTraces_CapturesEvents(t *testing.T) {
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "with-events",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   100,
		Events: []*tracepb.Span_Event{
			{
				Name:         "exception",
				TimeUnixNano: 50,
				Attributes: []*commonpb.KeyValue{
					kvString("exception.type", "RuntimeError"),
				},
			},
		},
	}, "svc")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)

	var events []map[string]any
	require.NoError(t, json.Unmarshal(spans[0].Events, &events))
	require.Len(t, events, 1)
	assert.Equal(t, "exception", events[0]["name"])
	assert.EqualValues(t, 50, events[0]["time_unix_nano"])
	assert.Equal(t, map[string]any{"exception.type": "RuntimeError"}, events[0]["attributes"])
}

func TestDecodeTraces_CapturesStatus(t *testing.T) {
	req := requestWithSpan(&tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x02),
		Name:              "errored",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
		Status: &tracepb.Status{
			Code:    tracepb.Status_STATUS_CODE_ERROR,
			Message: "boom",
		},
	}, "svc")

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	assert.EqualValues(t, 2, spans[0].StatusCode)
	assert.Equal(t, "boom", spans[0].StatusMessage)
}

func TestDecodeTraces_DropsOversizedSpans(t *testing.T) {
	bigValue := strings.Repeat("x", 70_000)
	oversize := &tracepb.Span{
		TraceId:           bytes16(0xFE),
		SpanId:            bytes8(0xFE),
		Name:              "huge",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
		Attributes:        []*commonpb.KeyValue{kvString("blob", bigValue)},
	}
	keep := &tracepb.Span{
		TraceId:           bytes16(0x01),
		SpanId:            bytes8(0x01),
		Name:              "small",
		StartTimeUnixNano: 3,
		EndTimeUnixNano:   4,
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

	spans, dropped, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	assert.Equal(t, 1, dropped)
	require.Len(t, spans, 1)
	assert.Equal(t, "small", spans[0].Name, "small span survives, oversize is dropped")
}

func TestDecodeTraces_FlattensMultipleResourceAndScopeSpans(t *testing.T) {
	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: resourceWithService("api"),
				ScopeSpans: []*tracepb.ScopeSpans{
					{Spans: []*tracepb.Span{minimalSpan(0x01)}},
					{Spans: []*tracepb.Span{minimalSpan(0x02)}},
				},
			},
			{
				Resource: resourceWithService("worker"),
				ScopeSpans: []*tracepb.ScopeSpans{
					{Spans: []*tracepb.Span{minimalSpan(0x03)}},
				},
			},
		},
	}

	spans, _, err := decodeTraces(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, spans, 3)
	assert.Equal(t, "api", spans[0].ServiceName)
	assert.Equal(t, "api", spans[1].ServiceName)
	assert.Equal(t, "worker", spans[2].ServiceName)
}

// --- helpers ---

func bytes16(b byte) []byte { return append(make([]byte, 0, 16), bytesN(16, b)...) }
func bytes8(b byte) []byte  { return append(make([]byte, 0, 8), bytesN(8, b)...) }
func bytesN(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func kvString(k, v string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: v}}}
}
func kvInt(k string, v int64) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: v}}}
}
func kvBool(k string, v bool) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: v}}}
}
func kvDouble(k string, v float64) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: k, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: v}}}
}

func resourceWithService(name string) *resourcepb.Resource {
	if name == "" {
		return &resourcepb.Resource{}
	}
	return &resourcepb.Resource{Attributes: []*commonpb.KeyValue{kvString("service.name", name)}}
}

func requestWithSpan(s *tracepb.Span, service string) *coltracepb.ExportTraceServiceRequest {
	return &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: resourceWithService(service),
				ScopeSpans: []*tracepb.ScopeSpans{
					{Spans: []*tracepb.Span{s}},
				},
			},
		},
	}
}

func minimalSpan(id byte) *tracepb.Span {
	return &tracepb.Span{
		TraceId:           bytes16(id),
		SpanId:            bytes8(id),
		Name:              "span",
		StartTimeUnixNano: 1,
		EndTimeUnixNano:   2,
	}
}
