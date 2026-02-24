package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
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
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `INSERT INTO status_pages (id, user_id, name, slug, description, is_public, created_at, updated_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := q.Exec(ctx, query,
		page.ID, page.UserID, page.Name, page.Slug, page.Description, page.IsPublic, page.CreatedAt, page.UpdatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("create status page: %w", err)
	}
	return nil
}

// GetByID returns a status page by ID.
func (r *StatusPageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.StatusPage, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT id, user_id, name, slug, description, is_public, created_at, updated_at
		FROM status_pages WHERE id = $1 AND tenant_id = $2`

	page := &domain.StatusPage{}
	err := q.QueryRow(ctx, query, id, tenantID).Scan(
		&page.ID, &page.UserID, &page.Name, &page.Slug, &page.Description, &page.IsPublic, &page.CreatedAt, &page.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get status page by id: %w", err)
	}
	return page, nil
}

// GetByUserAndSlug returns a status page by username and slug.
func (r *StatusPageRepository) GetByUserAndSlug(ctx context.Context, username, slug string) (*domain.StatusPage, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT sp.id, sp.user_id, sp.name, sp.slug, sp.description, sp.is_public, sp.created_at, sp.updated_at
		FROM status_pages sp
		JOIN users u ON sp.user_id = u.id
		WHERE u.username = $1 AND sp.slug = $2 AND sp.tenant_id = $3`

	page := &domain.StatusPage{}
	err := q.QueryRow(ctx, query, username, slug, tenantID).Scan(
		&page.ID, &page.UserID, &page.Name, &page.Slug, &page.Description, &page.IsPublic, &page.CreatedAt, &page.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get status page by user and slug: %w", err)
	}
	return page, nil
}

// GetByUserID returns all status pages for a user.
func (r *StatusPageRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.StatusPage, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT id, user_id, name, slug, description, is_public, created_at, updated_at
		FROM status_pages WHERE user_id = $1 AND tenant_id = $2 ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, userID, tenantID)
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
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `UPDATE status_pages SET name = $1, slug = $2, description = $3, is_public = $4, updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6`

	_, err := q.Exec(ctx, query, page.Name, page.Slug, page.Description, page.IsPublic, page.ID, tenantID)
	if err != nil {
		return fmt.Errorf("update status page: %w", err)
	}
	return nil
}

// Delete removes a status page.
func (r *StatusPageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	_, err := q.Exec(ctx, `DELETE FROM status_pages WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete status page: %w", err)
	}
	return nil
}

// SetMonitors replaces all monitors for a status page.
// Defense-in-depth: scopes DELETE by tenant and verifies monitor ownership via SQL join.
func (r *StatusPageRepository) SetMonitors(ctx context.Context, pageID uuid.UUID, monitorIDs []uuid.UUID) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		q := r.db.Querier(txCtx)
		tenantID := TenantIDFromContext(txCtx)

		// Scope DELETE by tenant via JOIN
		_, err := q.Exec(txCtx,
			`DELETE FROM status_page_monitors spm
			 USING status_pages sp
			 WHERE spm.status_page_id = sp.id
			   AND spm.status_page_id = $1
			   AND sp.tenant_id = $2`,
			pageID, tenantID)
		if err != nil {
			return fmt.Errorf("clear monitors: %w", err)
		}

		// INSERT with ownership verification via subquery
		for i, monitorID := range monitorIDs {
			result, err := q.Exec(txCtx,
				`INSERT INTO status_page_monitors (status_page_id, monitor_id, sort_order)
				 SELECT $1, m.id, $3
				 FROM monitors m
				 JOIN agents a ON m.agent_id = a.id
				 JOIN status_pages sp ON sp.id = $1
				 WHERE m.id = $2
				   AND a.user_id = sp.user_id
				   AND sp.tenant_id = $4`,
				pageID, monitorID, i, tenantID)
			if err != nil {
				return fmt.Errorf("insert monitor %s: %w", monitorID, err)
			}
			if result.RowsAffected() == 0 {
				return fmt.Errorf("monitor %s not owned by user", monitorID)
			}
		}

		return nil
	})
}

// GetMonitorIDs returns the monitor IDs for a status page.
// Defense-in-depth: scopes by tenant via JOIN on status_pages.
func (r *StatusPageRepository) GetMonitorIDs(ctx context.Context, pageID uuid.UUID) ([]uuid.UUID, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	rows, err := q.Query(ctx,
		`SELECT spm.monitor_id
		 FROM status_page_monitors spm
		 JOIN status_pages sp ON spm.status_page_id = sp.id
		 WHERE spm.status_page_id = $1 AND sp.tenant_id = $2
		 ORDER BY spm.sort_order`,
		pageID, tenantID,
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
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM status_pages WHERE user_id = $1 AND slug = $2 AND tenant_id = $3)`, userID, slug, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slug exists for user: %w", err)
	}
	return exists, nil
}
