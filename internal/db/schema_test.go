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
		"create table if not exists external_request_log",
		"create unique index if not exists external_request_log_idempotency_key_unique",
		"create index if not exists external_request_log_job_outcome_idx",
		"create table if not exists event_outbox",
		"create index if not exists event_outbox_global_tick_idx",
		"create table if not exists system_controls",
		"create table if not exists workflow_runs",
		"job_family text not null default 'unclassified'",
		"deferred_from_run_id bigint references workflow_runs(id)",
		"overlap_state text not null default 'none'",
		"create index if not exists workflow_runs_scheduled_family_active_idx",
		"create index if not exists workflow_runs_scheduled_family_overlap_idx",
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
		"create table if not exists auth_site_scope_mappings",
		"source_type text not null check (source_type in ('group', 'attribute'))",
		"site_codes jsonb not null default '[]'::jsonb",
		"create index if not exists auth_site_scope_mappings_source_idx",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(snippet)) {
			t.Fatalf("schema.sql must contain %q", snippet)
		}
	}
}

// TestAuthSiteScopeMigrationContainsPersistentMappingTable locks the Phase 0
// production-auth database contract to an idempotent migration. The future
// admin UI can populate this table when Google groups or SAML attributes do not
// fully carry site scope, and reviewers can diff this test when schema fields
// change.
func TestAuthSiteScopeMigrationContainsPersistentMappingTable(t *testing.T) {
	root := projectRoot(t)
	migration, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "202605210001_auth_site_scope_mappings.sql"))
	if err != nil {
		t.Fatalf("failed reading auth site-scope migration: %v", err)
	}
	text := string(migration)

	requiredSnippets := []string{
		"create table if not exists auth_site_scope_mappings",
		"source_type text not null check (source_type in ('group', 'attribute'))",
		"source_value text not null",
		"attribute_values jsonb not null default '[]'::jsonb",
		"site_codes jsonb not null default '[]'::jsonb",
		"actor_id text not null",
		"reason text not null",
		"unique (source_type, source_value)",
		"create index if not exists auth_site_scope_mappings_source_idx",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(snippet)) {
			t.Fatalf("auth site-scope migration must contain %q", snippet)
		}
	}
}

// TestOverlapMigrationContainsExistingDatabaseChanges exercises the migration
// script that keeps already-created dev and staging workflow_runs tables aligned
// with schema.sql. It checks for idempotent column and partial-index DDL so
// P0-0B-003 can be applied without rebuilding the database.
func TestOverlapMigrationContainsExistingDatabaseChanges(t *testing.T) {
	root := projectRoot(t)
	migration, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "202605200001_workflow_run_overlap.sql"))
	if err != nil {
		t.Fatalf("failed reading workflow overlap migration: %v", err)
	}
	text := string(migration)

	requiredSnippets := []string{
		"alter table if exists workflow_runs",
		"add column if not exists job_family text not null default 'unclassified'",
		"add column if not exists scheduled_for timestamptz",
		"add column if not exists deferred_from_run_id bigint references workflow_runs(id) on delete set null",
		"add column if not exists overlap_state text not null default 'none'",
		"add column if not exists overlap_count integer not null default 0",
		"create index if not exists workflow_runs_scheduled_family_active_idx",
		"where trigger_type = 'scheduled'",
		"and status in ('planned', 'running', 'recovering', 'waiting_manual')",
		"create index if not exists workflow_runs_scheduled_family_overlap_idx",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(snippet)) {
			t.Fatalf("workflow overlap migration must contain %q", snippet)
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
