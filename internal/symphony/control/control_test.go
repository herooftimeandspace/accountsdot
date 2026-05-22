package control_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
)

func TestNewValidatesControlCommands(t *testing.T) {
	if _, err := control.New("cancel", "", 0); err == nil {
		t.Fatal("expected cancel without worker id to fail")
	}
	if _, err := control.New("set-concurrency", "", 0); err == nil {
		t.Fatal("expected set-concurrency without positive count to fail")
	}
	if command, err := control.New("pause", "", 0); err != nil || command.Action != "pause" {
		t.Fatalf("expected pause command, got %#v err=%v", command, err)
	}
	if command, err := control.New("cancel-worker", "worker-1", 0); err != nil || command.Action != "cancel" {
		t.Fatalf("expected cancel-worker alias to queue cancel command, got %#v err=%v", command, err)
	}
}

func TestWriteAndReadPendingCommand(t *testing.T) {
	dir := t.TempDir()
	command, err := control.New("set-concurrency", "", 2)
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	if _, err := control.WriteCommand(dir, command); err != nil {
		t.Fatalf("write command: %v", err)
	}
	commands, err := control.ReadPending(dir)
	if err != nil {
		t.Fatalf("read command: %v", err)
	}
	if len(commands) != 1 || commands[0].Concurrency != 2 {
		t.Fatalf("expected one pending command, got %#v", commands)
	}
}
