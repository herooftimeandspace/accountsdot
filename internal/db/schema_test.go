package db_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchemaContainsCoreTablesAndConstraints exercises and documents internal/db/schema_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSchemaContainsCoreTablesAndConstraints(t *testing.T) {
	root := projectRoot(t)
	schema, err := os.ReadFile(filepath.Join(root, "internal", "db", "schema.sql"))
	if err != nil {
		t.Fatalf("failed reading schema.sql: %v", err)
	}
	text := string(schema)

	requiredSnippets := []string{
		"create table if not exists people",
		"create table if not exists known_identifiers",
		"create unique index if not exists known_identifiers_source_unique",
		"create sequence if not exists global_tick_seq",
		"create table if not exists jobs",
		"workflow_run_id bigint references workflow_runs(id)",
		"approval_required boolean not null default false",
		"create index if not exists jobs_claimable_global_tick_idx",
		"create index if not exists jobs_expired_lease_global_tick_idx",
		"create table if not exists event_outbox",
		"create table if not exists system_controls",
		"create table if not exists workflow_runs",
		"create table if not exists import_batches",
		"create table if not exists approval_requests",
		"create table if not exists provider_circuit_breakers",
		"create table if not exists user_sync_status",
		"primary key (user_type, user_id, school_year)",
		"people_uuid uuid references people(uuid)",
		"completion_summary text",
		"errors_warnings jsonb not null default '[]'::jsonb",
		"create table if not exists room_mapping_overrides",
		"create table if not exists feature_flags",
		"create table if not exists feature_flag_targets",
		"target_type text not null check (target_type in ('persona', 'site'))",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(snippet)) {
			t.Fatalf("schema.sql must contain %q", snippet)
		}
	}
}

// projectRoot resolves the repository root for schema text assertions. The
// schema test runs from internal/db, so this helper keeps the file read anchored
// to the checkout instead of depending on the caller's shell directory.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
