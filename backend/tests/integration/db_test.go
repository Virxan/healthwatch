//go:build integration

// Package integration runs PGStore against a real, throwaway PostgreSQL
// container via Testcontainers - the things a fake Store can't catch
// (actual SQL, actual driver behaviour, actual default values like
// created_at). Requires Docker. Run with:
//
//	go test -tags integration ./tests/integration/...
package integration

import (
	"context"
	"testing"

	"healthwatch/backend/db"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPGStoreAgainstRealPostgres(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("healthwatch_test"),
		postgres.WithUsername("healthwatch"),
		postgres.WithPassword("healthwatch"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	store, err := db.NewPGStore(ctx, connStr)
	if err != nil {
		t.Fatalf("connecting PGStore to the test container: %v", err)
	}
	t.Cleanup(store.Close)

	t.Run("Ping succeeds against a real database", func(t *testing.T) {
		if err := store.Ping(ctx); err != nil {
			t.Errorf("Ping() = %v, want nil", err)
		}
	})

	t.Run("CreateItem then ListItems round-trips through real Postgres", func(t *testing.T) {
		created, err := store.CreateItem(ctx, "integration test item", "https://example.com")
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}
		if created.ID == 0 {
			t.Error("created.ID is zero, want a non-zero ID assigned by Postgres")
		}
		if created.URL != "https://example.com" {
			t.Errorf("created.URL = %q, want %q", created.URL, "https://example.com")
		}
		if created.CreatedAt.IsZero() {
			t.Error("created.CreatedAt is zero, want a timestamp set by Postgres' default now()")
		}
		if created.LastStatus != nil {
			t.Errorf("created.LastStatus = %v, want nil before any check has run", *created.LastStatus)
		}

		items, err := store.ListItems(ctx)
		if err != nil {
			t.Fatalf("ListItems() error = %v", err)
		}

		found := false
		for _, item := range items {
			if item.ID == created.ID && item.Name == "integration test item" {
				found = true
			}
		}
		if !found {
			t.Errorf("created item not found in ListItems(): %+v", items)
		}
	})

	t.Run("SaveCheckResult persists status, latency and TLS days remaining", func(t *testing.T) {
		created, err := store.CreateItem(ctx, "checked item", "https://example.com")
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		days := 30
		updated, err := store.SaveCheckResult(ctx, created.ID, db.CheckResult{
			Status:           "up",
			HTTPStatus:       200,
			LatencyMS:        42,
			TLSDaysRemaining: &days,
		})
		if err != nil {
			t.Fatalf("SaveCheckResult() error = %v", err)
		}

		if updated.LastStatus == nil || *updated.LastStatus != "up" {
			t.Errorf("LastStatus = %v, want \"up\"", updated.LastStatus)
		}
		if updated.LastHTTPStatus == nil || *updated.LastHTTPStatus != 200 {
			t.Errorf("LastHTTPStatus = %v, want 200", updated.LastHTTPStatus)
		}
		if updated.TLSDaysRemaining == nil || *updated.TLSDaysRemaining != 30 {
			t.Errorf("TLSDaysRemaining = %v, want 30", updated.TLSDaysRemaining)
		}
		if updated.LastCheckedAt == nil {
			t.Error("LastCheckedAt is nil, want a timestamp set by Postgres' now()")
		}
	})

	t.Run("migrate is idempotent across a second PGStore on the same database", func(t *testing.T) {
		second, err := db.NewPGStore(ctx, connStr)
		if err != nil {
			t.Fatalf("opening a second PGStore against the same database: %v", err)
		}
		defer second.Close()

		if err := second.Ping(ctx); err != nil {
			t.Errorf("Ping() on second store = %v, want nil", err)
		}
	})
}
