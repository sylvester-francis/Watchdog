package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
)

func TestDecodeLogs_NilRequest(t *testing.T) {
	records, dropped, err := decodeLogs(nil, 64*1024)
	require.NoError(t, err)
	assert.Nil(t, records)
	assert.Zero(t, dropped)
}

func TestDecodeLogs_StringBody(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				Resource: resourceWithService("api"),
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								TimeUnixNano:         1_700_000_000_000_000_000,
								ObservedTimeUnixNano: 1_700_000_000_500_000_000,
								SeverityNumber:       logspb.SeverityNumber_SEVERITY_NUMBER_INFO,
								SeverityText:         "INFO",
								Body:                 anyValueString("hello world"),
								TraceId:              bytes16(0x01),
								SpanId:               bytes8(0x02),
								Flags:                1,
							},
						},
					},
				},
			},
		},
	}

	records, dropped, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	assert.Zero(t, dropped)
	require.Len(t, records, 1)

	rec := records[0]
	assert.Equal(t, "hello world", rec.Body)
	assert.Equal(t, "INFO", rec.SeverityText)
	assert.EqualValues(t, 9, rec.SeverityNumber)
	assert.Equal(t, "api", rec.ServiceName)
	assert.Equal(t, bytes16(0x01), rec.TraceID)
	assert.Equal(t, bytes8(0x02), rec.SpanID)
	assert.EqualValues(t, 1, rec.Flags)
}

func TestDecodeLogs_StructuredBodyMarshalsToJSON(t *testing.T) {
	body := &commonpb.AnyValue{Value: &commonpb.AnyValue_KvlistValue{
		KvlistValue: &commonpb.KeyValueList{Values: []*commonpb.KeyValue{
			kvString("event", "checkout"),
			kvInt("amount", 4200),
		}},
	}}
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService("api"),
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: body,
				}},
			}},
		}},
	}

	records, _, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, records, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(records[0].Body), &got))
	assert.Equal(t, "checkout", got["event"])
	assert.EqualValues(t, 4200, got["amount"])
}

func TestDecodeLogs_AttributesAndResourcePassThrough(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService("api"),
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano:         1,
					ObservedTimeUnixNano: 1,
					Body:                 anyValueString("x"),
					Attributes: []*commonpb.KeyValue{
						kvString("user.id", "u-1"),
						kvBool("retry", true),
					},
				}},
			}},
		}},
	}

	records, _, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, records, 1)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal(records[0].Attributes, &attrs))
	assert.Equal(t, "u-1", attrs["user.id"])
	assert.Equal(t, true, attrs["retry"])

	var resource map[string]any
	require.NoError(t, json.Unmarshal(records[0].Resource, &resource))
	assert.Equal(t, "api", resource["service.name"])
}

func TestDecodeLogs_UsesObservedTimeWhenTimeMissing(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService("api"),
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano:         0,
					ObservedTimeUnixNano: 1_700_000_000_000_000_000,
					Body:                 anyValueString("x"),
				}},
			}},
		}},
	}

	records, _, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, records[0].ObservedTimestamp, records[0].Timestamp,
		"timestamp falls back to observed timestamp when time_unix_nano=0")
}

func TestDecodeLogs_DropsOversizedRecords(t *testing.T) {
	bigBody := strings.Repeat("z", 70_000)
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			Resource: resourceWithService("api"),
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{
					{TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: anyValueString(bigBody)},
					{TimeUnixNano: 2, ObservedTimeUnixNano: 2, Body: anyValueString("ok")},
				},
			}},
		}},
	}

	records, dropped, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	assert.Equal(t, 1, dropped)
	require.Len(t, records, 1)
	assert.Equal(t, "ok", records[0].Body)
}

func TestDecodeLogs_MissingServiceNameDefaultsUnknown(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{
			ScopeLogs: []*logspb.ScopeLogs{{
				LogRecords: []*logspb.LogRecord{{
					TimeUnixNano: 1, ObservedTimeUnixNano: 1, Body: anyValueString("x"),
				}},
			}},
		}},
	}

	records, _, err := decodeLogs(req, 64*1024)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, unknownServiceName, records[0].ServiceName)
}

func anyValueString(s string) *commonpb.AnyValue {
	return &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: s}}
}
