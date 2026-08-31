// Package storage reads and writes the application data in Postgres.
package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("storage: not found")

// DB is a handle to the Postgres database.
type DB struct {
	pool *pgxpool.Pool
}

// Connect opens a connection pool and verifies that the database answers.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Pool exposes the underlying pool for libraries that need it, such as the
// session store.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close releases all pooled connections.
func (db *DB) Close() { db.pool.Close() }

// wrapNotFound converts pgx's no-rows sentinel into ErrNotFound.
func wrapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
