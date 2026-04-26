package domain

import "time"

// SeverityNumber mirrors the OTLP LogRecord.SeverityNumber enum. Stored
// as the raw int (matching the OTLP wire value) so we don't have to map
// back when re-serializing for clients.
type SeverityNumber int16

const (
	SeverityUnspecified SeverityNumber = 0
	SeverityTrace       SeverityNumber = 1
	SeverityDebug       SeverityNumber = 5
	SeverityInfo        SeverityNumber = 9
	SeverityWarn        SeverityNumber = 13
	SeverityError       SeverityNumber = 17
	SeverityFatal       SeverityNumber = 21
)

// LogRecord is one log line, persisted to the log_records hypertable.
//
// TraceID is 16 bytes when present, SpanID is 8 bytes; both are nil for
// records emitted outside an active trace context. Body is the rendered
// log message; structured fields live in Attributes (JSONB). Resource
// captures process-level attributes (service.name, host, etc.). Flags
// is the OTLP `flags` field — its lower 8 bits carry W3C trace flags.
type LogRecord struct {
	Timestamp              time.Time
	ObservedTimestamp      time.Time
	TraceID                []byte
	SpanID                 []byte
	SeverityNumber         SeverityNumber
	SeverityText           string
	Body                   string
	ServiceName            string
	Resource               []byte
	Attributes             []byte
	DroppedAttributesCount uint32
	Flags                  uint32
}
