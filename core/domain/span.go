package domain

import (
	"time"

	"github.com/google/uuid"
)

// SpanKind mirrors the OTLP Span.SpanKind enum. Stored as the raw int
// (matching the OTLP wire value) so we don't have to map back when
// re-serializing for clients.
type SpanKind int16

const (
	SpanKindUnspecified SpanKind = 0
	SpanKindInternal    SpanKind = 1
	SpanKindServer      SpanKind = 2
	SpanKindClient      SpanKind = 3
	SpanKindProducer    SpanKind = 4
	SpanKindConsumer    SpanKind = 5
)

// SpanStatusCode mirrors the OTLP Status.StatusCode enum.
type SpanStatusCode int16

const (
	SpanStatusUnset SpanStatusCode = 0
	SpanStatusOK    SpanStatusCode = 1
	SpanStatusError SpanStatusCode = 2
)

// TraceSummary is a per-trace digest aggregated from many spans.
// Used by the list endpoint to render the trace list without
// shipping every span over the wire.
type TraceSummary struct {
	TraceID    []byte
	StartTime  time.Time
	DurationNS int64
	SpanCount  int
	HasError   bool
}

// Span is one node of a trace, persisted to the spans hypertable.
//
// TraceID is 16 bytes, SpanID and ParentSpanID are 8 bytes; an empty
// ParentSpanID indicates a root span. Attributes/Resource/Events are
// stored as JSONB; we keep them as opaque JSON byte slices on the way
// in and out of the database to avoid re-marshalling at every layer.
//
// UserID and TenantID are stamped by the OTLP receiver from the
// authenticated request context and used at query time to scope reads
// per the project's standard isolation model.
type Span struct {
	UserID                 uuid.UUID
	TenantID               string
	TraceID                []byte
	SpanID                 []byte
	ParentSpanID           []byte
	TraceState             string
	Flags                  uint32
	Name                   string
	Kind                   SpanKind
	ServiceName            string
	StartTime              time.Time
	EndTime                time.Time
	DurationNS             int64
	StatusCode             SpanStatusCode
	StatusMessage          string
	Attributes             []byte
	Resource               []byte
	Events                 []byte
	DroppedAttributesCount uint32
	DroppedEventsCount     uint32
	DroppedLinksCount      uint32
}
