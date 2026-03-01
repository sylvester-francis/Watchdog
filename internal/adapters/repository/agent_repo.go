package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// AgentRepository implements ports.AgentRepository using PostgreSQL.
type AgentRepository struct {
	db *DB
}

// NewAgentRepository creates a new AgentRepository.
func NewAgentRepository(db *DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create inserts a new agent into the database.
func (r *AgentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO agents (id, user_id, name, api_key_encrypted, api_key_expires_at, last_seen_at, status, created_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := q.Exec(ctx, query,
		agent.ID,
		agent.UserID,
		agent.Name,
		agent.APIKeyEncrypted,
		agent.APIKeyExpiresAt,
		agent.LastSeenAt,
		agent.Status,
		agent.CreatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("agentRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves an agent by its ID.
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, user_id, name, api_key_encrypted, api_key_expires_at, last_seen_at, status, fingerprint, fingerprint_verified_at, created_at
		FROM agents
		WHERE id = $1 AND tenant_id = $2`

	agent := &domain.Agent{}
	var fingerprintJSON []byte
	err := q.QueryRow(ctx, query, id, tenantID).Scan(
		&agent.ID,
		&agent.UserID,
		&agent.Name,
		&agent.APIKeyEncrypted,
		&agent.APIKeyExpiresAt,
		&agent.LastSeenAt,
		&agent.Status,
		&fingerprintJSON,
		&agent.FingerprintVerifiedAt,
		&agent.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("agentRepo.GetByID(%s): %w", id, err)
	}
	if fingerprintJSON != nil {
		_ = json.Unmarshal(fingerprintJSON, &agent.Fingerprint)
	}

	return agent, nil
}

// GetByIDGlobal retrieves an agent by its ID without tenant scoping.
// Used for API key validation where the caller has no tenant context (e.g. WebSocket auth).
func (r *AgentRepository) GetByIDGlobal(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, name, api_key_encrypted, api_key_expires_at, last_seen_at, status, fingerprint, fingerprint_verified_at, tenant_id, created_at
		FROM agents
		WHERE id = $1`

	agent := &domain.Agent{}
	var fingerprintJSON []byte
	err := q.QueryRow(ctx, query, id).Scan(
		&agent.ID,
		&agent.UserID,
		&agent.Name,
		&agent.APIKeyEncrypted,
		&agent.APIKeyExpiresAt,
		&agent.LastSeenAt,
		&agent.Status,
		&fingerprintJSON,
		&agent.FingerprintVerifiedAt,
		&agent.TenantID,
		&agent.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("agentRepo.GetByIDGlobal(%s): %w", id, err)
	}
	if fingerprintJSON != nil {
		_ = json.Unmarshal(fingerprintJSON, &agent.Fingerprint)
	}

	return agent, nil
}

// GetByUserID retrieves all agents belonging to a user.
func (r *AgentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	// H-020: hard limit prevents unbounded result sets.
	query := `
		SELECT id, user_id, name, api_key_encrypted, api_key_expires_at, last_seen_at, status, fingerprint, fingerprint_verified_at, created_at
		FROM agents
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT 1000`

	rows, err := q.Query(ctx, query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("agentRepo.GetByUserID(%s): %w", userID, err)
	}
	defer rows.Close()

	var agents []*domain.Agent
	for rows.Next() {
		agent := &domain.Agent{}
		var fingerprintJSON []byte
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.APIKeyEncrypted,
			&agent.APIKeyExpiresAt,
			&agent.LastSeenAt,
			&agent.Status,
			&fingerprintJSON,
			&agent.FingerprintVerifiedAt,
			&agent.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("agentRepo.GetByUserID(%s): scan: %w", userID, err)
		}
		if fingerprintJSON != nil {
			_ = json.Unmarshal(fingerprintJSON, &agent.Fingerprint)
		}
		agents = append(agents, agent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("agentRepo.GetByUserID(%s): rows: %w", userID, err)
	}

	return agents, nil
}

// Update updates an existing agent in the database.
func (r *AgentRepository) Update(ctx context.Context, agent *domain.Agent) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		UPDATE agents
		SET name = $2, api_key_encrypted = $3, api_key_expires_at = $4, last_seen_at = $5, status = $6
		WHERE id = $1 AND tenant_id = $7`

	result, err := q.Exec(ctx, query,
		agent.ID,
		agent.Name,
		agent.APIKeyEncrypted,
		agent.APIKeyExpiresAt,
		agent.LastSeenAt,
		agent.Status,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("agentRepo.Update(%s): %w", agent.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.Update(%s): agent not found", agent.ID)
	}

	return nil
}

// Delete removes an agent from the database.
func (r *AgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `DELETE FROM agents WHERE id = $1 AND tenant_id = $2`

	result, err := q.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("agentRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.Delete(%s): agent not found", id)
	}

	return nil
}

// UpdateStatus updates only the status of an agent.
func (r *AgentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `UPDATE agents SET status = $2 WHERE id = $1 AND tenant_id = $3`

	result, err := q.Exec(ctx, query, id, status, tenantID)
	if err != nil {
		return fmt.Errorf("agentRepo.UpdateStatus(%s, %s): %w", id, status, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.UpdateStatus(%s): agent not found", id)
	}

	return nil
}

// CountByUserID returns the number of agents belonging to a user.
func (r *AgentRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT COUNT(*) FROM agents WHERE user_id = $1 AND tenant_id = $2`

	var count int
	err := q.QueryRow(ctx, query, userID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("agentRepo.CountByUserID(%s): %w", userID, err)
	}

	return count, nil
}

// UpdateFingerprint updates the fingerprint and sets the verified-at timestamp.
func (r *AgentRepository) UpdateFingerprint(ctx context.Context, id uuid.UUID, fingerprint map[string]string) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	fpJSON, err := json.Marshal(fingerprint)
	if err != nil {
		return fmt.Errorf("agentRepo.UpdateFingerprint(%s): marshal: %w", id, err)
	}

	query := `UPDATE agents SET fingerprint = $2, fingerprint_verified_at = NOW() WHERE id = $1 AND tenant_id = $3`

	result, err := q.Exec(ctx, query, id, fpJSON, tenantID)
	if err != nil {
		return fmt.Errorf("agentRepo.UpdateFingerprint(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.UpdateFingerprint(%s): agent not found", id)
	}

	return nil
}

// UpdateLastSeen updates only the last_seen_at timestamp of an agent.
func (r *AgentRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `UPDATE agents SET last_seen_at = $2 WHERE id = $1 AND tenant_id = $3`

	result, err := q.Exec(ctx, query, id, lastSeen, tenantID)
	if err != nil {
		return fmt.Errorf("agentRepo.UpdateLastSeen(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.UpdateLastSeen(%s): agent not found", id)
	}

	return nil
}
