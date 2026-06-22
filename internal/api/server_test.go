package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"healthwatch/internal/api"
	"healthwatch/internal/checker"
	"healthwatch/internal/store"
)

func TestHealthz(t *testing.T) {
	srv := api.New(store.NewMemory())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestListChecks(t *testing.T) {
	s := store.NewMemory()
	s.Save(checker.Result{Target: "example", Status: checker.StatusUp, CheckedAt: time.Now()})

	srv := api.New(s)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checks", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var results []checker.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("response is not a valid JSON array: %v", err)
	}
	if len(results) != 1 || results[0].Target != "example" {
		t.Errorf("unexpected results: %+v", results)
	}
}

func TestGetCheckFound(t *testing.T) {
	s := store.NewMemory()
	s.Save(checker.Result{Target: "example", Status: checker.StatusUp})

	srv := api.New(s)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checks/example", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestGetCheckNotFound(t *testing.T) {
	srv := api.New(store.NewMemory())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checks/missing", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDashboardRenders(t *testing.T) {
	s := store.NewMemory()
	s.Save(checker.Result{Target: "example", Status: checker.StatusUp, CheckedAt: time.Now()})

	srv := api.New(s)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}
