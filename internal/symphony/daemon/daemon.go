package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
)

var DefaultStateDir = filepath.Join(os.TempDir(), "accountsdot-symphony")

const daemonLockMaxAge = 30 * time.Minute
const daemonLockHeartbeatInterval = daemonLockMaxAge / 3

var daemonPIDProbe = signalDaemonPID

// Options configures the local Symphony daemon. The daemon is intentionally
// single-machine and file-state based so a developer can stop, inspect, and
// recover it before freeing resources or rebooting.
type Options struct {
	RepoRoot     string
	StateDir     string
	Phase        string
	PhaseBranch  string
	Interval     time.Duration
	MaxWorkers   int
	MaxTicks     int
	DryRun       bool
	JSON         bool
	InitialState string
}

// Run starts the local daemon loop. It reuses the core sync tick, applies local
// control commands between ticks, and persists status when not in dry-run mode.
func Run(ctx context.Context, options Options) (state.Snapshot, error) {
	options = normalizeOptions(options)
	daemonID := uuid.NewString()
	controller := state.ControllerState{
		DaemonID:            daemonID,
		PID:                 os.Getpid(),
		RepoRoot:            options.RepoRoot,
		StateDir:            options.StateDir,
		Phase:               options.Phase,
		PhaseBranch:         options.PhaseBranch,
		TickIntervalSeconds: int(options.Interval.Seconds()),
		MaxWorkers:          options.MaxWorkers,
		EffectiveWorkers:    options.MaxWorkers,
		Lifecycle:           lifecycleOrDefault(options.InitialState, "running"),
		DryRun:              options.DryRun,
		UpdatedAt:           time.Now().UTC(),
	}
	snapshot := state.Snapshot{Controller: controller}
	if err := symphony.ValidateSyncOptions(symphony.SyncOptions{
		DryRun:      options.DryRun,
		PhaseID:     options.Phase,
		PhaseBranch: options.PhaseBranch,
		MaxRuns:     options.MaxWorkers,
	}); err != nil {
		return snapshot, err
	}
	var lock *os.File
	var err error
	if !options.DryRun {
		lock, err = acquireLock(options.StateDir, daemonID)
		if err != nil {
			return snapshot, err
		}
		defer releaseLock(lock, options.StateDir)
		if err := touchDaemonLock(lock); err != nil {
			return snapshot, err
		}
		stopLockHeartbeat := startDaemonLockHeartbeat(lock, daemonLockHeartbeatInterval)
		defer stopLockHeartbeat()
		if err := state.AppendEvent(options.StateDir, state.Event{Kind: "daemon.started", Message: "local Symphony daemon started"}); err != nil {
			return snapshot, err
		}
		// Publish the live controller immediately after lock acquisition so
		// status and TUI clients do not inherit a stopped snapshot from a prior
		// daemon while the first sync tick is still running or failing.
		if err := state.WriteSnapshot(options.StateDir, snapshot); err != nil {
			return snapshot, err
		}
	}

	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()

	for tick := 0; ; tick++ {
		if !options.DryRun {
			if err := applyControl(options.StateDir, &snapshot.Controller); err != nil {
				_ = state.AppendEvent(options.StateDir, state.Event{Kind: "control.error", Message: err.Error()})
			}
		}
		switch snapshot.Controller.Lifecycle {
		case "running":
			result, err := symphony.RunSyncTick(ctx, options.RepoRoot, symphony.SyncOptions{
				DryRun:      options.DryRun,
				PhaseID:     options.Phase,
				PhaseBranch: options.PhaseBranch,
				MaxRuns:     snapshot.Controller.EffectiveWorkers,
			})
			now := time.Now().UTC()
			snapshot.Controller.LastTick = now
			snapshot.Controller.NextTick = now.Add(options.Interval)
			snapshot.Controller.UpdatedAt = now
			if err != nil {
				snapshot.Controller.LastStatus = "tick_failed"
				snapshot.Controller.Message = err.Error()
			} else {
				snapshot.Controller.LastStatus = result.TopLevelStatus
				snapshot.Controller.Message = ""
				snapshot.Workers = mergeWorkerObservations(snapshot.Workers, observedWorkers(result, now), now, options.Interval)
			}
		case "paused":
			now := time.Now().UTC()
			snapshot.Controller.UpdatedAt = now
			snapshot.Controller.NextTick = now.Add(options.Interval)
			snapshot.Controller.Message = "paused; no new workers will dispatch"
		case "draining", "stopping":
			now := time.Now().UTC()
			snapshot.Controller.UpdatedAt = now
			snapshot.Controller.NextTick = now.Add(options.Interval)
			snapshot.Controller.Message = snapshot.Controller.Lifecycle + "; no new workers will dispatch"
		default:
			now := time.Now().UTC()
			snapshot.Controller.UpdatedAt = now
			snapshot.Controller.NextTick = now.Add(options.Interval)
			snapshot.Controller.LastStatus = "invalid_lifecycle"
			snapshot.Controller.Message = fmt.Sprintf("unsupported daemon lifecycle %q", snapshot.Controller.Lifecycle)
		}
		if !options.DryRun {
			if err := touchDaemonLock(lock); err != nil {
				return snapshot, err
			}
			if err := state.WriteSnapshot(options.StateDir, snapshot); err != nil {
				return snapshot, err
			}
			_ = state.AppendEvent(options.StateDir, state.Event{Kind: "daemon.tick", Message: snapshot.Controller.LastStatus})
		}
		if options.MaxTicks > 0 && tick+1 >= options.MaxTicks {
			snapshot.Controller.Lifecycle = "stopped"
			snapshot.Controller.ShutdownRequested = true
			snapshot.Controller.UpdatedAt = time.Now().UTC()
			if !options.DryRun {
				_ = state.WriteSnapshot(options.StateDir, snapshot)
				_ = state.AppendEvent(options.StateDir, state.Event{Kind: "daemon.stopped", Message: "max tick count reached"})
			}
			return snapshot, nil
		}
		if snapshot.Controller.Lifecycle == "draining" || snapshot.Controller.Lifecycle == "stopping" {
			snapshot.Controller.Lifecycle = "stopped"
			snapshot.Controller.ShutdownRequested = true
			snapshot.Controller.UpdatedAt = time.Now().UTC()
			if !options.DryRun {
				_ = state.WriteSnapshot(options.StateDir, snapshot)
				_ = state.AppendEvent(options.StateDir, state.Event{Kind: "daemon.stopped", Message: "operator stop requested"})
			}
			return snapshot, nil
		}
		select {
		case <-ctx.Done():
			snapshot.Controller.Lifecycle = "draining"
			snapshot.Controller.ShutdownRequested = true
			snapshot.Controller.Message = "signal received; daemon is draining"
			snapshot.Controller.UpdatedAt = time.Now().UTC()
			if !options.DryRun {
				_ = state.WriteSnapshot(options.StateDir, snapshot)
				_ = state.AppendEvent(options.StateDir, state.Event{Kind: "daemon.signal", Message: "shutdown signal received"})
			}
			return snapshot, nil
		case <-ticker.C:
		}
	}
}

func normalizeOptions(options Options) Options {
	if options.StateDir == "" {
		options.StateDir = DefaultStateDir
	}
	if options.Interval <= 0 {
		options.Interval = 5 * time.Minute
	}
	if options.MaxWorkers <= 0 {
		options.MaxWorkers = 6
	}
	if options.InitialState == "" {
		options.InitialState = "running"
	}
	return options
}

func lifecycleOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func observedWorkers(result symphony.TickResult, now time.Time) []state.WorkerState {
	workers := make([]state.WorkerState, 0, len(result.RunnableWork))
	for index, item := range result.RunnableWork {
		workers = append(workers, state.WorkerState{
			ID:          fmt.Sprintf("worker-%d", index+1),
			WorkItemID:  item.ID,
			Kind:        item.Kind,
			Number:      item.Number,
			Title:       item.Title,
			Branch:      item.Branch,
			Workspace:   item.Workspace,
			Status:      string(item.State),
			StartedAt:   now,
			LatestEvent: item.Reason,
			LogPath:     item.LogPath,
		})
	}
	return workers
}

func mergeWorkerObservations(existing []state.WorkerState, observed []state.WorkerState, now time.Time, staleAfter time.Duration) []state.WorkerState {
	if staleAfter <= 0 {
		staleAfter = 5 * time.Minute
	}
	if len(observed) == 0 {
		return freshWorkers(existing, now, staleAfter)
	}
	merged := freshWorkers(existing, now, staleAfter)
	indexByWorkItem := map[string]int{}
	for index, worker := range merged {
		if worker.WorkItemID != "" {
			indexByWorkItem[worker.WorkItemID] = index
		}
	}
	for _, worker := range observed {
		if index, ok := indexByWorkItem[worker.WorkItemID]; ok {
			if !merged[index].StartedAt.IsZero() {
				worker.StartedAt = merged[index].StartedAt
			}
			merged[index] = worker
			continue
		}
		merged = append(merged, worker)
	}
	return merged
}

func freshWorkers(workers []state.WorkerState, now time.Time, staleAfter time.Duration) []state.WorkerState {
	fresh := make([]state.WorkerState, 0, len(workers))
	for _, worker := range workers {
		if worker.StartedAt.IsZero() || now.Sub(worker.StartedAt) <= staleAfter {
			fresh = append(fresh, worker)
		}
	}
	return fresh
}

func applyControl(stateDir string, controller *state.ControllerState) error {
	commands, err := control.ReadPending(stateDir)
	if err != nil {
		return err
	}
	for _, command := range commands {
		result := control.Result{ID: command.ID, Action: command.Action, Status: "applied", AppliedAt: time.Now().UTC()}
		switch command.Action {
		case "pause":
			controller.Lifecycle = "paused"
			result.Message = "daemon paused"
		case "resume":
			controller.Lifecycle = "running"
			result.Message = "daemon resumed"
		case "drain":
			controller.Lifecycle = "draining"
			controller.ShutdownRequested = true
			result.Message = "daemon draining"
		case "stop":
			controller.Lifecycle = "stopping"
			controller.ShutdownRequested = true
			result.Message = "daemon stopping"
		case "cancel":
			result.Message = "worker cancellation recorded; active process termination is handled by the worker runner"
		case "set-concurrency":
			controller.EffectiveWorkers = command.Concurrency
			result.Message = fmt.Sprintf("worker concurrency set to %d", command.Concurrency)
		default:
			result.Status = "rejected"
			result.Message = "unsupported command"
		}
		if err := control.MarkApplied(stateDir, command, result); err != nil {
			return err
		}
	}
	return nil
}

func acquireLock(stateDir string, daemonID string) (*os.File, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(stateDir, "daemon.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			if stale, reason := staleDaemonLock(path); stale {
				_ = os.Remove(path)
				file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
				if err == nil {
					_ = state.AppendEvent(stateDir, state.Event{Kind: "daemon.lock_recovered", Message: reason})
				}
			}
			if err != nil {
				return nil, fmt.Errorf("symphony daemon lock already exists at %s", path)
			}
		} else {
			return nil, err
		}
	}
	if _, err := file.WriteString(fmt.Sprintf("%s\n%d\n", daemonID, os.Getpid())); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return file, nil
}

// staleDaemonLock decides whether acquireLock may remove an existing daemon
// lock before starting a replacement daemon. It uses the lock mtime as the
// daemon heartbeat: fresh locks protect the singleton guarantee, while old
// locks can be recovered even when the recorded PID now belongs to another
// user and rejects signal probes with EPERM.
func staleDaemonLock(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, ""
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		return false, ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return true, "removed malformed daemon lock without pid metadata"
	}
	pid, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil || pid <= 0 {
		return true, "removed malformed daemon lock with invalid pid metadata"
	}
	lockAge := time.Since(info.ModTime())
	if err := daemonPIDProbe(pid); err != nil {
		if errors.Is(err, syscall.EPERM) {
			if lockAge > daemonLockMaxAge {
				return true, fmt.Sprintf("removed stale daemon lock for unrefreshable pid %d", pid)
			}
			return false, ""
		}
		return true, fmt.Sprintf("removed stale daemon lock for inactive pid %d", pid)
	}
	if lockAge > daemonLockMaxAge {
		return true, fmt.Sprintf("removed stale daemon lock for unrefreshed live pid %d", pid)
	}
	return false, ""
}

// signalDaemonPID is the production PID liveness probe used by staleDaemonLock.
// It sends signal 0, which checks whether the process exists without mutating
// it; syscall.EPERM means a process exists but the current user cannot signal
// it, so staleDaemonLock must fall back to lock heartbeat age before removal.
func signalDaemonPID(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.Signal(0))
}

// startDaemonLockHeartbeat refreshes the singleton lock while Run is inside a
// long sync tick. staleDaemonLock may still recover a live PID when the lock has
// not been refreshed for daemonLockMaxAge, but an actively running daemon should
// keep this heartbeat moving so a second daemon never deletes the active lock
// solely because one tick exceeded the stale-lock threshold.
func startDaemonLockHeartbeat(file *os.File, interval time.Duration) func() {
	if file == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = daemonLockHeartbeatInterval
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = touchDaemonLock(file)
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
	}
}

func touchDaemonLock(file *os.File) error {
	if file == nil {
		return nil
	}
	now := time.Now()
	return os.Chtimes(file.Name(), now, now)
}

func releaseLock(file *os.File, stateDir string) {
	_ = file.Close()
	_ = os.Remove(filepath.Join(stateDir, "daemon.lock"))
}
