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

	query := `
		INSERT INTO agents (id, user_id, name, api_key_encrypted, last_seen_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := q.Exec(ctx, query,
		agent.ID,
		agent.UserID,
		agent.Name,
		agent.APIKeyEncrypted,
		agent.LastSeenAt,
		agent.Status,
		agent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("agentRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves an agent by its ID.
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, name, api_key_encrypted, last_seen_at, status, created_at
		FROM agents
		WHERE id = $1`

	agent := &domain.Agent{}
	err := q.QueryRow(ctx, query, id).Scan(
		&agent.ID,
		&agent.UserID,
		&agent.Name,
		&agent.APIKeyEncrypted,
		&agent.LastSeenAt,
		&agent.Status,
		&agent.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("agentRepo.GetByID(%s): %w", id, err)
	}

	return agent, nil
}

// GetByUserID retrieves all agents belonging to a user.
func (r *AgentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Agent, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, name, api_key_encrypted, last_seen_at, status, created_at
		FROM agents
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("agentRepo.GetByUserID(%s): %w", userID, err)
	}
	defer rows.Close()

	var agents []*domain.Agent
	for rows.Next() {
		agent := &domain.Agent{}
		err := rows.Scan(
			&agent.ID,
			&agent.UserID,
			&agent.Name,
			&agent.APIKeyEncrypted,
			&agent.LastSeenAt,
			&agent.Status,
			&agent.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("agentRepo.GetByUserID(%s): scan: %w", userID, err)
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

	query := `
		UPDATE agents
		SET name = $2, api_key_encrypted = $3, last_seen_at = $4, status = $5
		WHERE id = $1`

	result, err := q.Exec(ctx, query,
		agent.ID,
		agent.Name,
		agent.APIKeyEncrypted,
		agent.LastSeenAt,
		agent.Status,
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

	query := `DELETE FROM agents WHERE id = $1`

	result, err := q.Exec(ctx, query, id)
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

	query := `UPDATE agents SET status = $2 WHERE id = $1`

	result, err := q.Exec(ctx, query, id, status)
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

	query := `SELECT COUNT(*) FROM agents WHERE user_id = $1`

	var count int
	err := q.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("agentRepo.CountByUserID(%s): %w", userID, err)
	}

	return count, nil
}

// UpdateLastSeen updates only the last_seen_at timestamp of an agent.
func (r *AgentRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error {
	q := r.db.Querier(ctx)

	query := `UPDATE agents SET last_seen_at = $2 WHERE id = $1`

	result, err := q.Exec(ctx, query, id, lastSeen)
	if err != nil {
		return fmt.Errorf("agentRepo.UpdateLastSeen(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("agentRepo.UpdateLastSeen(%s): agent not found", id)
	}

	return nil
}
