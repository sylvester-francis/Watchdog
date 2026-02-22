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

// IncidentRepository implements ports.IncidentRepository using PostgreSQL.
type IncidentRepository struct {
	db *DB
}

// NewIncidentRepository creates a new IncidentRepository.
func NewIncidentRepository(db *DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// Create inserts a new incident into the database.
func (r *IncidentRepository) Create(ctx context.Context, incident *domain.Incident) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO incidents (id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := q.Exec(ctx, query,
		incident.ID,
		incident.MonitorID,
		incident.StartedAt,
		incident.ResolvedAt,
		incident.TTRSeconds,
		incident.AcknowledgedBy,
		incident.AcknowledgedAt,
		incident.Status,
		incident.CreatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("incidentRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves an incident by its ID.
func (r *IncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE id = $1 AND tenant_id = $2`

	incident := &domain.Incident{}
	err := q.QueryRow(ctx, query, id, tenantID).Scan(
		&incident.ID,
		&incident.MonitorID,
		&incident.StartedAt,
		&incident.ResolvedAt,
		&incident.TTRSeconds,
		&incident.AcknowledgedBy,
		&incident.AcknowledgedAt,
		&incident.Status,
		&incident.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("incidentRepo.GetByID(%s): %w", id, err)
	}

	return incident, nil
}

// GetByMonitorID retrieves all incidents for a monitor.
func (r *IncidentRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, monitorID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("incidentRepo.GetByMonitorID(%s): %w", monitorID, err)
	}
	defer rows.Close()

	return scanIncidents(rows)
}

// GetOpenByMonitorID retrieves the currently open incident for a monitor, if any.
// There should only be one open incident per monitor at any time.
func (r *IncidentRepository) GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE monitor_id = $1 AND tenant_id = $2 AND status = 'open'
		LIMIT 1`

	incident := &domain.Incident{}
	err := q.QueryRow(ctx, query, monitorID, tenantID).Scan(
		&incident.ID,
		&incident.MonitorID,
		&incident.StartedAt,
		&incident.ResolvedAt,
		&incident.TTRSeconds,
		&incident.AcknowledgedBy,
		&incident.AcknowledgedAt,
		&incident.Status,
		&incident.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("incidentRepo.GetOpenByMonitorID(%s): %w", monitorID, err)
	}

	return incident, nil
}

// GetActiveIncidents retrieves all active (open or acknowledged) incidents.
func (r *IncidentRepository) GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("incidentRepo.GetActiveIncidents: %w", err)
	}
	defer rows.Close()

	return scanIncidents(rows)
}

// GetResolvedIncidents retrieves all resolved incidents, ordered by most recently resolved.
func (r *IncidentRepository) GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE tenant_id = $1 AND status = 'resolved'
		ORDER BY resolved_at DESC
		LIMIT 100`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("incidentRepo.GetResolvedIncidents: %w", err)
	}
	defer rows.Close()

	return scanIncidents(rows)
}

// GetAllIncidents retrieves all incidents, ordered by most recent first.
func (r *IncidentRepository) GetAllIncidents(ctx context.Context) ([]*domain.Incident, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, monitor_id, started_at, resolved_at, ttr_seconds, acknowledged_by, acknowledged_at, status, created_at
		FROM incidents
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 200`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("incidentRepo.GetAllIncidents: %w", err)
	}
	defer rows.Close()

	return scanIncidents(rows)
}

// Update updates an existing incident in the database.
func (r *IncidentRepository) Update(ctx context.Context, incident *domain.Incident) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		UPDATE incidents
		SET resolved_at = $2, ttr_seconds = $3, acknowledged_by = $4, acknowledged_at = $5, status = $6
		WHERE id = $1 AND tenant_id = $7`

	result, err := q.Exec(ctx, query,
		incident.ID,
		incident.ResolvedAt,
		incident.TTRSeconds,
		incident.AcknowledgedBy,
		incident.AcknowledgedAt,
		incident.Status,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("incidentRepo.Update(%s): %w", incident.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("incidentRepo.Update(%s): incident not found", incident.ID)
	}

	return nil
}

// Acknowledge marks an incident as acknowledged by a user.
func (r *IncidentRepository) Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	now := time.Now()
	query := `
		UPDATE incidents
		SET status = 'acknowledged', acknowledged_by = $2, acknowledged_at = $3
		WHERE id = $1 AND tenant_id = $4 AND status = 'open'`

	result, err := q.Exec(ctx, query, id, userID, now, tenantID)
	if err != nil {
		return fmt.Errorf("incidentRepo.Acknowledge(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("incidentRepo.Acknowledge(%s): incident not found or not open", id)
	}

	return nil
}

// Resolve marks an incident as resolved and calculates TTR.
func (r *IncidentRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	// Use database time for consistency and calculate TTR in SQL
	query := `
		UPDATE incidents
		SET status = 'resolved',
		    resolved_at = NOW(),
		    ttr_seconds = EXTRACT(EPOCH FROM (NOW() - started_at))::INT
		WHERE id = $1 AND tenant_id = $2 AND status IN ('open', 'acknowledged')`

	result, err := q.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("incidentRepo.Resolve(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("incidentRepo.Resolve(%s): incident not found or already resolved", id)
	}

	return nil
}

// scanIncidents is a helper function to scan rows into incidents slice.
func scanIncidents(rows pgx.Rows) ([]*domain.Incident, error) {
	var incidents []*domain.Incident
	for rows.Next() {
		incident := &domain.Incident{}
		err := rows.Scan(
			&incident.ID,
			&incident.MonitorID,
			&incident.StartedAt,
			&incident.ResolvedAt,
			&incident.TTRSeconds,
			&incident.AcknowledgedBy,
			&incident.AcknowledgedAt,
			&incident.Status,
			&incident.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		incidents = append(incidents, incident)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return incidents, nil
}
