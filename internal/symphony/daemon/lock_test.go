package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestStaleDaemonLockRecoversMalformedLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("partial\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if !stale || reason == "" {
		t.Fatalf("expected malformed lock to be stale, got stale=%v reason=%q", stale, reason)
	}
}

func TestStaleDaemonLockRecoversOldLivePidLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("old-daemon\n"+strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	old := time.Now().Add(-daemonLockMaxAge - time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if !stale || reason == "" {
		t.Fatalf("expected old live-pid lock to be stale, got stale=%v reason=%q", stale, reason)
	}
}

func TestStaleDaemonLockKeepsFreshLivePidLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("active-daemon\n"+strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if stale || reason != "" {
		t.Fatalf("expected fresh live-pid lock to remain active, got stale=%v reason=%q", stale, reason)
	}
}
