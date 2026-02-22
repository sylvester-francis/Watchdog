package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

const monitorColumns = "id, agent_id, name, type, target, interval_seconds, timeout_seconds, status, enabled, created_at"

// MonitorRepository implements ports.MonitorRepository using PostgreSQL.
type MonitorRepository struct {
	db *DB
}

// NewMonitorRepository creates a new MonitorRepository.
func NewMonitorRepository(db *DB) *MonitorRepository {
	return &MonitorRepository{db: db}
}

func scanMonitor(scanner interface{ Scan(dest ...any) error }) (*domain.Monitor, error) {
	m := &domain.Monitor{}
	err := scanner.Scan(
		&m.ID, &m.AgentID, &m.Name, &m.Type, &m.Target,
		&m.IntervalSeconds, &m.TimeoutSeconds, &m.Status, &m.Enabled, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func scanMonitors(rows pgx.Rows) ([]*domain.Monitor, error) {
	defer rows.Close()
	var monitors []*domain.Monitor
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

// Create inserts a new monitor into the database.
func (r *MonitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	q := r.db.Querier(ctx)

	query := `
		INSERT INTO monitors (id, agent_id, name, type, target, interval_seconds, timeout_seconds, status, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := q.Exec(ctx, query,
		monitor.ID, monitor.AgentID, monitor.Name, monitor.Type, monitor.Target,
		monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Status, monitor.Enabled, monitor.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("monitorRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a monitor by its ID.
func (r *MonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	q := r.db.Querier(ctx)

	query := `SELECT ` + monitorColumns + ` FROM monitors WHERE id = $1`

	monitor, err := scanMonitor(q.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("monitorRepo.GetByID(%s): %w", id, err)
	}

	return monitor, nil
}

// GetByAgentID retrieves all monitors belonging to an agent.
func (r *MonitorRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	q := r.db.Querier(ctx)

	query := `SELECT ` + monitorColumns + ` FROM monitors WHERE agent_id = $1 ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("monitorRepo.GetByAgentID(%s): %w", agentID, err)
	}

	monitors, err := scanMonitors(rows)
	if err != nil {
		return nil, fmt.Errorf("monitorRepo.GetByAgentID(%s): %w", agentID, err)
	}

	return monitors, nil
}

// GetEnabledByAgentID retrieves all enabled monitors for an agent.
func (r *MonitorRepository) GetEnabledByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	q := r.db.Querier(ctx)

	query := `SELECT ` + monitorColumns + ` FROM monitors WHERE agent_id = $1 AND enabled = true ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("monitorRepo.GetEnabledByAgentID(%s): %w", agentID, err)
	}

	monitors, err := scanMonitors(rows)
	if err != nil {
		return nil, fmt.Errorf("monitorRepo.GetEnabledByAgentID(%s): %w", agentID, err)
	}

	return monitors, nil
}

// Update updates an existing monitor in the database.
func (r *MonitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	q := r.db.Querier(ctx)

	query := `
		UPDATE monitors
		SET name = $2, type = $3, target = $4, interval_seconds = $5, timeout_seconds = $6, status = $7, enabled = $8
		WHERE id = $1`

	result, err := q.Exec(ctx, query,
		monitor.ID, monitor.Name, monitor.Type, monitor.Target,
		monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Status, monitor.Enabled,
	)
	if err != nil {
		return fmt.Errorf("monitorRepo.Update(%s): %w", monitor.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitorRepo.Update(%s): monitor not found", monitor.ID)
	}

	return nil
}

// Delete removes a monitor from the database.
func (r *MonitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)

	query := `DELETE FROM monitors WHERE id = $1`

	result, err := q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("monitorRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitorRepo.Delete(%s): monitor not found", id)
	}

	return nil
}

// CountByUserID returns the total number of monitors belonging to a user across all their agents.
func (r *MonitorRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	q := r.db.Querier(ctx)

	query := `SELECT COUNT(*) FROM monitors m JOIN agents a ON m.agent_id = a.id WHERE a.user_id = $1`

	var count int
	err := q.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("monitorRepo.CountByUserID(%s): %w", userID, err)
	}

	return count, nil
}

// UpdateStatus updates only the status of a monitor.
func (r *MonitorRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MonitorStatus) error {
	q := r.db.Querier(ctx)

	query := `UPDATE monitors SET status = $2 WHERE id = $1`

	result, err := q.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("monitorRepo.UpdateStatus(%s, %s): %w", id, status, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitorRepo.UpdateStatus(%s): monitor not found", id)
	}

	return nil
}
