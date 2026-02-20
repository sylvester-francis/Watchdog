package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sylvester-francis/watchdog/internal/config"
)

// txKey is the context key for storing a transaction.
type txKey struct{}

// tenantKey is the context key for storing the tenant ID.
type tenantKey struct{}

// WithTenantID returns a context with the given tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenantID)
}

// TenantIDFromContext extracts the tenant ID from context.
// Returns "default" if no tenant ID is set (single-tenant mode).
func TenantIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(tenantKey{}).(string); ok && id != "" {
		return id
	}
	return "default"
}

// Querier is an interface that both pgxpool.Pool and pgx.Tx satisfy.
// This allows repositories to work with or without transactions transparently.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// CopyFromSource is an alias for pgx.CopyFromSource for bulk inserts.
type CopyFromSource = pgx.CopyFromSource

// DB wraps the PostgreSQL connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a new database connection pool.
func NewDB(ctx context.Context, cfg config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Health checks if the database connection is healthy.
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics.
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// WithTransaction executes fn within a database transaction.
// If fn returns an error, the transaction is rolled back.
// If fn succeeds, the transaction is committed.
func (db *DB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Inject transaction into context
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Execute the function
	if err := fn(txCtx); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Querier returns a Querier from the context (transaction) or the pool.
// Repositories should use this to get a database handle that works
// both inside and outside transactions.
func (db *DB) Querier(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.Pool
}

// CopyFrom performs a bulk insert using PostgreSQL's COPY protocol.
// This method uses the pool directly as COPY doesn't work within
// a transaction in the same way as regular queries.
func (db *DB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc CopyFromSource) (int64, error) {
	// Check if we're in a transaction
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return db.Pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}
