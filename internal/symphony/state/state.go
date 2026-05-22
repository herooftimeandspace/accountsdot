package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ControllerFilename = "controller.json"
	StatusFilename     = "status.json"
	StatusMarkdown     = "status.md"
	RunsFilename       = "runs.jsonl"
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

// ReadSnapshot loads the daemon status file. Missing status is reported as a
// normal error so CLI callers can distinguish inactive Symphony from malformed
// state.
func ReadSnapshot(dir string) (Snapshot, error) {
	var snapshot Snapshot
	data, err := os.ReadFile(filepath.Join(dir, StatusFilename))
	if err != nil {
		return snapshot, err
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return snapshot, err
	}
	return snapshot, nil
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
