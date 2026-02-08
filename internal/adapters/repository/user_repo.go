package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester/watchdog/internal/core/domain"
)

// UserRepository implements ports.UserRepository using PostgreSQL.
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	q := r.db.Querier(ctx)

	query := `
		INSERT INTO users (id, email, password_hash, stripe_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.StripeID,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, email, password_hash, stripe_id, created_at, updated_at
		FROM users
		WHERE id = $1`

	user := &domain.User{}
	err := q.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.StripeID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.GetByID(%s): %w", id, err)
	}

	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, email, password_hash, stripe_id, created_at, updated_at
		FROM users
		WHERE email = $1`

	user := &domain.User{}
	err := q.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.StripeID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.GetByEmail(%s): %w", email, err)
	}

	return user, nil
}

// Update updates an existing user in the database.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	q := r.db.Querier(ctx)

	query := `
		UPDATE users
		SET email = $2, password_hash = $3, stripe_id = $4, updated_at = $5
		WHERE id = $1`

	result, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.StripeID,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Update(%s): %w", user.ID, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.Update(%s): user not found", user.ID)
	}

	return nil
}

// Delete removes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)

	query := `DELETE FROM users WHERE id = $1`

	result, err := q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("userRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.Delete(%s): user not found", id)
	}

	return nil
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	q := r.db.Querier(ctx)

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := q.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepo.ExistsByEmail(%s): %w", email, err)
	}

	return exists, nil
}
