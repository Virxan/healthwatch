package scheduler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"healthwatch/internal/checker"
	"healthwatch/internal/config"
	"healthwatch/internal/scheduler"
	"healthwatch/internal/store"
)

func TestSchedulerRunsImmediatelyAndOnInterval(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := config.Target{
		Name:            "fast",
		URL:             srv.URL,
		IntervalSeconds: 1,
		TimeoutSeconds:  1,
	}

	s := store.NewMemory()
	c := checker.New(1 * time.Second)
	sched := scheduler.New([]config.Target{target}, c, s, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	sched.Run(ctx)

	result, ok := s.Get("fast")
	if !ok {
		t.Fatal("expected at least one result to be saved")
	}
	if result.Status != checker.StatusUp {
		t.Errorf("Status = %q, want %q", result.Status, checker.StatusUp)
	}
	if hits == 0 {
		t.Error("expected the target server to have been hit at least once")
	}
}

func TestSchedulerStopsWhenContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := config.Target{Name: "stoppable", URL: srv.URL, IntervalSeconds: 60, TimeoutSeconds: 1}
	s := store.NewMemory()
	c := checker.New(1 * time.Second)
	sched := scheduler.New([]config.Target{target}, c, s, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Run returned promptly after cancellation, as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s of context cancellation")
	}
}
