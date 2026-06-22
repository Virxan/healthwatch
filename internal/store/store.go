// Package store keeps the most recent check Result for every target in
// memory. It is intentionally dependency-free: no database driver, no
// cgo, nothing that would bloat the static binary or its SBOM.
package store

import (
	"sort"
	"sync"

	"healthwatch/internal/checker"
)

// Store is the contract the API and scheduler depend on, so a different
// backend (e.g. a persistent one) can be swapped in later without
// touching either of them.
type Store interface {
	Save(result checker.Result)
	All() []checker.Result
	Get(name string) (checker.Result, bool)
}

// Memory is an in-process, thread-safe Store.
type Memory struct {
	mu      sync.RWMutex
	results map[string]checker.Result
}

// NewMemory creates an empty in-memory Store.
func NewMemory() *Memory {
	return &Memory{results: make(map[string]checker.Result)}
}

// Save records the latest result for its target, overwriting any
// previous one.
func (m *Memory) Save(result checker.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[result.Target] = result
}

// All returns every stored result, sorted by target name for stable API
// and dashboard output.
func (m *Memory) All() []checker.Result {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]checker.Result, 0, len(m.results))
	for _, r := range m.results {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Target < out[j].Target })
	return out
}

// Get returns the latest result for a single target, if any.
func (m *Memory) Get(name string) (checker.Result, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.results[name]
	return r, ok
}
