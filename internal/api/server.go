// Package api exposes the current check results over HTTP: a JSON API
// for machines (and for the Cucumber contract tests) and a minimal HTML
// dashboard for humans.
package api

import (
	"encoding/json"
	"html/template"
	"net/http"

	"healthwatch/internal/store"
)

// Server serves the Healthwatch HTTP API and dashboard.
type Server struct {
	store store.Store
	mux   *http.ServeMux
	tmpl  *template.Template
}

// New creates a Server backed by the given Store.
func New(s store.Store) *Server {
	srv := &Server{
		store: s,
		mux:   http.NewServeMux(),
		tmpl:  template.Must(template.New("dashboard").Parse(dashboardTemplate)),
	}
	srv.routes()
	return srv
}

// ServeHTTP makes Server an http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /api/v1/checks", s.handleListChecks)
	s.mux.HandleFunc("GET /api/v1/checks/{name}", s.handleGetCheck)
	s.mux.HandleFunc("GET /", s.handleDashboard)
}

// handleHealthz is the liveness/readiness probe target for Kubernetes -
// it reports on the Healthwatch process itself, not on the targets it
// monitors.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleListChecks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.All())
}

func (s *Server) handleGetCheck(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	result, ok := s.store.Get(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target not found"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, s.store.All()); err != nil {
		http.Error(w, "rendering dashboard", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

const dashboardTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Healthwatch</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 2rem; background: #fafafa; color: #1a1a1a; }
  h1 { margin-bottom: 0.25rem; }
  table { border-collapse: collapse; width: 100%; background: white; }
  th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid #e2e2e2; }
  .up { color: #1a7f37; font-weight: 600; }
  .down { color: #cf222e; font-weight: 600; }
</style>
</head>
<body>
<h1>Healthwatch</h1>
<p>{{len .}} target(s) monitored.</p>
<table>
  <tr><th>Target</th><th>Status</th><th>HTTP</th><th>Latency</th><th>TLS expiry</th><th>Checked at</th></tr>
  {{range .}}
  <tr>
    <td>{{.Target}}</td>
    <td class="{{.Status}}">{{.Status}}</td>
    <td>{{.HTTPStatusCode}}</td>
    <td>{{.LatencyMS}} ms</td>
    <td>{{if .TLSDaysRemaining}}{{.TLSDaysRemaining}} days{{else}}-{{end}}</td>
    <td>{{.CheckedAt.Format "2006-01-02 15:04:05"}}</td>
  </tr>
  {{end}}
</table>
</body>
</html>
`
