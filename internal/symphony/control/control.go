package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Command is the local operator request consumed by the daemon. The file-based
// transport is deliberately simple so a paused or degraded daemon can still be
// inspected and controlled from a terminal without a hosted service.
type Command struct {
	ID             string    `json:"id"`
	Action         string    `json:"action"`
	TargetWorkerID string    `json:"target_worker_id,omitempty"`
	Concurrency    int       `json:"concurrency,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Result records that a command file was observed and applied.
type Result struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	AppliedAt time.Time `json:"applied_at"`
}

// New builds a validated operator command.
func New(action string, target string, concurrency int) (Command, error) {
	switch action {
	case "pause", "resume", "drain", "stop":
		if target != "" {
			return Command{}, fmt.Errorf("%s does not accept a worker target", action)
		}
	case "cancel", "cancel-worker":
		if target == "" {
			return Command{}, fmt.Errorf("cancel requires a worker id")
		}
		action = "cancel"
	case "set-concurrency":
		if concurrency <= 0 {
			return Command{}, fmt.Errorf("set-concurrency requires a positive worker count")
		}
	default:
		return Command{}, fmt.Errorf("unsupported control action %q", action)
	}
	now := time.Now().UTC()
	return Command{
		ID:             fmt.Sprintf("%d", now.UnixNano()),
		Action:         action,
		TargetWorkerID: target,
		Concurrency:    concurrency,
		CreatedAt:      now,
	}, nil
}

// WriteCommand writes the command into stateDir/control so the daemon can apply
// it on the next control poll.
func WriteCommand(stateDir string, command Command) (string, error) {
	controlDir := filepath.Join(stateDir, "control")
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(command, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(controlDir, command.ID+".json")
	return path, os.WriteFile(path, append(data, '\n'), 0o644)
}

// ReadPending loads command files in creation-name order. The caller owns
// deleting or moving command files after applying them.
func ReadPending(stateDir string) ([]Command, error) {
	controlDir := filepath.Join(stateDir, "control")
	entries, err := os.ReadDir(controlDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	commands := []Command{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var command Command
		data, err := os.ReadFile(filepath.Join(controlDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &command); err != nil {
			return nil, fmt.Errorf("decode %s: %w", entry.Name(), err)
		}
		commands = append(commands, command)
	}
	return commands, nil
}

// MarkApplied records a command result and removes the pending command file.
func MarkApplied(stateDir string, command Command, result Result) error {
	if result.AppliedAt.IsZero() {
		result.AppliedAt = time.Now().UTC()
	}
	resultsDir := filepath.Join(stateDir, "control-results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(resultsDir, command.ID+".json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Remove(filepath.Join(stateDir, "control", command.ID+".json"))
}
