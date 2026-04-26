package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrSettingNotFound is returned when a system_settings key has no row.
var ErrSettingNotFound = errors.New("setting not found")

// SystemSettingsRepository persists app-level key/value settings.
type SystemSettingsRepository struct {
	db *DB
}

// NewSystemSettingsRepository creates a new SystemSettingsRepository.
func NewSystemSettingsRepository(db *DB) *SystemSettingsRepository {
	return &SystemSettingsRepository{db: db}
}

// Get returns the raw JSONB value for the given key.
// Returns ErrSettingNotFound when the key is absent.
func (r *SystemSettingsRepository) Get(ctx context.Context, key string) ([]byte, error) {
	q := r.db.Querier(ctx)
	var value []byte
	err := q.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSettingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("systemSettings.Get: %w", err)
	}
	return value, nil
}

// Set upserts the value for the given key, recording who changed it.
func (r *SystemSettingsRepository) Set(ctx context.Context, key string, value []byte, updatedBy uuid.UUID) error {
	q := r.db.Querier(ctx)
	_, err := q.Exec(ctx, `
		INSERT INTO system_settings (key, value, updated_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
			SET value = EXCLUDED.value,
			    updated_at = NOW(),
			    updated_by = EXCLUDED.updated_by`,
		key, value, updatedBy,
	)
	if err != nil {
		return fmt.Errorf("systemSettings.Set: %w", err)
	}
	return nil
}
