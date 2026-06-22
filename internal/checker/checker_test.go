package checker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"healthwatch/internal/checker"
	"healthwatch/internal/config"
)

func TestCheckUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := checker.New(2 * time.Second)
	target := config.Target{Name: "up-test", URL: srv.URL, TimeoutSeconds: 2}

	result := c.Check(context.Background(), target)

	if result.Status != checker.StatusUp {
		t.Errorf("Status = %q, want %q (error: %s)", result.Status, checker.StatusUp, result.Error)
	}
	if result.HTTPStatusCode != http.StatusOK {
		t.Errorf("HTTPStatusCode = %d, want 200", result.HTTPStatusCode)
	}
	if result.LatencyMS < 0 {
		t.Errorf("LatencyMS = %d, want >= 0", result.LatencyMS)
	}
	if result.TLSDaysRemaining != nil {
		t.Errorf("TLSDaysRemaining = %v, want nil for a plain HTTP server", *result.TLSDaysRemaining)
	}
}

func TestCheckDownOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := checker.New(2 * time.Second)
	target := config.Target{Name: "down-test", URL: srv.URL, TimeoutSeconds: 2}

	result := c.Check(context.Background(), target)

	if result.Status != checker.StatusDown {
		t.Errorf("Status = %q, want %q", result.Status, checker.StatusDown)
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error for a 500 response")
	}
}

func TestCheckDownOnUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately: the URL is now unreachable

	c := checker.New(1 * time.Second)
	target := config.Target{Name: "unreachable-test", URL: srv.URL, TimeoutSeconds: 1}

	result := c.Check(context.Background(), target)

	if result.Status != checker.StatusDown {
		t.Errorf("Status = %q, want %q", result.Status, checker.StatusDown)
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error for an unreachable target")
	}
}

func TestCheckReportsTLSDaysRemaining(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// srv.Client() already trusts the test server's self-signed
	// certificate, which lets this test exercise the real TLS-expiry code
	// path without weakening the Checker's default certificate validation.
	c := checker.NewWithClient(srv.Client())
	target := config.Target{Name: "tls-test", URL: srv.URL, TimeoutSeconds: 2}

	result := c.Check(context.Background(), target)

	if result.Status != checker.StatusUp {
		t.Fatalf("Status = %q, want %q (error: %s)", result.Status, checker.StatusUp, result.Error)
	}
	if result.TLSDaysRemaining == nil {
		t.Fatal("expected TLSDaysRemaining to be set for an HTTPS target")
	}
	if *result.TLSDaysRemaining <= 0 {
		t.Errorf("TLSDaysRemaining = %d, want > 0 for a freshly minted test certificate", *result.TLSDaysRemaining)
	}
}
