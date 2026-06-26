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

// Item is a single row of the "items" table - in Healthwatch, an item
// is a website being watched. Name/URL/CreatedAt are always set; the
// Last* fields are nil until the scheduler (or an immediate check on
// creation) has actually checked the URL at least once.
type Item struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`

	LastStatus       *string    `json:"last_status,omitempty"`
	LastHTTPStatus   *int       `json:"last_http_status,omitempty"`
	LastLatencyMS    *int64     `json:"last_latency_ms,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	TLSDaysRemaining *int       `json:"tls_days_remaining,omitempty"`
	LastError        *string    `json:"last_error,omitempty"`
}

// CheckResult is the outcome of checking one item's URL once. It is
// produced by the checker (see checker.go) and persisted onto the item
// via Store.SaveCheckResult.
type CheckResult struct {
	Status           string
	HTTPStatus       int
	LatencyMS        int64
	TLSDaysRemaining *int
	Error            string
}

// Store is everything the HTTP handlers and the background scheduler
// need from a backing database. It exists so handler logic can be
// unit-tested against an in-memory fake (see MemoryStore) without a
// real Postgres, while tests/integration exercises the real PGStore
// against one via Testcontainers.
type Store interface {
	// Ping reports whether the database is reachable. Used by GET /health.
	Ping(ctx context.Context) error
	ListItems(ctx context.Context) ([]Item, error)
	CreateItem(ctx context.Context, name, url string) (Item, error)
	// SaveCheckResult records the outcome of checking an item's URL and
	// returns the item with the result applied.
	SaveCheckResult(ctx context.Context, itemID int64, result CheckResult) (Item, error)
	// DeleteAllItems removes every item and returns how many were deleted.
	// Used by DELETE /items to reset the dashboard.
	DeleteAllItems(ctx context.Context) (int64, error)
}
