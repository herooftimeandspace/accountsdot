package tui

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/daemon"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
)

type tickMsg time.Time

type ProcessSample struct {
	CPUPercent float64
	Available  bool
	Reason     string
}

type ProcessSampler func(pid int) ProcessSample

// Model is the Bubble Tea read model for Symphony's local daemon. It treats the
// daemon's status files as truth and sends operator actions through the same
// control command files used by the CLI.
type Model struct {
	StateDir       string
	Snapshot       state.Snapshot
	Message        string
	Err            error
	Now            time.Time
	ProcessSamples map[int]ProcessSample
	Sampler        ProcessSampler
}

// NewModel loads initial daemon state for tests and the interactive program.
func NewModel(stateDir string) Model {
	if stateDir == "" {
		stateDir = daemon.DefaultStateDir
	}
	snapshot, err := state.ReadSnapshot(stateDir)
	now := time.Now().UTC()
	model := Model{StateDir: stateDir, Snapshot: snapshot, Err: err, Now: now, Sampler: sampleProcess}
	model.refreshProcessSamples()
	return model
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
		model.Now = time.Time(msg).UTC()
		model.refreshProcessSamples()
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
		fmt.Sprintf("Workers: %d/%d active", len(model.Snapshot.Workers), controller.EffectiveWorkers),
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
	for _, worker := range sortedWorkers(model.Snapshot.Workers) {
		lines = append(lines, model.renderWorker(worker)...)
	}
	lines = append(lines, "", "Controls: p pause | r resume | d drain | s stop | +/- concurrency | cancel via symphony control cancel <worker-id> | q quit")
	return strings.Join(lines, "\n") + "\n"
}

func (model Model) renderWorker(worker state.WorkerState) []string {
	ref := worker.Kind
	if worker.Number != 0 {
		ref = fmt.Sprintf("%s #%d", worker.Kind, worker.Number)
	}
	header := fmt.Sprintf("  %s  %s  %s  elapsed=%s  cpu=%s  retry=%d",
		valueOrDash(worker.ID),
		ref,
		valueOrDash(worker.Status),
		formatElapsed(model.Now, worker.StartedAt, worker.EndedAt),
		model.formatCPU(worker.PID),
		worker.RetryCount,
	)
	lines := []string{header}
	if worker.Title != "" {
		lines = append(lines, fmt.Sprintf("    title: %s", worker.Title))
	}
	if worker.LatestEvent != "" {
		lines = append(lines, fmt.Sprintf("    latest: %s", worker.LatestEvent))
	}
	lines = append(lines,
		fmt.Sprintf("    branch: %s", valueOrDash(worker.Branch)),
		fmt.Sprintf("    workspace: %s", valueOrDash(worker.Workspace)),
		fmt.Sprintf("    log: %s", valueOrDash(worker.LogPath)),
	)
	if worker.PID != 0 {
		lines = append(lines, fmt.Sprintf("    pid: %d", worker.PID))
	}
	return lines
}

func (model Model) formatCPU(pid int) string {
	if pid == 0 {
		return "n/a"
	}
	sample, ok := model.ProcessSamples[pid]
	if !ok || !sample.Available {
		if sample.Reason != "" {
			return "n/a"
		}
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", sample.CPUPercent)
}

func (model *Model) refreshProcessSamples() {
	samples := map[int]ProcessSample{}
	if model.Sampler == nil {
		model.Sampler = sampleProcess
	}
	for _, worker := range model.Snapshot.Workers {
		if worker.PID == 0 {
			continue
		}
		if _, seen := samples[worker.PID]; seen {
			continue
		}
		samples[worker.PID] = model.Sampler(worker.PID)
	}
	model.ProcessSamples = samples
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

func formatElapsed(now time.Time, started time.Time, ended time.Time) string {
	if started.IsZero() {
		return "-"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	end := now
	if !ended.IsZero() {
		end = ended
	}
	if end.Before(started) {
		return "0s"
	}
	return end.Sub(started).Round(time.Second).String()
}

func sortedWorkers(workers []state.WorkerState) []state.WorkerState {
	sorted := append([]state.WorkerState(nil), workers...)
	sort.SliceStable(sorted, func(left, right int) bool {
		if sorted[left].StartedAt.Equal(sorted[right].StartedAt) {
			return sorted[left].ID < sorted[right].ID
		}
		return sorted[left].StartedAt.Before(sorted[right].StartedAt)
	})
	return sorted
}

func sampleProcess(pid int) ProcessSample {
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu=").Output()
	if err != nil {
		return ProcessSample{Available: false, Reason: err.Error()}
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return ProcessSample{Available: false, Reason: "process not found"}
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return ProcessSample{Available: false, Reason: err.Error()}
	}
	return ProcessSample{CPUPercent: value, Available: true}
}
