package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/crypto"
)

// AlertChannelRepository implements ports.AlertChannelRepository using PostgreSQL.
type AlertChannelRepository struct {
	db        *DB
	encryptor *crypto.Encryptor
}

// NewAlertChannelRepository creates a new AlertChannelRepository.
func NewAlertChannelRepository(db *DB, encryptor *crypto.Encryptor) *AlertChannelRepository {
	return &AlertChannelRepository{db: db, encryptor: encryptor}
}

// Create inserts a new alert channel with encrypted config.
func (r *AlertChannelRepository) Create(ctx context.Context, channel *domain.AlertChannel) error {
	q := r.db.Querier(ctx)

	encrypted, err := r.encryptConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("alertChannelRepo.Create: encrypt: %w", err)
	}

	query := `
		INSERT INTO alert_channels (id, user_id, type, name, config_encrypted, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = q.Exec(ctx, query,
		channel.ID,
		channel.UserID,
		string(channel.Type),
		channel.Name,
		encrypted,
		channel.Enabled,
		channel.CreatedAt,
		channel.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("alertChannelRepo.Create: %w", err)
	}

	return nil
}

// GetByID retrieves an alert channel by ID and decrypts the config.
func (r *AlertChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertChannel, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, type, name, config_encrypted, enabled, created_at, updated_at
		FROM alert_channels
		WHERE id = $1`

	channel, err := r.scanChannel(q.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("alertChannelRepo.GetByID: %w", err)
	}

	return channel, nil
}

// GetByUserID retrieves all alert channels for a user.
func (r *AlertChannelRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AlertChannel, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, type, name, config_encrypted, enabled, created_at, updated_at
		FROM alert_channels
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("alertChannelRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	var channels []*domain.AlertChannel
	for rows.Next() {
		channel, err := r.scanChannelRow(rows)
		if err != nil {
			return nil, fmt.Errorf("alertChannelRepo.GetByUserID: scan: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// GetEnabledByUserID retrieves only enabled alert channels for a user.
func (r *AlertChannelRepository) GetEnabledByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.AlertChannel, error) {
	q := r.db.Querier(ctx)

	query := `
		SELECT id, user_id, type, name, config_encrypted, enabled, created_at, updated_at
		FROM alert_channels
		WHERE user_id = $1 AND enabled = true
		ORDER BY created_at DESC`

	rows, err := q.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("alertChannelRepo.GetEnabledByUserID: %w", err)
	}
	defer rows.Close()

	var channels []*domain.AlertChannel
	for rows.Next() {
		channel, err := r.scanChannelRow(rows)
		if err != nil {
			return nil, fmt.Errorf("alertChannelRepo.GetEnabledByUserID: scan: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// Update updates an alert channel's name, enabled status, and config.
func (r *AlertChannelRepository) Update(ctx context.Context, channel *domain.AlertChannel) error {
	q := r.db.Querier(ctx)

	encrypted, err := r.encryptConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("alertChannelRepo.Update: encrypt: %w", err)
	}

	query := `
		UPDATE alert_channels
		SET name = $1, config_encrypted = $2, enabled = $3, updated_at = NOW()
		WHERE id = $4`

	result, err := q.Exec(ctx, query, channel.Name, encrypted, channel.Enabled, channel.ID)
	if err != nil {
		return fmt.Errorf("alertChannelRepo.Update: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("alertChannelRepo.Update: channel not found")
	}

	return nil
}

// Delete removes an alert channel.
func (r *AlertChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := r.db.Querier(ctx)

	result, err := q.Exec(ctx, `DELETE FROM alert_channels WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("alertChannelRepo.Delete(%s): %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("alertChannelRepo.Delete(%s): channel not found", id)
	}

	return nil
}

// encryptConfig marshals config to JSON and encrypts it.
func (r *AlertChannelRepository) encryptConfig(config map[string]string) ([]byte, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return r.encryptor.Encrypt(data)
}

// decryptConfig decrypts and unmarshals config from the database.
func (r *AlertChannelRepository) decryptConfig(encrypted []byte) (map[string]string, error) {
	data, err := r.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return config, nil
}

// scanChannel scans a single row into an AlertChannel.
func (r *AlertChannelRepository) scanChannel(row pgx.Row) (*domain.AlertChannel, error) {
	var ch domain.AlertChannel
	var channelType string
	var encrypted []byte

	if err := row.Scan(
		&ch.ID, &ch.UserID, &channelType, &ch.Name,
		&encrypted, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
	); err != nil {
		return nil, err
	}

	ch.Type = domain.AlertChannelType(channelType)

	config, err := r.decryptConfig(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	ch.Config = config

	return &ch, nil
}

// scanChannelRow scans a row from pgx.Rows into an AlertChannel.
func (r *AlertChannelRepository) scanChannelRow(rows pgx.Rows) (*domain.AlertChannel, error) {
	var ch domain.AlertChannel
	var channelType string
	var encrypted []byte

	if err := rows.Scan(
		&ch.ID, &ch.UserID, &channelType, &ch.Name,
		&encrypted, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
	); err != nil {
		return nil, err
	}

	ch.Type = domain.AlertChannelType(channelType)

	config, err := r.decryptConfig(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	ch.Config = config

	return &ch, nil
}
