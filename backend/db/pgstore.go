package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore is the real, Postgres-backed Store implementation.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore connects to databaseURL, verifies the connection, and
// ensures the "items" table exists.
func NewPGStore(ctx context.Context, databaseURL string) (*PGStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	store := &PGStore{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return store, nil
}

// migrate creates the "items" table if it doesn't already exist. A
// project this size doesn't need a migration framework - one
// idempotent statement, run on every startup, is enough.
func (s *PGStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS items (
			id         BIGSERIAL PRIMARY KEY,
			name       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

// Close releases the connection pool. Call this once on shutdown.
func (s *PGStore) Close() {
	s.pool.Close()
}

// Ping implements Store.
func (s *PGStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// ListItems implements Store.
func (s *PGStore) ListItems(ctx context.Context) ([]Item, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, created_at FROM items ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying items: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning item row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading item rows: %w", err)
	}

	return items, nil
}

// CreateItem implements Store.
func (s *PGStore) CreateItem(ctx context.Context, name string) (Item, error) {
	var item Item
	err := s.pool.QueryRow(
		ctx,
		`INSERT INTO items (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("inserting item: %w", err)
	}
	return item, nil
}
