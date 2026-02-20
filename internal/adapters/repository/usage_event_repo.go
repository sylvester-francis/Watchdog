package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// UsageEventRepository implements ports.UsageEventRepository using PostgreSQL.
type UsageEventRepository struct {
	db *DB
}

// NewUsageEventRepository creates a new UsageEventRepository.
func NewUsageEventRepository(db *DB) *UsageEventRepository {
	return &UsageEventRepository{db: db}
}

func scanUsageEvent(row pgx.Row) (*domain.UsageEvent, error) {
	e := &domain.UsageEvent{}
	err := row.Scan(
		&e.ID, &e.UserID, &e.EventType, &e.ResourceType,
		&e.CurrentCount, &e.MaxAllowed, &e.Plan, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// Create inserts a new usage event into the database.
func (r *UsageEventRepository) Create(ctx context.Context, event *domain.UsageEvent) error {
	q := r.db.Querier(ctx)

	query := `
		INSERT INTO usage_events (id, user_id, event_type, resource_type, current_count, max_allowed, plan, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := q.Exec(ctx, query,
		event.ID, event.UserID, event.EventType, event.ResourceType,
		event.CurrentCount, event.MaxAllowed, event.Plan, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("usageEventRepo.Create: %w", err)
	}

	return nil
}

// GetRecentByUserID retrieves recent usage events for a specific user.
func (r *UsageEventRepository) GetRecentByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UsageEvent, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, event_type, resource_type, current_count, max_allowed, plan, created_at
		FROM usage_events
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := q.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("usageEventRepo.GetRecentByUserID(%s): %w", userID, err)
	}
	defer rows.Close()

	var events []*domain.UsageEvent
	for rows.Next() {
		e, err := scanUsageEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("usageEventRepo.GetRecentByUserID(%s): scan: %w", userID, err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// GetRecent retrieves the most recent usage events across all users.
func (r *UsageEventRepository) GetRecent(ctx context.Context, limit int) ([]*domain.UsageEvent, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, event_type, resource_type, current_count, max_allowed, plan, created_at
		FROM usage_events
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := q.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("usageEventRepo.GetRecent: %w", err)
	}
	defer rows.Close()

	var events []*domain.UsageEvent
	for rows.Next() {
		e, err := scanUsageEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("usageEventRepo.GetRecent: scan: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// CountByEventType returns the number of events of a given type since a timestamp.
func (r *UsageEventRepository) CountByEventType(ctx context.Context, eventType domain.EventType, since time.Time) (int, error) {
	q := r.db.Querier(ctx)

	query := `SELECT COUNT(*) FROM usage_events WHERE event_type = $1 AND created_at >= $2`

	var count int
	err := q.QueryRow(ctx, query, eventType, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("usageEventRepo.CountByEventType(%s): %w", eventType, err)
	}

	return count, nil
}
