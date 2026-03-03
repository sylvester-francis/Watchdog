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

// MaintenanceWindowRepository implements ports.MaintenanceWindowRepository using PostgreSQL.
type MaintenanceWindowRepository struct {
	db *DB
}

// NewMaintenanceWindowRepository creates a new MaintenanceWindowRepository.
func NewMaintenanceWindowRepository(db *DB) *MaintenanceWindowRepository {
	return &MaintenanceWindowRepository{db: db}
}

// Create inserts a new maintenance window.
func (r *MaintenanceWindowRepository) Create(ctx context.Context, window *domain.MaintenanceWindow) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO maintenance_windows (id, agent_id, user_id, name, starts_at, ends_at, recurrence, created_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := q.Exec(ctx, query,
		window.ID,
		window.AgentID,
		window.UserID,
		window.Name,
		window.StartsAt,
		window.EndsAt,
		window.Recurrence,
		window.CreatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("maintenanceWindowRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a maintenance window by ID.
func (r *MaintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MaintenanceWindow, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, agent_id, user_id, name, starts_at, ends_at, recurrence, created_at, tenant_id
		FROM maintenance_windows
		WHERE id = $1 AND tenant_id = $2`

	var mw domain.MaintenanceWindow
	err := q.QueryRow(ctx, query, id, tenantID).Scan(
		&mw.ID, &mw.AgentID, &mw.UserID, &mw.Name,
		&mw.StartsAt, &mw.EndsAt, &mw.Recurrence, &mw.CreatedAt, &mw.TenantID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("maintenanceWindowRepo.GetByID: %w", err)
	}

	return &mw, nil
}

// GetByTenant retrieves all maintenance windows for the current tenant.
func (r *MaintenanceWindowRepository) GetByTenant(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, agent_id, user_id, name, starts_at, ends_at, recurrence, created_at, tenant_id
		FROM maintenance_windows
		WHERE tenant_id = $1
		ORDER BY starts_at DESC
		LIMIT 100`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("maintenanceWindowRepo.GetByTenant: %w", err)
	}
	defer rows.Close()

	var windows []*domain.MaintenanceWindow
	for rows.Next() {
		var mw domain.MaintenanceWindow
		if err := rows.Scan(
			&mw.ID, &mw.AgentID, &mw.UserID, &mw.Name,
			&mw.StartsAt, &mw.EndsAt, &mw.Recurrence, &mw.CreatedAt, &mw.TenantID,
		); err != nil {
			return nil, fmt.Errorf("maintenanceWindowRepo.GetByTenant: scan: %w", err)
		}
		windows = append(windows, &mw)
	}

	return windows, rows.Err()
}

// GetActiveByAgentID returns the first active maintenance window for an agent.
func (r *MaintenanceWindowRepository) GetActiveByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.MaintenanceWindow, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, agent_id, user_id, name, starts_at, ends_at, recurrence, created_at, tenant_id
		FROM maintenance_windows
		WHERE agent_id = $1 AND tenant_id = $2 AND starts_at <= NOW() AND ends_at > NOW()
		LIMIT 1`

	var mw domain.MaintenanceWindow
	err := q.QueryRow(ctx, query, agentID, tenantID).Scan(
		&mw.ID, &mw.AgentID, &mw.UserID, &mw.Name,
		&mw.StartsAt, &mw.EndsAt, &mw.Recurrence, &mw.CreatedAt, &mw.TenantID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("maintenanceWindowRepo.GetActiveByAgentID: %w", err)
	}

	return &mw, nil
}

// Update updates a maintenance window.
func (r *MaintenanceWindowRepository) Update(ctx context.Context, window *domain.MaintenanceWindow) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		UPDATE maintenance_windows
		SET name = $1, starts_at = $2, ends_at = $3, recurrence = $4
		WHERE id = $5 AND tenant_id = $6`

	result, err := q.Exec(ctx, query, window.Name, window.StartsAt, window.EndsAt, window.Recurrence, window.ID, tenantID)
	if err != nil {
		return fmt.Errorf("maintenanceWindowRepo.Update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("maintenanceWindowRepo.Update: window not found")
	}

	return nil
}

// Delete removes a maintenance window.
func (r *MaintenanceWindowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	result, err := q.Exec(ctx, `DELETE FROM maintenance_windows WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("maintenanceWindowRepo.Delete(%s): %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("maintenanceWindowRepo.Delete(%s): window not found", id)
	}

	return nil
}

// GetExpiredWithOfflineAgents returns maintenance windows that expired in the last 10 minutes
// where the agent is still offline. Used by the background job to trigger delayed alerts.
func (r *MaintenanceWindowRepository) GetExpiredWithOfflineAgents(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT mw.id, mw.agent_id, mw.user_id, mw.name, mw.starts_at, mw.ends_at, mw.recurrence, mw.created_at, mw.tenant_id
		FROM maintenance_windows mw
		JOIN agents a ON a.id = mw.agent_id
		WHERE mw.ends_at <= NOW()
		  AND mw.ends_at > NOW() - INTERVAL '10 minutes'
		  AND a.status = 'offline'`

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("maintenanceWindowRepo.GetExpiredWithOfflineAgents: %w", err)
	}
	defer rows.Close()

	var windows []*domain.MaintenanceWindow
	for rows.Next() {
		var mw domain.MaintenanceWindow
		if err := rows.Scan(
			&mw.ID, &mw.AgentID, &mw.UserID, &mw.Name,
			&mw.StartsAt, &mw.EndsAt, &mw.Recurrence, &mw.CreatedAt, &mw.TenantID,
		); err != nil {
			return nil, fmt.Errorf("maintenanceWindowRepo.GetExpiredWithOfflineAgents: scan: %w", err)
		}
		windows = append(windows, &mw)
	}

	return windows, rows.Err()
}

// GetExpiredRecurring returns recurring maintenance windows that expired in the last 10 minutes.
// Used by the background job to create the next occurrence.
func (r *MaintenanceWindowRepository) GetExpiredRecurring(ctx context.Context) ([]*domain.MaintenanceWindow, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, agent_id, user_id, name, starts_at, ends_at, recurrence, created_at, tenant_id
		FROM maintenance_windows
		WHERE recurrence != 'once'
		  AND ends_at <= NOW()
		  AND ends_at > NOW() - INTERVAL '10 minutes'`

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("maintenanceWindowRepo.GetExpiredRecurring: %w", err)
	}
	defer rows.Close()

	var windows []*domain.MaintenanceWindow
	for rows.Next() {
		var mw domain.MaintenanceWindow
		if err := rows.Scan(
			&mw.ID, &mw.AgentID, &mw.UserID, &mw.Name,
			&mw.StartsAt, &mw.EndsAt, &mw.Recurrence, &mw.CreatedAt, &mw.TenantID,
		); err != nil {
			return nil, fmt.Errorf("maintenanceWindowRepo.GetExpiredRecurring: scan: %w", err)
		}
		windows = append(windows, &mw)
	}

	return windows, rows.Err()
}

// DeleteExpired removes maintenance windows that ended before the given time.
func (r *MaintenanceWindowRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	q := r.db.Querier(ctx)

	_, err := q.Exec(ctx, `DELETE FROM maintenance_windows WHERE ends_at < $1`, before)
	if err != nil {
		return fmt.Errorf("maintenanceWindowRepo.DeleteExpired: %w", err)
	}

	return nil
}
