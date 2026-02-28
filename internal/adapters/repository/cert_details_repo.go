package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// CertDetailsRepository implements ports.CertDetailsRepository using PostgreSQL.
type CertDetailsRepository struct {
	db *DB
}

// NewCertDetailsRepository creates a new CertDetailsRepository.
func NewCertDetailsRepository(db *DB) *CertDetailsRepository {
	return &CertDetailsRepository{db: db}
}

// Upsert inserts or updates cert details for a monitor+tenant pair.
func (r *CertDetailsRepository) Upsert(ctx context.Context, d *domain.CertDetails) error {
	tenantID := TenantIDFromContext(ctx)
	query := `
		INSERT INTO cert_details (monitor_id, tenant_id, last_checked_at, expiry_days, issuer, sans, algorithm, key_size, serial_number, chain_valid)
		VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (monitor_id, tenant_id) DO UPDATE SET
			last_checked_at = NOW(),
			expiry_days = EXCLUDED.expiry_days,
			issuer = EXCLUDED.issuer,
			sans = EXCLUDED.sans,
			algorithm = EXCLUDED.algorithm,
			key_size = EXCLUDED.key_size,
			serial_number = EXCLUDED.serial_number,
			chain_valid = EXCLUDED.chain_valid`

	_, err := r.db.Pool.Exec(ctx, query,
		d.MonitorID, tenantID, d.ExpiryDays, d.Issuer, d.SANs,
		d.Algorithm, d.KeySize, d.SerialNumber, d.ChainValid,
	)
	if err != nil {
		return fmt.Errorf("cert_details upsert: %w", err)
	}
	return nil
}

// GetByMonitorID returns cert details for a specific monitor, scoped by tenant.
func (r *CertDetailsRepository) GetByMonitorID(ctx context.Context, monitorID uuid.UUID) (*domain.CertDetails, error) {
	tenantID := TenantIDFromContext(ctx)
	query := `
		SELECT monitor_id, tenant_id, last_checked_at, expiry_days, issuer, sans, algorithm, key_size, serial_number, chain_valid
		FROM cert_details
		WHERE monitor_id = $1 AND tenant_id = $2`

	d := &domain.CertDetails{}
	err := r.db.Pool.QueryRow(ctx, query, monitorID, tenantID).Scan(
		&d.MonitorID, &d.TenantID, &d.LastCheckedAt, &d.ExpiryDays, &d.Issuer,
		&d.SANs, &d.Algorithm, &d.KeySize, &d.SerialNumber, &d.ChainValid,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cert_details get by monitor: %w", err)
	}
	return d, nil
}

// GetExpiring returns cert details where expiry_days is within the specified threshold, scoped by tenant.
func (r *CertDetailsRepository) GetExpiring(ctx context.Context, withinDays int) ([]*domain.CertDetails, error) {
	tenantID := TenantIDFromContext(ctx)

	// Cap to prevent abuse
	if withinDays > 365 {
		withinDays = 365
	}

	query := `
		SELECT monitor_id, tenant_id, last_checked_at, expiry_days, issuer, sans, algorithm, key_size, serial_number, chain_valid
		FROM cert_details
		WHERE tenant_id = $1 AND expiry_days IS NOT NULL AND expiry_days <= $2
		ORDER BY expiry_days ASC
		LIMIT 1000`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, withinDays)
	if err != nil {
		return nil, fmt.Errorf("cert_details get expiring: %w", err)
	}
	defer rows.Close()

	var results []*domain.CertDetails
	for rows.Next() {
		d := &domain.CertDetails{}
		if err := rows.Scan(
			&d.MonitorID, &d.TenantID, &d.LastCheckedAt, &d.ExpiryDays, &d.Issuer,
			&d.SANs, &d.Algorithm, &d.KeySize, &d.SerialNumber, &d.ChainValid,
		); err != nil {
			return nil, fmt.Errorf("cert_details scan: %w", err)
		}
		results = append(results, d)
	}

	return results, rows.Err()
}
