package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ControllerFilename = "controller.json"
	StatusFilename     = "status.json"
	StatusMarkdown     = "status.md"
	RunsFilename       = "runs.jsonl"
	liveLockMaxAge     = 30 * time.Minute
)

// ControllerState is the daemon's durable local control-plane snapshot. It is
// intentionally local-machine scoped so operators can inspect and recover
// Symphony without relying on Codex automation state.
type ControllerState struct {
	DaemonID            string    `json:"daemon_id"`
	PID                 int       `json:"pid"`
	RepoRoot            string    `json:"repo_root"`
	StateDir            string    `json:"state_dir"`
	Phase               string    `json:"phase,omitempty"`
	PhaseBranch         string    `json:"phase_branch,omitempty"`
	TickIntervalSeconds int       `json:"tick_interval_seconds"`
	MaxWorkers          int       `json:"max_workers"`
	EffectiveWorkers    int       `json:"effective_workers"`
	Lifecycle           string    `json:"lifecycle"`
	DryRun              bool      `json:"dry_run"`
	LastTick            time.Time `json:"last_tick,omitempty"`
	NextTick            time.Time `json:"next_tick,omitempty"`
	LastStatus          string    `json:"last_status,omitempty"`
	ShutdownRequested   bool      `json:"shutdown_requested"`
	Message             string    `json:"message,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// WorkerState describes one local worker known to the daemon or recovered from
// the state directory.
type WorkerState struct {
	ID          string    `json:"id"`
	WorkItemID  string    `json:"work_item_id"`
	Kind        string    `json:"kind"`
	Number      int       `json:"number,omitempty"`
	Title       string    `json:"title,omitempty"`
	Branch      string    `json:"branch,omitempty"`
	Workspace   string    `json:"workspace,omitempty"`
	PID         int       `json:"pid,omitempty"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
	LatestEvent string    `json:"latest_event,omitempty"`
	RetryCount  int       `json:"retry_count"`
	LogPath     string    `json:"log_path,omitempty"`
}

// Snapshot is the complete read model consumed by status, control, and the TUI.
type Snapshot struct {
	Controller ControllerState `json:"controller"`
	Workers    []WorkerState   `json:"workers,omitempty"`
}

// Event is appended to runs.jsonl for both daemon lifecycle changes and worker
// events. The message must be sanitized by the caller before it reaches this
// file.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	RunID     string    `json:"run_id,omitempty"`
	Kind      string    `json:"kind"`
	Message   string    `json:"message"`
}

// WriteSnapshot persists machine-readable status plus a small Markdown summary
// for humans tailing the runtime directory.
func WriteSnapshot(dir string, snapshot Snapshot) error {
	if err := os.MkdirAll(filepath.Join(dir, "workers"), 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, ControllerFilename), snapshot.Controller); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, StatusFilename), snapshot); err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(dir, StatusMarkdown), []byte(renderMarkdown(snapshot)), 0o644)
}

// ReadSnapshot loads the daemon status file and reconciles it with the daemon
// lock when a live process is holding the singleton. Status and TUI callers use
// this read path while the daemon may be between startup and its first full
// tick, so a newer live lock must outrank an older stopped controller snapshot.
func ReadSnapshot(dir string) (Snapshot, error) {
	var snapshot Snapshot
	data, err := os.ReadFile(filepath.Join(dir, StatusFilename))
	if err != nil {
		if os.IsNotExist(err) {
			if live, ok := liveSnapshotFromLock(dir); ok {
				return live, nil
			}
		}
		return snapshot, err
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, err
	}
	return reconcileSnapshotWithLock(dir, snapshot), nil
}

func reconcileSnapshotWithLock(dir string, snapshot Snapshot) Snapshot {
	lock, ok := readLiveLock(dir)
	if !ok {
		return snapshot
	}
	if snapshot.Controller.DaemonID == lock.daemonID && snapshot.Controller.PID == lock.pid {
		return snapshot
	}
	if !snapshot.Controller.UpdatedAt.IsZero() && !lock.updatedAt.After(snapshot.Controller.UpdatedAt) {
		return snapshot
	}
	snapshot.Controller.DaemonID = lock.daemonID
	snapshot.Controller.PID = lock.pid
	snapshot.Controller.StateDir = dir
	snapshot.Controller.Lifecycle = "running"
	snapshot.Controller.ShutdownRequested = false
	snapshot.Controller.LastStatus = "status_snapshot_stale"
	snapshot.Controller.Message = "live daemon lock is newer than status files; waiting for next status write"
	snapshot.Controller.UpdatedAt = lock.updatedAt
	return snapshot
}

func liveSnapshotFromLock(dir string) (Snapshot, bool) {
	lock, ok := readLiveLock(dir)
	if !ok {
		return Snapshot{}, false
	}
	return Snapshot{Controller: ControllerState{
		DaemonID:          lock.daemonID,
		PID:               lock.pid,
		StateDir:          dir,
		Lifecycle:         "running",
		LastStatus:        "status_snapshot_pending",
		Message:           "live daemon lock exists but status files have not been written yet",
		UpdatedAt:         lock.updatedAt,
		ShutdownRequested: false,
	}}, true
}

type liveLock struct {
	daemonID  string
	pid       int
	updatedAt time.Time
}

func readLiveLock(dir string) (liveLock, bool) {
	path := filepath.Join(dir, "daemon.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return liveLock{}, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return liveLock{}, false
	}
	updatedAt := info.ModTime().UTC()
	if time.Since(updatedAt) > liveLockMaxAge {
		return liveLock{}, false
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return liveLock{}, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil || pid <= 0 {
		return liveLock{}, false
	}
	if err := probeLivePID(pid); err != nil && !os.IsPermission(err) {
		return liveLock{}, false
	}
	return liveLock{daemonID: strings.TrimSpace(lines[0]), pid: pid, updatedAt: updatedAt}, true
}

func probeLivePID(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.Signal(0))
}

// AppendEvent records a lifecycle or worker event without rewriting previous
// history.
func AppendEvent(dir string, event Event) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, RunsFilename), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(data, '\n'), 0o644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(perm); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func renderMarkdown(snapshot Snapshot) string {
	controller := snapshot.Controller
	return fmt.Sprintf(`# Symphony Status

- Lifecycle: %s
- Top-level status: %s
- PID: %d
- Phase: %s
- Phase branch: %s
- Workers: %d/%d
- Last tick: %s
- Next tick: %s
`,
		controller.Lifecycle,
		controller.LastStatus,
		controller.PID,
		emptyDash(controller.Phase),
		emptyDash(controller.PhaseBranch),
		len(snapshot.Workers),
		controller.EffectiveWorkers,
		formatTime(controller.LastTick),
		formatTime(controller.NextTick),
	)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(time.RFC3339)
}
