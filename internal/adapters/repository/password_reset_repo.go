package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// PasswordResetTokenRepository implements ports.PasswordResetTokenRepository using PostgreSQL.
type PasswordResetTokenRepository struct {
	db *DB
}

// NewPasswordResetTokenRepository creates a new PasswordResetTokenRepository.
func NewPasswordResetTokenRepository(db *DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

// Create inserts a new password reset token.
func (r *PasswordResetTokenRepository) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	q := r.db.Querier(ctx)
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)`
	if _, err := q.Exec(ctx, query, t.ID, t.UserID, t.TokenHash, t.ExpiresAt, t.IPAddress, t.CreatedAt); err != nil {
		return fmt.Errorf("insert password_reset_token: %w", err)
	}
	return nil
}

// GetByHash returns the token matching the given SHA-256 hash, or nil if not found.
func (r *PasswordResetTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error) {
	q := r.db.Querier(ctx)
	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, COALESCE(ip_address, ''), created_at
		FROM password_reset_tokens
		WHERE token_hash = $1`
	var t domain.PasswordResetToken
	err := q.QueryRow(ctx, query, hash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.IPAddress, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query password_reset_token: %w", err)
	}
	return &t, nil
}

// MarkUsed sets used_at = NOW() for the given token id (no-op if already used).
func (r *PasswordResetTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	if _, err := q.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, id); err != nil {
		return fmt.Errorf("mark used: %w", err)
	}
	return nil
}

// DeleteExpired removes tokens that have expired OR were used more than 7 days ago.
// Returns the number of rows deleted.
func (r *PasswordResetTokenRepository) DeleteExpired(ctx context.Context) (int, error) {
	q := r.db.Querier(ctx)
	tag, err := q.Exec(ctx, `DELETE FROM password_reset_tokens WHERE expires_at < NOW() OR used_at < NOW() - INTERVAL '7 days'`)
	if err != nil {
		return 0, fmt.Errorf("delete expired: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
