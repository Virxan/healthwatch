package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"healthwatch/backend/db"
)

// Checker performs a single HTTP health check against a URL, measuring
// latency and, for HTTPS targets, the remaining lifetime of the
// server's TLS certificate.
type Checker struct {
	client *http.Client
}

// NewChecker creates a Checker. defaultTimeout is a fallback the
// underlying HTTP client enforces in addition to any per-check timeout
// passed via ctx.
func NewChecker(defaultTimeout time.Duration) *Checker {
	return &Checker{client: &http.Client{Timeout: defaultTimeout}}
}

// NewCheckerWithClient creates a Checker backed by a caller-supplied
// HTTP client - mainly useful for tests that need a client configured
// with a custom certificate trust store (e.g. to talk to an
// httptest.NewTLSServer). Production code should use NewChecker
// instead, so targets are always verified against the system trust
// store.
func NewCheckerWithClient(client *http.Client) *Checker {
	return &Checker{client: client}
}

// Check performs a single HTTP GET against targetURL and reports the
// outcome. ctx controls cancellation/timeout.
func (c *Checker) Check(ctx context.Context, targetURL string) db.CheckResult {
	result := db.CheckResult{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		result.Status = "down"
		result.Error = fmt.Sprintf("building request: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "healthwatch/0.1")

	start := time.Now()
	resp, err := c.client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	// Drain the body so the connection can be reused, but don't keep the
	// content - Healthwatch only cares about reachability, not payloads.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	result.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
	} else {
		result.Status = "down"
		result.Error = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
	}

	result.TLSDaysRemaining = tlsDaysRemaining(resp.TLS)

	return result
}

// tlsDaysRemaining returns the number of whole days remaining before the
// leaf certificate's expiry, or nil if the connection wasn't TLS.
func tlsDaysRemaining(state *tls.ConnectionState) *int {
	if state == nil || len(state.PeerCertificates) == 0 {
		return nil
	}
	leaf := state.PeerCertificates[0]
	days := int(time.Until(leaf.NotAfter).Hours() / 24)
	return &days
}
