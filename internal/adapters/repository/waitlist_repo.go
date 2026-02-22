package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// WaitlistRepository implements ports.WaitlistRepository using PostgreSQL.
type WaitlistRepository struct {
	db *DB
}

// NewWaitlistRepository creates a new WaitlistRepository.
func NewWaitlistRepository(db *DB) *WaitlistRepository {
	return &WaitlistRepository{db: db}
}

// Create inserts a new waitlist signup. Duplicate emails are silently ignored.
func (r *WaitlistRepository) Create(ctx context.Context, signup *domain.WaitlistSignup) error {
	q := r.db.Querier(ctx)

	query := `
		INSERT INTO waitlist_signups (id, email, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING`

	_, err := q.Exec(ctx, query, signup.ID, signup.Email, signup.CreatedAt)
	if err != nil {
		return fmt.Errorf("waitlistRepo.Create: %w", err)
	}

	return nil
}

// GetByEmail retrieves a waitlist signup by email.
func (r *WaitlistRepository) GetByEmail(ctx context.Context, email string) (*domain.WaitlistSignup, error) {
	q := r.db.Querier(ctx)

	query := `SELECT id, email, created_at FROM waitlist_signups WHERE email = $1`

	s := &domain.WaitlistSignup{}
	err := q.QueryRow(ctx, query, email).Scan(&s.ID, &s.Email, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("waitlistRepo.GetByEmail(%s): %w", email, err)
	}

	return s, nil
}

// Count returns the total number of waitlist signups.
func (r *WaitlistRepository) Count(ctx context.Context) (int, error) {
	q := r.db.Querier(ctx)

	var count int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM waitlist_signups`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("waitlistRepo.Count: %w", err)
	}

	return count, nil
}
