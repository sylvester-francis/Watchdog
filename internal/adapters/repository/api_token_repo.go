package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// APITokenRepository implements ports.APITokenRepository using PostgreSQL.
type APITokenRepository struct {
	db *DB
}

// NewAPITokenRepository creates a new APITokenRepository.
func NewAPITokenRepository(db *DB) *APITokenRepository {
	return &APITokenRepository{db: db}
}

// Create inserts a new API token into the database.
func (r *APITokenRepository) Create(ctx context.Context, token *domain.APIToken) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, scope, expires_at, created_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := q.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Name,
		token.TokenHash,
		token.Prefix,
		token.Scope,
		token.ExpiresAt,
		token.CreatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.Create: %w", err)
	}

	return nil
}

// GetByTokenHash retrieves an API token by its SHA-256 hash.
// This query is intentionally unscoped by tenant — the token_hash column has a
// global UNIQUE constraint, and the lookup must succeed before tenant context
// is established (e.g. during API token authentication).
func (r *APITokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, name, token_hash, prefix, scope, last_used_at, last_used_ip, expires_at, created_at
		FROM api_tokens
		WHERE token_hash = $1`

	token := &domain.APIToken{}
	err := q.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.TokenHash,
		&token.Prefix,
		&token.Scope,
		&token.LastUsedAt,
		&token.LastUsedIP,
		&token.ExpiresAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("apiTokenRepo.GetByTokenHash: %w", err)
	}

	return token, nil
}

// GetByUserID retrieves all API tokens for a user.
func (r *APITokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	// H-020: hard limit prevents unbounded result sets.
	query := `
		SELECT id, user_id, name, token_hash, prefix, scope, last_used_at, last_used_ip, expires_at, created_at
		FROM api_tokens
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT 100`

	rows, err := q.Query(ctx, query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("apiTokenRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	var tokens []*domain.APIToken
	for rows.Next() {
		token := &domain.APIToken{}
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Name,
			&token.TokenHash,
			&token.Prefix,
			&token.Scope,
			&token.LastUsedAt,
			&token.LastUsedIP,
			&token.ExpiresAt,
			&token.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("apiTokenRepo.GetByUserID: scan: %w", err)
		}
		tokens = append(tokens, token)
	}

	return tokens, rows.Err()
}

// Delete removes an API token from the database.
func (r *APITokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	result, err := q.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("apiTokenRepo.Delete(%s): token not found", id)
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp and IP for a token.
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID, ip string) error {
	q := r.db.Querier(ctx)

	// Intentionally unscoped by tenant — token ID is globally unique (UUID)
	// and this runs in the auth middleware before tenant context is available.
	_, err := q.Exec(ctx, `UPDATE api_tokens SET last_used_at = NOW(), last_used_ip = $2 WHERE id = $1`, id, ip)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.UpdateLastUsed(%s): %w", id, err)
	}

	return nil
}
