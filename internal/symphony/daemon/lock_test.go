package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
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

func TestDefaultStateDirUsesOSTempDir(t *testing.T) {
	expected := filepath.Join(os.TempDir(), "accountsdot-symphony")
	if DefaultStateDir != expected {
		t.Fatalf("expected default state dir %q, got %q", expected, DefaultStateDir)
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

func TestStaleDaemonLockKeepsFreshEPERMPIDLock(t *testing.T) {
	restoreDaemonPIDProbe(t, func(pid int) error {
		return syscall.EPERM
	})
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("active-daemon\n12345\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if stale || reason != "" {
		t.Fatalf("expected fresh eperm lock to remain active, got stale=%v reason=%q", stale, reason)
	}
}

func TestStaleDaemonLockRecoversOldEPERMPIDLock(t *testing.T) {
	restoreDaemonPIDProbe(t, func(pid int) error {
		return syscall.EPERM
	})
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("old-daemon\n12345\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	old := time.Now().Add(-daemonLockMaxAge - time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if !stale || reason == "" {
		t.Fatalf("expected old eperm lock to be stale, got stale=%v reason=%q", stale, reason)
	}
}

func TestStaleDaemonLockRecoversInactivePID(t *testing.T) {
	restoreDaemonPIDProbe(t, func(pid int) error {
		return syscall.ESRCH
	})
	path := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(path, []byte("old-daemon\n12345\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if !stale || reason == "" {
		t.Fatalf("expected inactive pid lock to be stale, got stale=%v reason=%q", stale, reason)
	}
}

func TestTouchDaemonLockRefreshesLockAge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("create lock: %v", err)
	}
	defer file.Close()
	if err := os.WriteFile(path, []byte("active-daemon\n"+strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	old := time.Now().Add(-daemonLockMaxAge - time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age lock: %v", err)
	}
	if err := touchDaemonLock(file); err != nil {
		t.Fatalf("touch lock: %v", err)
	}
	stale, reason := staleDaemonLock(path)
	if stale || reason != "" {
		t.Fatalf("expected refreshed live-pid lock to remain active, got stale=%v reason=%q", stale, reason)
	}
}

func restoreDaemonPIDProbe(t *testing.T, probe func(int) error) {
	t.Helper()
	original := daemonPIDProbe
	daemonPIDProbe = probe
	t.Cleanup(func() {
		daemonPIDProbe = original
	})
}
