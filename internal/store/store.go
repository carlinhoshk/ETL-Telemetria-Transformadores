// Package store is the PostgreSQL access layer: pool management plus the
// repositories used by the ingestion hot path and the API (Phases 5/6/11).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned by single-row lookups when the key is absent.
var ErrNotFound = errors.New("not found")

// DB wraps a pgx connection pool with the platform's repositories.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL and verifies the pool with a ping.
func Open(ctx context.Context, dsn string) (*DB, error) {
	if dsn == "" {
		return nil, errors.New("store: empty database dsn")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("store: parse dsn: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: create pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close releases the pool.
func (d *DB) Close() { d.pool.Close() }

// Ping reports database reachability (used by health checks).
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Pool exposes the underlying pool for advanced queries.
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

// IsUniqueViolation reports whether err is a Postgres unique constraint
// violation (used for idempotent ingestion).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsNoRows reports whether err is a not-found row scan.
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
