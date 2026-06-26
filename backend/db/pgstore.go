package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore is the real, Postgres-backed Store implementation.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore connects to databaseURL, verifies the connection, and
// ensures the "items" table exists (and is up to date).
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

// migrate creates the "items" table if it doesn't already exist, and
// adds any columns introduced since (each as a separate, idempotent
// statement) so an existing database upgrades in place. A project this
// size doesn't need a full migration framework - this runs on every
// startup and is safe to repeat.
func (s *PGStore) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS items (
			id         BIGSERIAL PRIMARY KEY,
			name       TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS last_status TEXT`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS last_http_status INT`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS last_latency_ms BIGINT`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS last_checked_at TIMESTAMPTZ`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS tls_days_remaining INT`,
		`ALTER TABLE items ADD COLUMN IF NOT EXISTS last_error TEXT`,
	}
	for _, stmt := range statements {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("running migration statement %q: %w", stmt, err)
		}
	}
	return nil
}

// Close releases the connection pool. Call this once on shutdown.
func (s *PGStore) Close() {
	s.pool.Close()
}

// Ping implements Store.
func (s *PGStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

const itemColumns = `
	id, name, url, created_at,
	last_status, last_http_status, last_latency_ms, last_checked_at,
	tls_days_remaining, last_error
`

func scanItem(row pgx.Row) (Item, error) {
	var item Item
	err := row.Scan(
		&item.ID, &item.Name, &item.URL, &item.CreatedAt,
		&item.LastStatus, &item.LastHTTPStatus, &item.LastLatencyMS, &item.LastCheckedAt,
		&item.TLSDaysRemaining, &item.LastError,
	)
	return item, err
}

// ListItems implements Store.
func (s *PGStore) ListItems(ctx context.Context) ([]Item, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+itemColumns+` FROM items ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying items: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
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
func (s *PGStore) CreateItem(ctx context.Context, name, url string) (Item, error) {
	row := s.pool.QueryRow(
		ctx,
		`INSERT INTO items (name, url) VALUES ($1, $2) RETURNING `+itemColumns,
		name, url,
	)
	item, err := scanItem(row)
	if err != nil {
		return Item{}, fmt.Errorf("inserting item: %w", err)
	}
	return item, nil
}

// SaveCheckResult implements Store.
func (s *PGStore) SaveCheckResult(ctx context.Context, itemID int64, result CheckResult) (Item, error) {
	var lastError *string
	if result.Error != "" {
		lastError = &result.Error
	}

	row := s.pool.QueryRow(
		ctx, `
		UPDATE items SET
			last_status = $1,
			last_http_status = $2,
			last_latency_ms = $3,
			last_checked_at = now(),
			tls_days_remaining = $4,
			last_error = $5
		WHERE id = $6
		RETURNING `+itemColumns,
		result.Status, result.HTTPStatus, result.LatencyMS, result.TLSDaysRemaining, lastError, itemID,
	)
	item, err := scanItem(row)
	if err != nil {
		return Item{}, fmt.Errorf("saving check result for item %d: %w", itemID, err)
	}
	return item, nil
}

// DeleteAllItems implements Store.
func (s *PGStore) DeleteAllItems(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM items`)
	if err != nil {
		return 0, fmt.Errorf("deleting all items: %w", err)
	}
	return tag.RowsAffected(), nil
}
