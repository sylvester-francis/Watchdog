package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
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
	tenantID := TenantIDFromContext(ctx)

	query := `
		INSERT INTO users (id, email, username, password_hash, plan, is_admin, created_at, updated_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		string(user.Plan),
		user.IsAdmin,
		user.CreatedAt,
		user.UpdatedAt,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, email, username, password_hash, plan, is_admin, tenant_id, created_at, updated_at
		FROM users
		WHERE id = $1 AND tenant_id = $2`

	user := &domain.User{}
	err := q.QueryRow(ctx, query, id, tenantID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Plan,
		&user.IsAdmin,
		&user.TenantID,
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
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, email, username, password_hash, plan, is_admin, tenant_id, created_at, updated_at
		FROM users
		WHERE email = $1 AND tenant_id = $2`

	user := &domain.User{}
	err := q.QueryRow(ctx, query, email, tenantID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Plan,
		&user.IsAdmin,
		&user.TenantID,
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

// GetByUsername retrieves a user by their username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT id, email, username, password_hash, plan, is_admin, tenant_id, created_at, updated_at
		FROM users
		WHERE username = $1 AND tenant_id = $2`

	user := &domain.User{}
	err := q.QueryRow(ctx, query, username, tenantID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Plan,
		&user.IsAdmin,
		&user.TenantID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.GetByUsername(%s): %w", username, err)
	}

	return user, nil
}

// Update updates an existing user in the database.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		UPDATE users
		SET email = $2, username = $3, password_hash = $4, plan = $5, is_admin = $6, updated_at = $7
		WHERE id = $1 AND tenant_id = $8`

	result, err := q.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		string(user.Plan),
		user.IsAdmin,
		user.UpdatedAt,
		tenantID,
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
	tenantID := TenantIDFromContext(ctx)

	query := `DELETE FROM users WHERE id = $1 AND tenant_id = $2`

	result, err := q.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("userRepo.Delete(%s): %w", id, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("userRepo.Delete(%s): user not found", id)
	}

	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	var count int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("userRepo.Count: %w", err)
	}

	return count, nil
}

// CountByPlan returns the number of users per plan type.
func (r *UserRepository) CountByPlan(ctx context.Context) (map[domain.Plan]int, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT plan, COUNT(*) FROM users WHERE tenant_id = $1 GROUP BY plan`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.CountByPlan: %w", err)
	}
	defer rows.Close()

	result := make(map[domain.Plan]int)
	for rows.Next() {
		var plan domain.Plan
		var count int
		if err := rows.Scan(&plan, &count); err != nil {
			return nil, fmt.Errorf("userRepo.CountByPlan: scan: %w", err)
		}
		result[plan] = count
	}

	return result, rows.Err()
}

// GetUsersNearLimits returns users who are at or near their plan limits (80%+).
func (r *UserRepository) GetUsersNearLimits(ctx context.Context) ([]ports.UserUsageSummary, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT u.email, u.plan,
			COALESCE(ac.cnt, 0) AS agent_count,
			COALESCE(mc.cnt, 0) AS monitor_count
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS cnt FROM agents WHERE tenant_id = $1 GROUP BY user_id
		) ac ON ac.user_id = u.id
		LEFT JOIN (
			SELECT a.user_id, COUNT(*) AS cnt
			FROM monitors m JOIN agents a ON m.agent_id = a.id
			WHERE m.tenant_id = $1
			GROUP BY a.user_id
		) mc ON mc.user_id = u.id
		WHERE u.tenant_id = $1
		ORDER BY u.email`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetUsersNearLimits: %w", err)
	}
	defer rows.Close()

	var results []ports.UserUsageSummary
	for rows.Next() {
		var s ports.UserUsageSummary
		if err := rows.Scan(&s.Email, &s.Plan, &s.AgentCount, &s.MonitorCount); err != nil {
			return nil, fmt.Errorf("userRepo.GetUsersNearLimits: scan: %w", err)
		}
		limits := s.Plan.Limits()
		s.AgentMax = limits.MaxAgents
		s.MonitorMax = limits.MaxMonitors

		// Only include users at 80%+ of either limit
		agentNear := limits.MaxAgents > 0 && float64(s.AgentCount) >= float64(limits.MaxAgents)*0.8
		monitorNear := limits.MaxMonitors > 0 && float64(s.MonitorCount) >= float64(limits.MaxMonitors)*0.8
		if agentNear || monitorNear {
			results = append(results, s)
		}
	}

	return results, rows.Err()
}

// GetAllWithUsage returns all users with their agent and monitor counts.
func (r *UserRepository) GetAllWithUsage(ctx context.Context) ([]ports.AdminUserView, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `
		SELECT u.id, u.email, u.plan, u.is_admin,
			COALESCE(ac.cnt, 0) AS agent_count,
			COALESCE(mc.cnt, 0) AS monitor_count,
			u.created_at
		FROM users u
		LEFT JOIN (
			SELECT user_id, COUNT(*) AS cnt FROM agents WHERE tenant_id = $1 GROUP BY user_id
		) ac ON ac.user_id = u.id
		LEFT JOIN (
			SELECT a.user_id, COUNT(*) AS cnt
			FROM monitors m JOIN agents a ON m.agent_id = a.id
			WHERE m.tenant_id = $1
			GROUP BY a.user_id
		) mc ON mc.user_id = u.id
		WHERE u.tenant_id = $1
		ORDER BY u.created_at DESC`

	rows, err := q.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetAllWithUsage: %w", err)
	}
	defer rows.Close()

	var results []ports.AdminUserView
	for rows.Next() {
		var u ports.AdminUserView
		if err := rows.Scan(&u.ID, &u.Email, &u.Plan, &u.IsAdmin, &u.AgentCount, &u.MonitorCount, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("userRepo.GetAllWithUsage: scan: %w", err)
		}
		limits := u.Plan.Limits()
		u.AgentMax = limits.MaxAgents
		u.MonitorMax = limits.MaxMonitors
		results = append(results, u)
	}

	return results, rows.Err()
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND tenant_id = $2)`

	var exists bool
	err := q.QueryRow(ctx, query, email, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepo.ExistsByEmail(%s): %w", email, err)
	}

	return exists, nil
}

// UsernameExists checks if a username is already taken.
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	q := r.db.Querier(ctx)
	tenantID := TenantIDFromContext(ctx)

	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND tenant_id = $2)`, username, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepo.UsernameExists(%s): %w", username, err)
	}

	return exists, nil
}
