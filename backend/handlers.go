package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"healthwatch/backend/db"
)

// Server wires the Store to HTTP routes and to the embedded frontend
// build.
type Server struct {
	store    db.Store
	mux      *http.ServeMux
	frontend http.Handler
}

// NewServer builds a Server. frontend is the embedded, already-built
// Vue app (see web.go) - in production Go serves it directly; nothing
// stops you from running the frontend separately via Vite in dev
// instead (see vite.config.js's proxy).
func NewServer(store db.Store, frontend fs.FS) *Server {
	s := &Server{
		store:    store,
		mux:      http.NewServeMux(),
		frontend: http.FileServerFS(frontend),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Registered at both the bare path (what curl/Hurl/k6 hit directly,
	// matching the spec literally) and under /api/ (what the frontend
	// calls, so the exact same fetch('/api/items') works unmodified
	// whether Vite is proxying in dev or Go is serving everything in
	// prod - see vite.config.js).
	for _, prefix := range []string{"", "/api"} {
		s.mux.HandleFunc("GET "+prefix+"/health", s.handleHealth)
		s.mux.HandleFunc("GET "+prefix+"/items", s.handleListItems)
		s.mux.HandleFunc("POST "+prefix+"/items", s.handleCreateItem)
	}

	s.mux.Handle("GET /", s.frontend)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "down",
			"error":  err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListItems(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type createItemRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	item, err := s.store.CreateItem(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
