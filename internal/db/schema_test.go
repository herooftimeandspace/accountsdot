package db_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"create table if not exists event_outbox",
		"create table if not exists system_controls",
		"create table if not exists workflow_runs",
		"create table if not exists import_batches",
		"create table if not exists approval_requests",
		"create table if not exists provider_circuit_breakers",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(snippet)) {
			t.Fatalf("schema.sql must contain %q", snippet)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}
