package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"healthwatch/backend/db"
)

//go:embed testdata/empty
var emptyFS embed.FS

// newTestServer builds a Server backed by a fresh MemoryStore and a
// real Scheduler/Checker - the Checker's own HTTP calls during tests
// always target newTestTargetServer's URL, so they're fast and don't
// touch the network.
func newTestServer() *Server {
	store := db.NewMemoryStore()
	sched := NewScheduler(store, NewChecker(2*time.Second), 2*time.Second, nil)
	return NewServer(store, sched, emptyFS)
}

// newTestTargetServer is a stand-in "website" for create-item tests
// that need a real, reachable URL.
func newTestTargetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHealthOK(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want %q", body["status"], "ok")
	}
}

func TestHealthDown(t *testing.T) {
	store := db.NewMemoryStore()
	store.SetPingError(errors.New("connection refused"))
	sched := NewScheduler(store, NewChecker(2*time.Second), 2*time.Second, nil)
	srv := NewServer(store, sched, emptyFS)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestHealthAlsoServedUnderAPIPrefix(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestListItemsEmpty(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var items []db.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestCreateAndListItem(t *testing.T) {
	srv := newTestServer()
	target := newTestTargetServer(t)

	body := fmt.Sprintf(`{"name":"first item","url":%q}`, target.URL)
	createReq := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(body))
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body: %s)", createRec.Code, createRec.Body.String())
	}

	var created db.Item
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("create response is not valid JSON: %v", err)
	}
	if created.Name != "first item" {
		t.Errorf("created.Name = %q, want %q", created.Name, "first item")
	}
	if created.ID == 0 {
		t.Error("created.ID is zero, want a non-zero ID")
	}
	if created.LastStatus == nil || *created.LastStatus != "up" {
		t.Errorf("created.LastStatus = %v, want \"up\" (the immediate check on create should have run)", created.LastStatus)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/items", nil)
	listRec := httptest.NewRecorder()
	srv.ServeHTTP(listRec, listReq)

	var items []db.Item
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("list response is not a JSON array: %v", err)
	}
	if len(items) != 1 || items[0].Name != "first item" {
		t.Errorf("unexpected items after create: %+v", items)
	}
}

func TestCreateItemRejectsEmptyName(t *testing.T) {
	srv := newTestServer()
	target := newTestTargetServer(t)

	body := fmt.Sprintf(`{"name":"   ","url":%q}`, target.URL)
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateItemRejectsMissingURL(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`{"name":"no url"}`))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateItemRejectsInvalidURL(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`{"name":"bad url","url":"not-a-url"}`))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateItemRejectsInvalidJSON(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`not json`))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateItemAlsoServedUnderAPIPrefix(t *testing.T) {
	srv := newTestServer()
	target := newTestTargetServer(t)

	body := fmt.Sprintf(`{"name":"via api prefix","url":%q}`, target.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/items", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
}
