package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// StatusPageSubscriberRepository implements ports.StatusPageSubscriberRepository using PostgreSQL.
type StatusPageSubscriberRepository struct {
	db *DB
}

// NewStatusPageSubscriberRepository constructs the repo.
func NewStatusPageSubscriberRepository(db *DB) *StatusPageSubscriberRepository {
	return &StatusPageSubscriberRepository{db: db}
}

// Upsert inserts a new subscriber row or refreshes the token + sent-at
// timestamp on conflict (page_id, email). Token rotation on re-subscribe is
// intentional: each subscribe attempt yields a fresh confirmation link, and
// the old token stops working.
func (r *StatusPageSubscriberRepository) Upsert(ctx context.Context, s *domain.StatusPageSubscriber) error {
	q := r.db.Querier(ctx)
	query := `
		INSERT INTO status_page_subscribers
			(id, status_page_id, email, token_hash, last_confirmation_sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (status_page_id, email) DO UPDATE
		SET token_hash                = EXCLUDED.token_hash,
		    last_confirmation_sent_at = EXCLUDED.last_confirmation_sent_at`
	if _, err := q.Exec(ctx, query, s.ID, s.StatusPageID, s.Email, s.TokenHash, s.LastConfirmationSentAt, s.CreatedAt); err != nil {
		return fmt.Errorf("upsert subscriber: %w", err)
	}
	return nil
}

// GetByPageAndEmail returns the subscriber row for (page, email) or nil if not found.
func (r *StatusPageSubscriberRepository) GetByPageAndEmail(ctx context.Context, pageID uuid.UUID, email string) (*domain.StatusPageSubscriber, error) {
	q := r.db.Querier(ctx)
	row := q.QueryRow(ctx,
		`SELECT id, status_page_id, email, token_hash, confirmed_at, unsubscribed_at, last_confirmation_sent_at, created_at
		 FROM status_page_subscribers
		 WHERE status_page_id = $1 AND email = $2`,
		pageID, email,
	)
	return scanSubscriberRow(row)
}

// GetByTokenHash returns the subscriber row by hashed token or nil if not found.
func (r *StatusPageSubscriberRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.StatusPageSubscriber, error) {
	q := r.db.Querier(ctx)
	row := q.QueryRow(ctx,
		`SELECT id, status_page_id, email, token_hash, confirmed_at, unsubscribed_at, last_confirmation_sent_at, created_at
		 FROM status_page_subscribers
		 WHERE token_hash = $1`,
		hash,
	)
	return scanSubscriberRow(row)
}

// MarkConfirmed sets confirmed_at=NOW() for a subscriber. Idempotent: re-running
// on an already-confirmed row is a no-op.
func (r *StatusPageSubscriberRepository) MarkConfirmed(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	if _, err := q.Exec(ctx, `UPDATE status_page_subscribers SET confirmed_at = NOW() WHERE id = $1 AND confirmed_at IS NULL`, id); err != nil {
		return fmt.Errorf("mark confirmed: %w", err)
	}
	return nil
}

// MarkUnsubscribed sets unsubscribed_at=NOW() for a subscriber. Idempotent.
func (r *StatusPageSubscriberRepository) MarkUnsubscribed(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	if _, err := q.Exec(ctx, `UPDATE status_page_subscribers SET unsubscribed_at = NOW() WHERE id = $1 AND unsubscribed_at IS NULL`, id); err != nil {
		return fmt.Errorf("mark unsubscribed: %w", err)
	}
	return nil
}

// ListActiveForPage returns all confirmed-and-not-unsubscribed subscribers
// for a page. Used by the incident-opened notification fan-out.
func (r *StatusPageSubscriberRepository) ListActiveForPage(ctx context.Context, pageID uuid.UUID) ([]*domain.StatusPageSubscriber, error) {
	q := r.db.Querier(ctx)
	rows, err := q.Query(ctx,
		`SELECT id, status_page_id, email, token_hash, confirmed_at, unsubscribed_at, last_confirmation_sent_at, created_at
		 FROM status_page_subscribers
		 WHERE status_page_id = $1 AND confirmed_at IS NOT NULL AND unsubscribed_at IS NULL`,
		pageID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active subs: %w", err)
	}
	defer rows.Close()

	var out []*domain.StatusPageSubscriber
	for rows.Next() {
		s := &domain.StatusPageSubscriber{}
		if err := rows.Scan(&s.ID, &s.StatusPageID, &s.Email, &s.TokenHash, &s.ConfirmedAt, &s.UnsubscribedAt, &s.LastConfirmationSentAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan subscriber row: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// scanSubscriberRow handles single-row scans for both GetByPageAndEmail and
// GetByTokenHash. Returns (nil, nil) when no row found.
func scanSubscriberRow(row pgx.Row) (*domain.StatusPageSubscriber, error) {
	s := &domain.StatusPageSubscriber{}
	err := row.Scan(&s.ID, &s.StatusPageID, &s.Email, &s.TokenHash, &s.ConfirmedAt, &s.UnsubscribedAt, &s.LastConfirmationSentAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan subscriber: %w", err)
	}
	return s, nil
}
