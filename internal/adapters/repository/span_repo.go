package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// SpanRepository persists OTLP spans to the spans hypertable.
type SpanRepository struct {
	db *DB
}

// NewSpanRepository creates a new SpanRepository.
func NewSpanRepository(db *DB) *SpanRepository {
	return &SpanRepository{db: db}
}

// InsertBatch bulk-inserts spans via COPY. Caller is responsible for any
// per-span filtering (size cap, validation) before reaching this layer.
func (r *SpanRepository) InsertBatch(ctx context.Context, spans []*domain.Span) error {
	if len(spans) == 0 {
		return nil
	}

	_, err := r.db.CopyFrom(ctx,
		pgx.Identifier{"spans"},
		[]string{
			"start_time", "trace_id", "span_id", "parent_span_id",
			"trace_state", "flags", "name", "kind", "service_name",
			"end_time", "duration_ns", "status_code", "status_message",
			"attributes", "resource", "events",
			"dropped_attributes_count", "dropped_events_count", "dropped_links_count",
		},
		pgx.CopyFromSlice(len(spans), func(i int) ([]any, error) {
			s := spans[i]
			return []any{
				s.StartTime, s.TraceID, s.SpanID, s.ParentSpanID,
				s.TraceState, s.Flags, s.Name, int16(s.Kind), s.ServiceName,
				s.EndTime, s.DurationNS, int16(s.StatusCode), s.StatusMessage,
				s.Attributes, s.Resource, s.Events,
				s.DroppedAttributesCount, s.DroppedEventsCount, s.DroppedLinksCount,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("spanRepo.InsertBatch: %w", err)
	}
	return nil
}

// GetByTraceID returns every span sharing the given 16-byte trace ID,
// ordered by start_time so callers can render a waterfall directly.
func (r *SpanRepository) GetByTraceID(ctx context.Context, traceID []byte) ([]*domain.Span, error) {
	q := r.db.Querier(ctx)

	rows, err := q.Query(ctx, `
		SELECT start_time, trace_id, span_id, parent_span_id,
		       trace_state, flags, name, kind, service_name,
		       end_time, duration_ns, status_code, status_message,
		       attributes, resource, events,
		       dropped_attributes_count, dropped_events_count, dropped_links_count
		FROM spans
		WHERE trace_id = $1
		ORDER BY start_time ASC, span_id ASC`, traceID)
	if err != nil {
		return nil, fmt.Errorf("spanRepo.GetByTraceID: %w", err)
	}
	defer rows.Close()

	var spans []*domain.Span
	for rows.Next() {
		s := &domain.Span{}
		var kind, statusCode int16
		if err := rows.Scan(
			&s.StartTime, &s.TraceID, &s.SpanID, &s.ParentSpanID,
			&s.TraceState, &s.Flags, &s.Name, &kind, &s.ServiceName,
			&s.EndTime, &s.DurationNS, &statusCode, &s.StatusMessage,
			&s.Attributes, &s.Resource, &s.Events,
			&s.DroppedAttributesCount, &s.DroppedEventsCount, &s.DroppedLinksCount,
		); err != nil {
			return nil, fmt.Errorf("spanRepo.GetByTraceID scan: %w", err)
		}
		s.Kind = domain.SpanKind(kind)
		s.StatusCode = domain.SpanStatusCode(statusCode)
		spans = append(spans, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("spanRepo.GetByTraceID rows: %w", err)
	}
	return spans, nil
}

// DeleteOlderThan removes every span chunk with start_time < cutoff.
// On a hypertable this is a chunk drop, not a row-level delete.
func (r *SpanRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	q := r.db.Querier(ctx)
	_, err := q.Exec(ctx, `DELETE FROM spans WHERE start_time < $1`, cutoff)
	if err != nil {
		return fmt.Errorf("spanRepo.DeleteOlderThan: %w", err)
	}
	return nil
}
