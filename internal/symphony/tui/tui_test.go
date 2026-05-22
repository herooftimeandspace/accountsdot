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
