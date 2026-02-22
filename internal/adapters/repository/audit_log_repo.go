package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// AuditLogRepository implements ports.AuditLogRepository using PostgreSQL.
type AuditLogRepository struct {
	db *DB
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create inserts a new audit log entry.
func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	metadata, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("auditLogRepo.Create: marshal metadata: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, user_id, action, metadata, ip_address, created_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5::inet, $6, $7)`

	_, err = q.Exec(ctx, query,
		log.ID, log.UserID, string(log.Action), metadata, nullableIP(log.IPAddress), log.CreatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("auditLogRepo.Create: %w", err)
	}

	return nil
}

// GetByUserID returns audit logs for a specific user.
func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.AuditLog, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, user_id, action, metadata, ip_address::text, created_at
		FROM audit_logs
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := q.Query(ctx, query, userID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("auditLogRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// GetRecent returns the most recent audit logs across all users.
func (r *AuditLogRepository) GetRecent(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, user_id, action, metadata, ip_address::text, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := q.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("auditLogRepo.GetRecent: %w", err)
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

func scanAuditLogs(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	for rows.Next() {
		var (
			log      domain.AuditLog
			metadata []byte
			ip       *string
		)
		if err := rows.Scan(&log.ID, &log.UserID, &log.Action, &metadata, &ip, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("auditLogRepo.scan: %w", err)
		}
		if metadata != nil {
			_ = json.Unmarshal(metadata, &log.Metadata)
		}
		if ip != nil {
			log.IPAddress = *ip
		}
		logs = append(logs, &log)
	}
	return logs, nil
}

// nullableIP returns nil for empty IP strings to avoid postgres INET parse errors.
func nullableIP(ip string) any {
	if ip == "" {
		return nil
	}
	return ip
}
