package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// HeartbeatRepository implements ports.HeartbeatRepository using PostgreSQL/TimescaleDB.
type HeartbeatRepository struct {
	db *DB
}

// NewHeartbeatRepository creates a new HeartbeatRepository.
func NewHeartbeatRepository(db *DB) *HeartbeatRepository {
	return &HeartbeatRepository{db: db}
}

// Create inserts a single heartbeat into the database.
func (r *HeartbeatRepository) Create(ctx context.Context, heartbeat *domain.Heartbeat) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO heartbeats (time, monitor_id, agent_id, status, latency_ms, error_message, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := q.Exec(ctx, query,
		heartbeat.Time,
		heartbeat.MonitorID,
		heartbeat.AgentID,
		heartbeat.Status,
		heartbeat.LatencyMs,
		heartbeat.ErrorMessage,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("heartbeatRepo.Create: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple heartbeats efficiently using PostgreSQL's COPY protocol.
func (r *HeartbeatRepository) CreateBatch(ctx context.Context, heartbeats []*domain.Heartbeat) error {
	if len(heartbeats) == 0 {
		return nil
	}

	tenantID := TenantIDFromContext(ctx)

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"heartbeats"},
		[]string{"time", "monitor_id", "agent_id", "status", "latency_ms", "error_message", "tenant_id"},
		pgx.CopyFromSlice(len(heartbeats), func(i int) ([]any, error) {
			h := heartbeats[i]
			return []any{
				h.Time,
				h.MonitorID,
				h.AgentID,
				h.Status,
				h.LatencyMs,
				h.ErrorMessage,
				tenantID,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("heartbeatRepo.CreateBatch: %w", err)
	}

	return nil
}

// GetByMonitorID retrieves the most recent heartbeats for a monitor.
func (r *HeartbeatRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*domain.Heartbeat, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT time, monitor_id, agent_id, status, latency_ms, error_message
		FROM heartbeats
		WHERE monitor_id = $1 AND tenant_id = $2
		ORDER BY time DESC
		LIMIT $3`

	rows, err := q.Query(ctx, query, monitorID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("heartbeatRepo.GetByMonitorID(%s): %w", monitorID, err)
	}
	defer rows.Close()

	return scanHeartbeats(rows, monitorID)
}

// GetByMonitorIDInRange retrieves heartbeats for a monitor within a time range.
func (r *HeartbeatRepository) GetByMonitorIDInRange(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*domain.Heartbeat, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT time, monitor_id, agent_id, status, latency_ms, error_message
		FROM heartbeats
		WHERE monitor_id = $1 AND tenant_id = $2 AND time >= $3 AND time <= $4
		ORDER BY time DESC`

	rows, err := q.Query(ctx, query, monitorID, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("heartbeatRepo.GetByMonitorIDInRange(%s): %w", monitorID, err)
	}
	defer rows.Close()

	return scanHeartbeats(rows, monitorID)
}

// GetLatestByMonitorID retrieves the most recent heartbeat for a monitor.
func (r *HeartbeatRepository) GetLatestByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Heartbeat, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT time, monitor_id, agent_id, status, latency_ms, error_message
		FROM heartbeats
		WHERE monitor_id = $1 AND tenant_id = $2
		ORDER BY time DESC
		LIMIT 1`

	heartbeat := &domain.Heartbeat{}
	err := q.QueryRow(ctx, query, monitorID, tenantID).Scan(
		&heartbeat.Time,
		&heartbeat.MonitorID,
		&heartbeat.AgentID,
		&heartbeat.Status,
		&heartbeat.LatencyMs,
		&heartbeat.ErrorMessage,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("heartbeatRepo.GetLatestByMonitorID(%s): %w", monitorID, err)
	}

	return heartbeat, nil
}

// GetRecentFailures retrieves the most recent failure heartbeats for a monitor.
// This is used for the 3-strike rule to determine if an incident should be created.
func (r *HeartbeatRepository) GetRecentFailures(ctx context.Context, monitorID uuid.UUID, count int) ([]*domain.Heartbeat, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	// Get the most recent `count` heartbeats that are not 'up' status
	query := `
		SELECT time, monitor_id, agent_id, status, latency_ms, error_message
		FROM heartbeats
		WHERE monitor_id = $1 AND tenant_id = $2 AND status != 'up'
		ORDER BY time DESC
		LIMIT $3`

	rows, err := q.Query(ctx, query, monitorID, tenantID, count)
	if err != nil {
		return nil, fmt.Errorf("heartbeatRepo.GetRecentFailures(%s): %w", monitorID, err)
	}
	defer rows.Close()

	return scanHeartbeats(rows, monitorID)
}

// DeleteOlderThan removes heartbeats older than the specified time.
// Returns the number of rows deleted.
func (r *HeartbeatRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `DELETE FROM heartbeats WHERE time < $1 AND tenant_id = $2`

	result, err := q.Exec(ctx, query, before, tenantID)
	if err != nil {
		return 0, fmt.Errorf("heartbeatRepo.DeleteOlderThan: %w", err)
	}

	return result.RowsAffected(), nil
}

// scanHeartbeats is a helper function to scan rows into heartbeats slice.
func scanHeartbeats(rows pgx.Rows, contextID uuid.UUID) ([]*domain.Heartbeat, error) {
	var heartbeats []*domain.Heartbeat
	for rows.Next() {
		heartbeat := &domain.Heartbeat{}
		err := rows.Scan(
			&heartbeat.Time,
			&heartbeat.MonitorID,
			&heartbeat.AgentID,
			&heartbeat.Status,
			&heartbeat.LatencyMs,
			&heartbeat.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		heartbeats = append(heartbeats, heartbeat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return heartbeats, nil
}
