package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/daemon"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
)

type tickMsg time.Time

// Model is the Bubble Tea read model for Symphony's local daemon. It treats the
// daemon's status files as truth and sends operator actions through the same
// control command files used by the CLI.
type Model struct {
	StateDir string
	Snapshot state.Snapshot
	Message  string
	Err      error
}

// NewModel loads initial daemon state for tests and the interactive program.
func NewModel(stateDir string) Model {
	if stateDir == "" {
		stateDir = daemon.DefaultStateDir
	}
	snapshot, err := state.ReadSnapshot(stateDir)
	return Model{StateDir: stateDir, Snapshot: snapshot, Err: err}
}

// Run starts the interactive terminal monitor.
func Run(stateDir string) error {
	_, err := tea.NewProgram(NewModel(stateDir)).Run()
	return err
}

func (model Model) Init() tea.Cmd {
	return tick()
}

func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return model, tea.Quit
		case "p":
			model.Message = model.send("pause", "", 0)
		case "r":
			model.Message = model.send("resume", "", 0)
		case "d":
			model.Message = model.send("drain", "", 0)
		case "s":
			model.Message = model.send("stop", "", 0)
		case "+", "=":
			next := model.Snapshot.Controller.EffectiveWorkers + 1
			if next <= 0 {
				next = 1
			}
			model.Message = model.send("set-concurrency", "", next)
		case "-":
			next := model.Snapshot.Controller.EffectiveWorkers - 1
			if next < 1 {
				next = 1
			}
			model.Message = model.send("set-concurrency", "", next)
		}
	case tickMsg:
		snapshot, err := state.ReadSnapshot(model.StateDir)
		model.Snapshot = snapshot
		model.Err = err
		return model, tick()
	}
	return model, nil
}

func (model Model) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Symphony")
	if model.Err != nil {
		return fmt.Sprintf("%s\n\nNo daemon status at %s\n%s\n\nq quit\n", title, model.StateDir, model.Err)
	}
	controller := model.Snapshot.Controller
	lines := []string{
		title,
		"",
		fmt.Sprintf("Lifecycle: %s", controller.Lifecycle),
		fmt.Sprintf("Top-level status: %s", controller.LastStatus),
		fmt.Sprintf("PID: %d", controller.PID),
		fmt.Sprintf("Phase: %s", valueOrDash(controller.Phase)),
		fmt.Sprintf("Phase branch: %s", valueOrDash(controller.PhaseBranch)),
		fmt.Sprintf("Workers: %d/%d", len(model.Snapshot.Workers), controller.EffectiveWorkers),
		fmt.Sprintf("Last tick: %s", formatTime(controller.LastTick)),
		fmt.Sprintf("Next tick: %s", formatTime(controller.NextTick)),
	}
	if model.Message != "" {
		lines = append(lines, "", model.Message)
	}
	lines = append(lines, "", "Workers")
	if len(model.Snapshot.Workers) == 0 {
		lines = append(lines, "  none")
	}
	for _, worker := range model.Snapshot.Workers {
		lines = append(lines, fmt.Sprintf("  %s %s #%d %s %s", worker.ID, worker.Kind, worker.Number, worker.Status, worker.Branch))
	}
	lines = append(lines, "", "Controls: p pause | r resume | d drain | s stop | +/- concurrency | q quit")
	return strings.Join(lines, "\n") + "\n"
}

func (model Model) send(action string, target string, concurrency int) string {
	command, err := control.New(action, target, concurrency)
	if err != nil {
		return err.Error()
	}
	if _, err := control.WriteCommand(model.StateDir, command); err != nil {
		return err.Error()
	}
	return fmt.Sprintf("queued %s", action)
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func valueOrDash(value string) string {
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
