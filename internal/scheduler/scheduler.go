// Package scheduler runs one independent ticking loop per target, so a
// slow or misconfigured target never delays the others.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"healthwatch/internal/checker"
	"healthwatch/internal/config"
	"healthwatch/internal/store"
)

// Scheduler periodically checks every configured target and saves the
// result to a Store.
type Scheduler struct {
	targets []config.Target
	checker *checker.Checker
	store   store.Store
	logger  *slog.Logger
}

// New creates a Scheduler for the given targets.
func New(targets []config.Target, c *checker.Checker, s store.Store, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{targets: targets, checker: c, store: s, logger: logger}
}

// Run starts one goroutine per target and blocks until ctx is cancelled,
// at which point every goroutine stops and Run returns.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, target := range s.targets {
		wg.Add(1)
		go func(target config.Target) {
			defer wg.Done()
			s.runTarget(ctx, target)
		}(target)
	}
	wg.Wait()
}

func (s *Scheduler) runTarget(ctx context.Context, target config.Target) {
	s.checkOnce(ctx, target)

	ticker := time.NewTicker(target.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkOnce(ctx, target)
		}
	}
}

func (s *Scheduler) checkOnce(ctx context.Context, target config.Target) {
	result := s.checker.Check(ctx, target)
	s.store.Save(result)

	if result.Status == checker.StatusUp {
		s.logger.Info("check ok", "target", target.Name, "latency_ms", result.LatencyMS, "tls_days_remaining", result.TLSDaysRemaining)
	} else {
		s.logger.Warn("check failed", "target", target.Name, "error", result.Error)
	}
}
