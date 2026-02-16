package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
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

	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, prefix, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := q.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Name,
		token.TokenHash,
		token.Prefix,
		token.ExpiresAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.Create: %w", err)
	}

	return nil
}

// GetByTokenHash retrieves an API token by its SHA-256 hash.
func (r *APITokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, name, token_hash, prefix, last_used_at, expires_at, created_at
		FROM api_tokens
		WHERE token_hash = $1`

	token := &domain.APIToken{}
	err := q.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.TokenHash,
		&token.Prefix,
		&token.LastUsedAt,
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

	query := `
		SELECT id, user_id, name, token_hash, prefix, last_used_at, expires_at, created_at
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, userID)
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
			&token.LastUsedAt,
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

	result, err := q.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("apiTokenRepo.Delete(%s): token not found", id)
	}

	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token.
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)

	_, err := q.Exec(ctx, `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("apiTokenRepo.UpdateLastUsed(%s): %w", id, err)
	}

	return nil
}
