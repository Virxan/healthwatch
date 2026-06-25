package db

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-process Store used by unit tests so handler logic
// can be tested without a real Postgres - see tests/integration for the
// tests that exercise PGStore against a real database via Testcontainers.
type MemoryStore struct {
	mu        sync.Mutex
	items     []Item
	nextID    int64
	pingError error // set to simulate a database that's down
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1}
}

// SetPingError makes subsequent Ping calls fail with err (or succeed, if
// err is nil) - used to test the unhealthy path of GET /health.
func (s *MemoryStore) SetPingError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pingError = err
}

// Ping implements Store. It never fails unless SetPingError was used to
// simulate a database that's down.
func (s *MemoryStore) Ping(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pingError
}

// ListItems implements Store.
func (s *MemoryStore) ListItems(_ context.Context) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out, nil
}

// CreateItem implements Store.
func (s *MemoryStore) CreateItem(_ context.Context, name string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Item{
		ID:        s.nextID,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.items = append(s.items, item)
	return item, nil
}
