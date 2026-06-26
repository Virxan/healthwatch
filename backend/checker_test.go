package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckerUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewChecker(2 * time.Second)
	result := c.Check(context.Background(), srv.URL)

	if result.Status != "up" {
		t.Errorf("Status = %q, want %q (error: %s)", result.Status, "up", result.Error)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Errorf("HTTPStatus = %d, want 200", result.HTTPStatus)
	}
	if result.LatencyMS < 0 {
		t.Errorf("LatencyMS = %d, want >= 0", result.LatencyMS)
	}
	if result.TLSDaysRemaining != nil {
		t.Errorf("TLSDaysRemaining = %v, want nil for a plain HTTP server", *result.TLSDaysRemaining)
	}
}

func TestCheckerDownOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewChecker(2 * time.Second)
	result := c.Check(context.Background(), srv.URL)

	if result.Status != "down" {
		t.Errorf("Status = %q, want %q", result.Status, "down")
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error for a 500 response")
	}
}

func TestCheckerDownOnUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // close immediately: the URL is now unreachable

	c := NewChecker(1 * time.Second)
	result := c.Check(context.Background(), srv.URL)

	if result.Status != "down" {
		t.Errorf("Status = %q, want %q", result.Status, "down")
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error for an unreachable target")
	}
}

func TestCheckerReportsTLSDaysRemaining(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// srv.Client() already trusts the test server's self-signed
	// certificate, which lets this test exercise the real TLS-expiry
	// code path without weakening the Checker's default certificate
	// validation.
	c := NewCheckerWithClient(srv.Client())
	result := c.Check(context.Background(), srv.URL)

	if result.Status != "up" {
		t.Fatalf("Status = %q, want %q (error: %s)", result.Status, "up", result.Error)
	}
	if result.TLSDaysRemaining == nil {
		t.Fatal("expected TLSDaysRemaining to be set for an HTTPS target")
	}
	if *result.TLSDaysRemaining <= 0 {
		t.Errorf("TLSDaysRemaining = %d, want > 0 for a freshly minted test certificate", *result.TLSDaysRemaining)
	}
}

func TestTLSDaysRemainingNilForPlainHTTP(t *testing.T) {
	if got := tlsDaysRemaining(nil); got != nil {
		t.Errorf("tlsDaysRemaining(nil) = %v, want nil", *got)
	}
}
