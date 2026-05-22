package main

import "testing"

func TestIntFromLegacyReadsNestedFloat(t *testing.T) {
	value := map[string]any{"dispatcher": map[string]any{"max_concurrent_runs": 6.0}}
	if got := intFromLegacy(value, "dispatcher", "max_concurrent_runs"); got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}

func TestIntFromLegacyMissingPathReturnsZero(t *testing.T) {
	if got := intFromLegacy(map[string]any{}, "missing"); got != 0 {
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

func containsEnv(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
