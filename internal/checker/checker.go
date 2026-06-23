// Package checker performs a single HTTP health check against a target,
// measuring latency and, for HTTPS targets, the remaining lifetime of the
// server's TLS certificate.
package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"healthwatch/internal/config"
)

// Status is the outcome of a single check.
type Status string

// Status values a check can report.
const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// Result is the outcome of checking one target once.
type Result struct {
	Target           string    `json:"target"`
	URL              string    `json:"url"`
	Status           Status    `json:"status"`
	HTTPStatusCode   int       `json:"http_status_code,omitempty"`
	LatencyMS        int64     `json:"latency_ms"`
	TLSDaysRemaining *int      `json:"tls_days_remaining,omitempty"`
	Error            string    `json:"error,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
}

// Checker performs HTTP health checks.
type Checker struct {
	client *http.Client
}

// New creates a Checker. defaultTimeout is a fallback the underlying HTTP
// client enforces in addition to any per-target timeout applied in Check.
func New(defaultTimeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewWithClient creates a Checker backed by a caller-supplied HTTP
// client. This exists mainly for tests that need a client configured
// with a custom certificate trust store (e.g. to talk to an
// httptest.NewTLSServer); production code should use New instead, so
// targets are always verified against the system trust store.
func NewWithClient(client *http.Client) *Checker {
	return &Checker{client: client}
}

// Check performs a single HTTP GET against target.URL and reports the
// outcome. ctx controls cancellation; the per-target timeout from the
// target configuration is additionally applied on top of it.
func (c *Checker) Check(ctx context.Context, target config.Target) Result {
	checkCtx, cancel := context.WithTimeout(ctx, target.Timeout())
	defer cancel()

	result := Result{
		Target:    target.Name,
		URL:       target.URL,
		CheckedAt: time.Now().UTC(),
	}

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("building request: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "healthwatch/0.1")

	start := time.Now()
	resp, err := c.client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = StatusDown
		result.Error = err.Error()
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	// Drain the body so the connection can be reused, but don't keep the
	// content - Healthwatch only cares about reachability, not payloads.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	result.HTTPStatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = StatusUp
	} else {
		result.Status = StatusDown
		result.Error = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
	}

	if days := tlsDaysRemaining(resp.TLS); days != nil {
		result.TLSDaysRemaining = days
	}

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
