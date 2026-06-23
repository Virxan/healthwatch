// Package db holds the "items" data access layer: the Store interface,
// the real Postgres-backed implementation (PGStore), and an in-memory
// fake (MemoryStore) used by the backend's own unit tests. It is its
// own importable package - rather than living in package main - so that
// tests/integration/db_test.go (a different directory, and therefore
// necessarily a different package) can import and exercise PGStore
// directly against a real database via Testcontainers.
package db

import (
	"context"
	"time"
)

// Item is a single row of the "items" table.
type Item struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is everything the HTTP handlers need from a backing database.
// It exists so handler logic can be unit-tested against an in-memory
// fake (see MemoryStore) without a real Postgres, while
// tests/integration exercises the real PGStore against one via
// Testcontainers.
type Store interface {
	// Ping reports whether the database is reachable. Used by GET /health.
	Ping(ctx context.Context) error
	ListItems(ctx context.Context) ([]Item, error)
	CreateItem(ctx context.Context, name string) (Item, error)
}
