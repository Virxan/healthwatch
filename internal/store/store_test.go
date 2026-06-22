package store_test

import (
	"testing"
	"time"

	"healthwatch/internal/checker"
	"healthwatch/internal/store"
)

func TestSaveAndGet(t *testing.T) {
	s := store.NewMemory()

	result := checker.Result{Target: "example", Status: checker.StatusUp, CheckedAt: time.Now()}
	s.Save(result)

	got, ok := s.Get("example")
	if !ok {
		t.Fatal("expected target 'example' to be found")
	}
	if got.Status != checker.StatusUp {
		t.Errorf("Status = %q, want %q", got.Status, checker.StatusUp)
	}
}

func TestGetMissing(t *testing.T) {
	s := store.NewMemory()
	if _, ok := s.Get("missing"); ok {
		t.Error("expected ok=false for a target that was never saved")
	}
}

func TestSaveOverwrites(t *testing.T) {
	s := store.NewMemory()
	s.Save(checker.Result{Target: "example", Status: checker.StatusUp})
	s.Save(checker.Result{Target: "example", Status: checker.StatusDown})

	got, _ := s.Get("example")
	if got.Status != checker.StatusDown {
		t.Errorf("Status = %q, want %q after overwrite", got.Status, checker.StatusDown)
	}
}

func TestAllIsSortedByTarget(t *testing.T) {
	s := store.NewMemory()
	s.Save(checker.Result{Target: "zebra"})
	s.Save(checker.Result{Target: "alpha"})
	s.Save(checker.Result{Target: "mike"})

	all := s.All()
	if len(all) != 3 {
		t.Fatalf("len(All()) = %d, want 3", len(all))
	}
	want := []string{"alpha", "mike", "zebra"}
	for i, target := range want {
		if all[i].Target != target {
			t.Errorf("All()[%d].Target = %q, want %q", i, all[i].Target, target)
		}
	}
}

func TestAllOnEmptyStore(t *testing.T) {
	s := store.NewMemory()
	if all := s.All(); len(all) != 0 {
		t.Errorf("len(All()) = %d, want 0 on an empty store", len(all))
	}
}
