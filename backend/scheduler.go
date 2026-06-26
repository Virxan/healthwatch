package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"healthwatch/backend/db"
)

// Scheduler periodically re-checks every item's URL and saves the
// result. Unlike a static config file, the watch list is whatever's
// currently in the "items" table - items added through the API are
// picked up on the next tick automatically.
type Scheduler struct {
	store   db.Store
	checker *Checker
	logger  *slog.Logger
	timeout time.Duration
}

// NewScheduler creates a Scheduler. timeout bounds how long a single
// item's check may take.
func NewScheduler(store db.Store, checker *Checker, timeout time.Duration, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{store: store, checker: checker, timeout: timeout, logger: logger}
}

// Run checks every item once immediately, then every interval, until
// ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	s.checkAll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAll(ctx)
		}
	}
}

func (s *Scheduler) checkAll(ctx context.Context) {
	items, err := s.store.ListItems(ctx)
	if err != nil {
		s.logger.Error("scheduler: listing items failed", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(item db.Item) {
			defer wg.Done()
			s.checkOne(ctx, item)
		}(item)
	}
	wg.Wait()
}

// CheckOne checks a single item immediately and saves the result. Used
// both by the scheduler's own loop and by the API handler so a newly
// created item gets feedback right away instead of waiting for the
// next tick.
func (s *Scheduler) CheckOne(ctx context.Context, item db.Item) (db.Item, error) {
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	result := s.checker.Check(checkCtx, item.URL)

	updated, err := s.store.SaveCheckResult(ctx, item.ID, result)
	if err != nil {
		s.logger.Error("scheduler: saving check result failed", "item_id", item.ID, "error", err)
		return db.Item{}, err
	}

	if result.Status == "up" {
		s.logger.Info("check ok", "item_id", item.ID, "url", item.URL, "latency_ms", result.LatencyMS)
	} else {
		s.logger.Warn("check failed", "item_id", item.ID, "url", item.URL, "error", result.Error)
	}

	return updated, nil
}

func (s *Scheduler) checkOne(ctx context.Context, item db.Item) {
	_, _ = s.CheckOne(ctx, item)
}
