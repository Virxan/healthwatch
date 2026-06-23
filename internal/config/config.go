// Package config loads and validates the list of websites Healthwatch
// should monitor.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// Target is a single website to monitor.
type Target struct {
	Name            string `yaml:"name"`
	URL             string `yaml:"url"`
	IntervalSeconds int    `yaml:"interval_seconds"`
	TimeoutSeconds  int    `yaml:"timeout_seconds"`
}

// Interval returns the check interval as a time.Duration, applying a
// sensible default when unset.
func (t Target) Interval() time.Duration {
	if t.IntervalSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(t.IntervalSeconds) * time.Second
}

// Timeout returns the per-check HTTP timeout, applying a sensible default
// when unset.
func (t Target) Timeout() time.Duration {
	if t.TimeoutSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(t.TimeoutSeconds) * time.Second
}

// Config is the root configuration document.
type Config struct {
	Targets []Target `yaml:"targets"`
}

// Load reads and parses a YAML targets file from disk.
func Load(path string) (*Config, error) {
	// path is operator-supplied (CLI flag), not untrusted user input -
	// reading a configurable path is this function's whole purpose.
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is operator-controlled, not user input
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return &cfg, nil
}

// Validate checks that every target has the minimum required fields and
// that names are unique (the name doubles as the lookup key in the store
// and in the API).
func (c Config) Validate() error {
	if len(c.Targets) == 0 {
		return fmt.Errorf("no targets configured")
	}

	seen := make(map[string]bool, len(c.Targets))
	for i, t := range c.Targets {
		if t.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if t.URL == "" {
			return fmt.Errorf("target %q: url is required", t.Name)
		}
		if seen[t.Name] {
			return fmt.Errorf("target %q: duplicate name", t.Name)
		}
		seen[t.Name] = true
	}

	return nil
}
