package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"healthwatch/internal/config"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeTempConfig(t, `
targets:
  - name: example
    url: https://example.com
    interval_seconds: 15
    timeout_seconds: 3
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(cfg.Targets))
	}

	got := cfg.Targets[0]
	if got.Name != "example" || got.URL != "https://example.com" {
		t.Errorf("unexpected target: %+v", got)
	}
	if got.Interval() != 15*time.Second {
		t.Errorf("Interval() = %v, want 15s", got.Interval())
	}
	if got.Timeout() != 3*time.Second {
		t.Errorf("Timeout() = %v, want 3s", got.Timeout())
	}
}

func TestDefaults(t *testing.T) {
	target := config.Target{Name: "no-tuning", URL: "https://example.com"}

	if target.Interval() != 30*time.Second {
		t.Errorf("default Interval() = %v, want 30s", target.Interval())
	}
	if target.Timeout() != 5*time.Second {
		t.Errorf("default Timeout() = %v, want 5s", target.Timeout())
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := config.Load("/nonexistent/targets.yaml"); err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}

func TestValidateMissingName(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{{URL: "https://example.com"}}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for a missing name, got nil")
	}
}

func TestValidateMissingURL(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{{Name: "example"}}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for a missing url, got nil")
	}
}

func TestValidateDuplicateName(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{Name: "dup", URL: "https://a.example.com"},
		{Name: "dup", URL: "https://b.example.com"},
	}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for a duplicate name, got nil")
	}
}

func TestValidateNoTargets(t *testing.T) {
	cfg := config.Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for an empty target list, got nil")
	}
}
