package tui_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
	symphonytui "github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/tui"
)

func TestModelQueuesPauseCommand(t *testing.T) {
	dir := t.TempDir()
	if err := state.WriteSnapshot(dir, state.Snapshot{Controller: state.ControllerState{
		PID:              123,
		StateDir:         dir,
		Lifecycle:        "running",
		EffectiveWorkers: 2,
		UpdatedAt:        time.Now().UTC(),
	}}); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	model := symphonytui.NewModel(dir)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	view := updated.(symphonytui.Model).View()
	if !strings.Contains(view, "queued pause") {
		t.Fatalf("expected queued pause message in view, got:\n%s", view)
	}
}

func TestModelRendersWorkerDiagnostics(t *testing.T) {
	started := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	model := symphonytui.Model{
		Now: started.Add(2*time.Minute + 3*time.Second),
		Snapshot: state.Snapshot{
			Controller: state.ControllerState{
				PID:              123,
				Lifecycle:        "running",
				LastStatus:       "dispatches_started",
				EffectiveWorkers: 6,
			},
			Workers: []state.WorkerState{{
				ID:          "worker-1",
				WorkItemID:  "issue-327",
				Kind:        "issue",
				Number:      327,
				Title:       "Improve Symphony TUI worker diagnostics",
				Branch:      "codex/issue-327-symphony-tui-diagnostics",
				Workspace:   "/private/tmp/accountsdot-symphony/repo327",
				PID:         4242,
				Status:      "runnable",
				StartedAt:   started,
				LatestEvent: "pushing PR branch",
				RetryCount:  2,
				LogPath:     "/private/tmp/accountsdot-symphony/repo327/logs/agent-stdout.log",
			}},
		},
		ProcessSamples: map[int]symphonytui.ProcessSample{
			4242: {CPUPercent: 12.4, Available: true},
		},
	}

	view := model.View()
	for _, expected := range []string{
		"issue #327",
		"elapsed=2m3s",
		"cpu=12.4%",
		"retry=2",
		"Improve Symphony TUI worker diagnostics",
		"pushing PR branch",
		"/private/tmp/accountsdot-symphony/repo327",
		"/private/tmp/accountsdot-symphony/repo327/logs/agent-stdout.log",
	} {
		if !strings.Contains(view, expected) {
			t.Fatalf("expected %q in view, got:\n%s", expected, view)
		}
	}
}

func TestModelRendersCPUUnavailableWithoutPID(t *testing.T) {
	started := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	model := symphonytui.Model{
		Now: started.Add(time.Minute),
		Snapshot: state.Snapshot{Workers: []state.WorkerState{{
			ID:        "worker-1",
			Kind:      "pull_request",
			Number:    323,
			Status:    "waiting_external_review",
			StartedAt: started,
		}}},
	}

	view := model.View()
	if !strings.Contains(view, "cpu=n/a") {
		t.Fatalf("expected unavailable CPU in view, got:\n%s", view)
	}
}
