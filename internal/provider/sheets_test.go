package provider_test

import (
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

func TestChecksumIsStable(t *testing.T) {
	rows := [][]string{
		{"Email", "First Name"},
		{"a@example.com", "Ada"},
	}
	left := provider.ChecksumRows(rows)
	right := provider.ChecksumRows(rows)
	if left == "" || left != right {
		t.Fatalf("expected stable checksum, got %q and %q", left, right)
	}
}

func TestBuildSentinelRow(t *testing.T) {
	row := provider.BuildSentinelRow(2, "abc123", 42)
	if row[0] != provider.SentinelMarker {
		t.Fatalf("expected sentinel marker %q, got %q", provider.SentinelMarker, row[0])
	}
	if row[1] != "2" || row[2] != "abc123" || row[3] != "42" {
		t.Fatalf("unexpected sentinel row: %#v", row)
	}
}

func TestSyncConfigCellForTab(t *testing.T) {
	tests := map[string]string{
		"Zoom_SLG":        "B2",
		"Zoom_Users":      "B3",
		"Zoom_CallQueues": "B4",
		"Zoom_CommonArea": "B5",
		"Zoom_AR":         "B6",
	}
	for tab, want := range tests {
		got, err := provider.SyncConfigCell(tab)
		if err != nil {
			t.Fatalf("SyncConfigCell(%q) returned error: %v", tab, err)
		}
		if got != want {
			t.Fatalf("SyncConfigCell(%q) = %q, want %q", tab, got, want)
		}
	}
}

func TestVisibleTabFormula(t *testing.T) {
	got, err := provider.VisibleTabFormula("Zoom_Users")
	if err != nil {
		t.Fatalf("VisibleTabFormula returned error: %v", err)
	}
	if !strings.Contains(got, `Sync_Config!B3`) {
		t.Fatalf("expected formula to reference Sync_Config!B3, got %q", got)
	}
	if !strings.Contains(got, "QUERY(INDIRECT(") {
		t.Fatalf("expected query/indirect formula, got %q", got)
	}
}

func TestUnknownSheetTabReturnsError(t *testing.T) {
	if _, err := provider.SyncConfigCell("bogus"); err == nil {
		t.Fatal("expected unknown tab to return an error")
	}
	if _, err := provider.VisibleTabFormula("bogus"); err == nil {
		t.Fatal("expected unknown tab formula request to return an error")
	}
}
