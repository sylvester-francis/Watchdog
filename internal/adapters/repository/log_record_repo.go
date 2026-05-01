package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// LogRecordRepository persists OTLP log records to the log_records hypertable.
type LogRecordRepository struct {
	db *DB
}

// NewLogRecordRepository creates a new LogRecordRepository.
func NewLogRecordRepository(db *DB) *LogRecordRepository {
	return &LogRecordRepository{db: db}
}

// InsertBatch bulk-inserts log records via COPY. Each record must
// already carry UserID and TenantID — the receiver stamps both from
// request context before calling this method.
func (r *LogRecordRepository) InsertBatch(ctx context.Context, records []*domain.LogRecord) error {
	if len(records) == 0 {
		return nil
	}

	_, err := r.db.CopyFrom(ctx,
		pgx.Identifier{"log_records"},
		[]string{
			"user_id", "tenant_id",
			"timestamp", "observed_timestamp", "trace_id", "span_id",
			"severity_number", "severity_text", "body", "service_name",
			"resource", "attributes", "dropped_attributes_count", "flags",
		},
		pgx.CopyFromSlice(len(records), func(i int) ([]any, error) {
			r := records[i]
			return []any{
				r.UserID, r.TenantID,
				r.Timestamp, r.ObservedTimestamp, r.TraceID, r.SpanID,
				int16(r.SeverityNumber), r.SeverityText, r.Body, r.ServiceName,
				r.Resource, r.Attributes, r.DroppedAttributesCount, r.Flags,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("logRecordRepo.InsertBatch: %w", err)
	}
	return nil
}

// ListRecent returns log records scoped to (userID, tenantID-from-context)
// emitted since `since`, optionally filtered by service name and severity
// text. Empty filters match all. Results are ordered newest-first and
// capped at limit.
func (r *LogRecordRepository) ListRecent(ctx context.Context, userID uuid.UUID, since time.Time, service, severity string, limit int) ([]*domain.LogRecord, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	rows, err := q.Query(ctx, `
		SELECT timestamp, observed_timestamp, trace_id, span_id,
		       severity_number, severity_text, body, service_name,
		       resource, attributes, dropped_attributes_count, flags
		FROM log_records
		WHERE timestamp >= $1
		  AND user_id = $2
		  AND tenant_id = $3
		  AND ($4 = '' OR service_name = $4)
		  AND ($5 = '' OR severity_text = $5)
		ORDER BY timestamp DESC
		LIMIT $6`, since, userID, tenantID, service, severity, limit)
	if err != nil {
		return nil, fmt.Errorf("logRecordRepo.ListRecent: %w", err)
	}
	defer rows.Close()

	var out []*domain.LogRecord
	for rows.Next() {
		rec := &domain.LogRecord{}
		var sev int16
		if err := rows.Scan(
			&rec.Timestamp, &rec.ObservedTimestamp, &rec.TraceID, &rec.SpanID,
			&sev, &rec.SeverityText, &rec.Body, &rec.ServiceName,
			&rec.Resource, &rec.Attributes, &rec.DroppedAttributesCount, &rec.Flags,
		); err != nil {
			return nil, fmt.Errorf("logRecordRepo.ListRecent scan: %w", err)
		}
		rec.SeverityNumber = domain.SeverityNumber(sev)
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("logRecordRepo.ListRecent rows: %w", err)
	}
	return out, nil
}

// DeleteOlderThan removes every log record chunk with timestamp < cutoff.
// On a hypertable this is a chunk drop, not a row-level delete.
func (r *LogRecordRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	q := r.db.Querier(ctx)
	_, err := q.Exec(ctx, `DELETE FROM log_records WHERE timestamp < $1`, cutoff)
	if err != nil {
		return fmt.Errorf("logRecordRepo.DeleteOlderThan: %w", err)
	}
	return nil
}
