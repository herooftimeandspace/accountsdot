package state

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWriteSnapshotReplacesStatusAtomically(t *testing.T) {
	dir := t.TempDir()
	stalePath := filepath.Join(dir, StatusFilename)
	if err := os.WriteFile(stalePath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("seed stale status: %v", err)
	}

	snapshot := Snapshot{
		Controller: ControllerState{
			DaemonID:            "daemon-1",
			PID:                 123,
			StateDir:            dir,
			TickIntervalSeconds: 30,
			MaxWorkers:          4,
			EffectiveWorkers:    2,
			Lifecycle:           "running",
			LastStatus:          "dispatches_available",
			UpdatedAt:           time.Now().UTC(),
		},
		Workers: []WorkerState{{
			ID:         "worker-1",
			WorkItemID: "issue-292",
			Kind:       "issue",
			Status:     "running",
			StartedAt:  time.Now().UTC(),
		}},
	}

	if err := WriteSnapshot(dir, snapshot); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	loaded, err := ReadSnapshot(dir)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if loaded.Controller.DaemonID != "daemon-1" || len(loaded.Workers) != 1 {
		t.Fatalf("unexpected snapshot readback: %#v", loaded)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".status.json.tmp-*"))
	if err != nil {
		t.Fatalf("glob temporary status files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temporary status files to be cleaned up, got %v", matches)
	}
	markdown, err := os.ReadFile(filepath.Join(dir, StatusMarkdown))
	if err != nil {
		t.Fatalf("read status markdown: %v", err)
	}
	if !strings.Contains(string(markdown), "dispatches_available") {
		t.Fatalf("expected markdown status to include top-level status, got %q", string(markdown))
	}
}

func TestReadSnapshotReconcilesStaleControllerWithLiveLock(t *testing.T) {
	dir := t.TempDir()
	stale := Snapshot{Controller: ControllerState{
		DaemonID:          "old-daemon",
		PID:               999999999,
		StateDir:          dir,
		Lifecycle:         "draining",
		LastStatus:        "tick_failed",
		ShutdownRequested: true,
		Phase:             "phase-0",
		PhaseBranch:       "phase-0-platform-foundation",
		UpdatedAt:         time.Now().Add(-time.Hour).UTC(),
	}}
	if err := WriteSnapshot(dir, stale); err != nil {
		t.Fatalf("write stale snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "daemon.lock"), []byte("live-daemon\n"+strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatalf("write live lock: %v", err)
	}

	loaded, err := ReadSnapshot(dir)
	if err != nil {
		t.Fatalf("read reconciled snapshot: %v", err)
	}
	if loaded.Controller.DaemonID != "live-daemon" || loaded.Controller.PID != os.Getpid() {
		t.Fatalf("expected live lock identity, got %#v", loaded.Controller)
	}
	if loaded.Controller.Lifecycle != "running" || loaded.Controller.ShutdownRequested {
		t.Fatalf("expected live daemon state, got %#v", loaded.Controller)
	}
	if loaded.Controller.LastStatus != "status_snapshot_stale" {
		t.Fatalf("expected stale projection status, got %#v", loaded.Controller)
	}
	if loaded.Controller.Phase != "phase-0" || loaded.Controller.PhaseBranch != "phase-0-platform-foundation" {
		t.Fatalf("expected existing phase metadata to be preserved, got %#v", loaded.Controller)
	}
}

func TestReadSnapshotCanReportLiveLockBeforeStatusFileExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "daemon.lock"), []byte("live-daemon\n"+strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatalf("write live lock: %v", err)
	}

	loaded, err := ReadSnapshot(dir)
	if err != nil {
		t.Fatalf("read lock-only snapshot: %v", err)
	}
	if loaded.Controller.DaemonID != "live-daemon" || loaded.Controller.Lifecycle != "running" {
		t.Fatalf("expected live lock snapshot, got %#v", loaded.Controller)
	}
	if loaded.Controller.LastStatus != "status_snapshot_pending" {
		t.Fatalf("expected pending projection status, got %#v", loaded.Controller)
	}
}
