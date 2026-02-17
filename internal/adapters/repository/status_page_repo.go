package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

// StatusPageRepository implements ports.StatusPageRepository using PostgreSQL.
type StatusPageRepository struct {
	db *DB
}

// NewStatusPageRepository creates a new StatusPageRepository.
func NewStatusPageRepository(db *DB) *StatusPageRepository {
	return &StatusPageRepository{db: db}
}

// Create inserts a new status page.
func (r *StatusPageRepository) Create(ctx context.Context, page *domain.StatusPage) error {
	query := `INSERT INTO status_pages (id, user_id, name, slug, description, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Pool.Exec(ctx, query,
		page.ID, page.UserID, page.Name, page.Slug, page.Description, page.IsPublic, page.CreatedAt, page.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create status page: %w", err)
	}
	return nil
}

// GetByID returns a status page by ID.
func (r *StatusPageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StatusPage, error) {
	query := `SELECT id, user_id, name, slug, description, is_public, created_at, updated_at
		FROM status_pages WHERE id = $1`

	page := &domain.StatusPage{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&page.ID, &page.UserID, &page.Name, &page.Slug, &page.Description, &page.IsPublic, &page.CreatedAt, &page.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get status page by id: %w", err)
	}
	return page, nil
}

// GetByUserAndSlug returns a status page by username and slug.
func (r *StatusPageRepository) GetByUserAndSlug(ctx context.Context, username, slug string) (*domain.StatusPage, error) {
	query := `SELECT sp.id, sp.user_id, sp.name, sp.slug, sp.description, sp.is_public, sp.created_at, sp.updated_at
		FROM status_pages sp
		JOIN users u ON sp.user_id = u.id
		WHERE u.username = $1 AND sp.slug = $2`

	page := &domain.StatusPage{}
	err := r.db.Pool.QueryRow(ctx, query, username, slug).Scan(
		&page.ID, &page.UserID, &page.Name, &page.Slug, &page.Description, &page.IsPublic, &page.CreatedAt, &page.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get status page by user and slug: %w", err)
	}
	return page, nil
}

// GetByUserID returns all status pages for a user.
func (r *StatusPageRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.StatusPage, error) {
	query := `SELECT id, user_id, name, slug, description, is_public, created_at, updated_at
		FROM status_pages WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get status pages by user: %w", err)
	}
	defer rows.Close()

	var pages []*domain.StatusPage
	for rows.Next() {
		page := &domain.StatusPage{}
		if err := rows.Scan(
			&page.ID, &page.UserID, &page.Name, &page.Slug, &page.Description, &page.IsPublic, &page.CreatedAt, &page.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan status page: %w", err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// Update updates a status page.
func (r *StatusPageRepository) Update(ctx context.Context, page *domain.StatusPage) error {
	query := `UPDATE status_pages SET name = $1, slug = $2, description = $3, is_public = $4, updated_at = NOW()
		WHERE id = $5`

	_, err := r.db.Pool.Exec(ctx, query, page.Name, page.Slug, page.Description, page.IsPublic, page.ID)
	if err != nil {
		return fmt.Errorf("update status page: %w", err)
	}
	return nil
}

// Delete removes a status page.
func (r *StatusPageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM status_pages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete status page: %w", err)
	}
	return nil
}

// SetMonitors replaces all monitors for a status page.
func (r *StatusPageRepository) SetMonitors(ctx context.Context, pageID uuid.UUID, monitorIDs []uuid.UUID) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM status_page_monitors WHERE status_page_id = $1`, pageID)
	if err != nil {
		return fmt.Errorf("clear monitors: %w", err)
	}

	for i, monitorID := range monitorIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO status_page_monitors (status_page_id, monitor_id, sort_order) VALUES ($1, $2, $3)`,
			pageID, monitorID, i,
		)
		if err != nil {
			return fmt.Errorf("insert monitor %s: %w", monitorID, err)
		}
	}

	return tx.Commit(ctx)
}

// GetMonitorIDs returns the monitor IDs for a status page.
func (r *StatusPageRepository) GetMonitorIDs(ctx context.Context, pageID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT monitor_id FROM status_page_monitors WHERE status_page_id = $1 ORDER BY sort_order`,
		pageID,
	)
	if err != nil {
		return nil, fmt.Errorf("get monitor ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan monitor id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SlugExistsForUser checks if a slug is already taken by a specific user.
func (r *StatusPageRepository) SlugExistsForUser(ctx context.Context, userID uuid.UUID, slug string) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM status_pages WHERE user_id = $1 AND slug = $2)`, userID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slug exists for user: %w", err)
	}
	return exists, nil
}
