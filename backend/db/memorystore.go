package db

import (
	"context"
	"fmt"
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
func (s *MemoryStore) CreateItem(_ context.Context, name, url string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := Item{
		ID:        s.nextID,
		Name:      name,
		URL:       url,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.items = append(s.items, item)
	return item, nil
}

// SaveCheckResult implements Store.
func (s *MemoryStore) SaveCheckResult(_ context.Context, itemID int64, result CheckResult) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.items {
		if s.items[i].ID != itemID {
			continue
		}

		now := time.Now().UTC()
		status, httpStatus, latency := result.Status, result.HTTPStatus, result.LatencyMS
		s.items[i].LastStatus = &status
		s.items[i].LastHTTPStatus = &httpStatus
		s.items[i].LastLatencyMS = &latency
		s.items[i].LastCheckedAt = &now
		s.items[i].TLSDaysRemaining = result.TLSDaysRemaining
		if result.Error != "" {
			errCopy := result.Error
			s.items[i].LastError = &errCopy
		} else {
			s.items[i].LastError = nil
		}
		return s.items[i], nil
	}

	return Item{}, fmt.Errorf("item %d not found", itemID)
}
