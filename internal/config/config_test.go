package config_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
)

// TestLoadDefaults exercises and documents internal/config/config_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppPort != "8080" {
		t.Fatalf("expected default app port 8080, got %q", cfg.AppPort)
	}
	if cfg.ZoomSLGMaxMembers != 10 {
		t.Fatalf("expected default zoom SLG max members 10, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 100 {
		t.Fatalf("expected replay limit 100, got %d", cfg.ReplayEventLimit)
	}
}

// TestLoadOverridesFromEnvironment exercises and documents internal/config/config_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestLoadOverridesFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("ZOOM_SLG_MAX_MEMBERS", "12")
	t.Setenv("REPLAY_EVENT_LIMIT", "55")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppEnv != "test" {
		t.Fatalf("expected app env test, got %q", cfg.AppEnv)
	}
	if cfg.AppPort != "9090" {
		t.Fatalf("expected app port 9090, got %q", cfg.AppPort)
	}
	if cfg.ZoomSLGMaxMembers != 12 {
		t.Fatalf("expected zoom limit 12, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 55 {
		t.Fatalf("expected replay limit 55, got %d", cfg.ReplayEventLimit)
	}
}

// TestLoadFallsBackOnInvalidIntegerValues exercises and documents internal/config/config_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestLoadFallsBackOnInvalidIntegerValues(t *testing.T) {
	t.Setenv("ZOOM_SLG_MAX_MEMBERS", "bogus")
	t.Setenv("REPLAY_EVENT_LIMIT", "bogus")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ZoomSLGMaxMembers != 10 {
		t.Fatalf("expected zoom limit fallback 10, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 100 {
		t.Fatalf("expected replay limit fallback 100, got %d", cfg.ReplayEventLimit)
	}
}
