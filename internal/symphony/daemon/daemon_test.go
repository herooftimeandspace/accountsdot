package daemon_test

import (
	"context"
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/daemon"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
)

func TestDaemonAppliesPauseControlWithoutDispatching(t *testing.T) {
	dir := t.TempDir()
	command, err := control.New("pause", "", 0)
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	if _, err := control.WriteCommand(dir, command); err != nil {
		t.Fatalf("write command: %v", err)
	}
	snapshot, err := daemon.Run(context.Background(), daemon.Options{
		RepoRoot:     t.TempDir(),
		StateDir:     dir,
		Interval:     time.Millisecond,
		MaxWorkers:   3,
		MaxTicks:     1,
		InitialState: "running",
	})
	if err != nil {
		t.Fatalf("daemon run: %v", err)
	}
	if snapshot.Controller.Lifecycle != "stopped" || !snapshot.Controller.ShutdownRequested {
		t.Fatalf("expected daemon to stop after max tick, got %#v", snapshot.Controller)
	}
	loaded, err := state.ReadSnapshot(dir)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if loaded.Controller.LastStatus != "" {
		t.Fatalf("paused daemon should not run a sync tick, got %#v", loaded.Controller)
	}
}

func TestDaemonAppliesSetConcurrencyWhilePaused(t *testing.T) {
	dir := t.TempDir()
	command, err := control.New("set-concurrency", "", 4)
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	if _, err := control.WriteCommand(dir, command); err != nil {
		t.Fatalf("write command: %v", err)
	}
	snapshot, err := daemon.Run(context.Background(), daemon.Options{
		RepoRoot:     t.TempDir(),
		StateDir:     dir,
		Interval:     time.Millisecond,
		MaxWorkers:   6,
		MaxTicks:     1,
		InitialState: "paused",
	})
	if err != nil {
		t.Fatalf("daemon run: %v", err)
	}
	if snapshot.Controller.EffectiveWorkers != 4 {
		t.Fatalf("expected concurrency control to apply, got %#v", snapshot.Controller)
	}
}

func TestDaemonDryRunDoesNotConsumeControlCommands(t *testing.T) {
	dir := t.TempDir()
	command, err := control.New("pause", "", 0)
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	if _, err := control.WriteCommand(dir, command); err != nil {
		t.Fatalf("write command: %v", err)
	}
	snapshot, err := daemon.Run(context.Background(), daemon.Options{
		RepoRoot:     t.TempDir(),
		StateDir:     dir,
		Interval:     time.Millisecond,
		MaxTicks:     1,
		DryRun:       true,
		InitialState: "paused",
	})
	if err != nil {
		t.Fatalf("daemon dry-run: %v", err)
	}
	if snapshot.Controller.DryRun != true {
		t.Fatalf("expected dry-run snapshot, got %#v", snapshot.Controller)
	}
	commands, err := control.ReadPending(dir)
	if err != nil {
		t.Fatalf("read pending commands: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected dry-run to leave queued control command untouched, got %#v", commands)
	}
}

func TestDaemonRejectsSecondActiveLock(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := daemon.Run(ctx, daemon.Options{
			RepoRoot:     t.TempDir(),
			StateDir:     dir,
			Interval:     time.Hour,
			MaxWorkers:   1,
			InitialState: "paused",
		})
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	_, err := daemon.Run(context.Background(), daemon.Options{
		RepoRoot:     t.TempDir(),
		StateDir:     dir,
		Interval:     time.Millisecond,
		MaxTicks:     1,
		InitialState: "paused",
	})
	if err == nil {
		t.Fatal("expected second daemon to fail lock acquisition")
	}
	cancel()
	<-done
}
