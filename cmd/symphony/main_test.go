package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
)

func TestIntFromLegacyReadsNestedFloat(t *testing.T) {
	value := map[string]any{"dispatcher": map[string]any{"max_concurrent_runs": 6.0}}
	if got := symphony.IntFromLegacy(value, "dispatcher", "max_concurrent_runs"); got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}

func TestIntFromLegacyMissingPathReturnsZero(t *testing.T) {
	if got := symphony.IntFromLegacy(map[string]any{}, "missing"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestGoCacheEnvSetsRepoLocalDefaults(t *testing.T) {
	env := goCacheEnv([]string{"PATH=/bin"}, "/repo")
	if !containsEnv(env, "GOCACHE=/repo/.gocache") {
		t.Fatalf("expected repo-local GOCACHE in %#v", env)
	}
	if !containsEnv(env, "GOMODCACHE=/repo/.gomodcache") {
		t.Fatalf("expected repo-local GOMODCACHE in %#v", env)
	}
}

func TestHasEnvKeyDetectsExistingCache(t *testing.T) {
	env := goCacheEnv([]string{"GOCACHE=/custom", "GOMODCACHE=/mods"}, "/repo")
	if !containsEnv(env, "GOCACHE=/custom") || !containsEnv(env, "GOMODCACHE=/mods") {
		t.Fatalf("expected existing cache env to be preserved: %#v", env)
	}
	if containsEnv(env, "GOCACHE=/repo/.gocache") {
		t.Fatalf("did not expect default GOCACHE when custom value exists: %#v", env)
	}
}

func TestRunControlQueuesCommand(t *testing.T) {
	dir := t.TempDir()
	if err := runControl([]string{"--state-dir", dir, "set-concurrency", "3"}); err != nil {
		t.Fatalf("runControl returned error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "control", "*.json"))
	if err != nil {
		t.Fatalf("glob command files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one command file, got %#v", matches)
	}
}

func TestRunControlAcceptsActionLocalConcurrencyFlag(t *testing.T) {
	dir := t.TempDir()
	if err := runControl([]string{"--state-dir", dir, "set-concurrency", "--concurrency", "4"}); err != nil {
		t.Fatalf("runControl returned error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "control", "*.json"))
	if err != nil {
		t.Fatalf("glob command files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one command file, got %#v", matches)
	}
}

func TestRunControlWorksOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
	if err := run(context.Background(), []string{"control", "--state-dir", dir, "cancel-worker", "worker-1"}); err != nil {
		t.Fatalf("control outside repo returned error: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "control", "*.json"))
	if err != nil {
		t.Fatalf("glob command files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one command file, got %#v", matches)
	}
}

func containsEnv(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
