package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"healthwatch/backend/db"
)

func TestSchedulerCheckOneSavesResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := db.NewMemoryStore()
	item, err := store.CreateItem(context.Background(), "test site", srv.URL)
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	sched := NewScheduler(store, NewChecker(2*time.Second), 2*time.Second, nil)

	updated, err := sched.CheckOne(context.Background(), item)
	if err != nil {
		t.Fatalf("CheckOne() error = %v", err)
	}

	if updated.LastStatus == nil || *updated.LastStatus != "up" {
		t.Errorf("LastStatus = %v, want \"up\"", updated.LastStatus)
	}
	if updated.LastCheckedAt == nil {
		t.Error("LastCheckedAt is nil, want a timestamp")
	}
}

func TestSchedulerRunChecksAllItemsOnEachTick(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := db.NewMemoryStore()
	if _, err := store.CreateItem(context.Background(), "site one", srv.URL); err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}
	if _, err := store.CreateItem(context.Background(), "site two", srv.URL); err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	sched := NewScheduler(store, NewChecker(1*time.Second), 1*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	// interval longer than the test's own deadline: this only exercises
	// the immediate "check everything once on start" pass.
	sched.Run(ctx, time.Hour)

	if hits.Load() < 2 {
		t.Errorf("hits = %d, want at least 2 (one per item)", hits.Load())
	}

	items, _ := store.ListItems(context.Background())
	for _, item := range items {
		if item.LastStatus == nil {
			t.Errorf("item %d has no LastStatus after Run", item.ID)
		}
	}
}
