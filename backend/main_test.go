package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"healthwatch/backend/db"
)

//go:embed testdata/empty
var emptyFS embed.FS

func newTestServer() *Server {
	return NewServer(db.NewMemoryStore(), emptyFS)
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
	srv := NewServer(store, emptyFS)

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

	createReq := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`{"name":"first item"}`))
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
	req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(`{"name":"   "}`))
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
	req := httptest.NewRequest(http.MethodPost, "/api/items", bytes.NewBufferString(`{"name":"via api prefix"}`))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
}
