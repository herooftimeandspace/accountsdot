package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/control"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/daemon"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
	symphonytui "github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/tui"
)

// main is the Go entrypoint for the repo-owned Symphony automation. It keeps
// the existing npm command surface stable while moving source scanning, work
// graph construction, and capacity/status decisions into typed Go code.
func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: symphony <daemon|status|control|tui|report|sync|ui-monitor|record-browser-results|test> [options]")
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	switch args[0] {
	case "daemon":
		return runDaemon(ctx, repoRoot, args[1:])
	case "status":
		return runStatus(ctx, args[1:])
	case "control":
		return runControl(args[1:])
	case "tui":
		return runTUI(args[1:])
	case "sync":
		return runSync(ctx, repoRoot, args[1:])
	case "report", "ui-monitor", "record-browser-results":
		return runLegacyPassthrough(ctx, repoRoot, args)
	case "test":
		return runTests(ctx, repoRoot)
	default:
		return fmt.Errorf("unknown symphony command %q", args[0])
	}
}

func runSync(ctx context.Context, repoRoot string, args []string) error {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	dryRun := flags.Bool("dry-run", false, "do not mutate git, GitHub, or runner state")
	jsonOnly := flags.Bool("json", false, "print JSON")
	phaseID := flags.String("phase", "", "implementation phase id to materialize")
	phaseBranch := flags.String("phase-branch", "", "phase branch override")
	maxRuns := flags.Int("max-runs", 0, "maximum runnable work items")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := symphony.RunSyncTick(ctx, repoRoot, symphony.SyncOptions{
		DryRun:      *dryRun,
		PhaseID:     *phaseID,
		PhaseBranch: *phaseBranch,
		MaxRuns:     *maxRuns,
	})
	if err != nil {
		return err
	}
	encoded, err := symphony.MarshalTickResult(result)
	if err != nil {
		return err
	}
	if *jsonOnly {
		fmt.Println(string(encoded))
		return nil
	}
	fmt.Println(string(encoded))
	return nil
}

func runDaemon(ctx context.Context, repoRoot string, args []string) error {
	flags := flag.NewFlagSet("daemon", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	dryRun := flags.Bool("dry-run", false, "do not mutate git, GitHub, or daemon state")
	jsonOnly := flags.Bool("json", false, "print JSON")
	phaseID := flags.String("phase", "", "implementation phase id")
	phaseBranch := flags.String("phase-branch", "", "phase branch override")
	maxWorkers := flags.Int("max-workers", 0, "maximum runnable workers")
	maxRuns := flags.Int("max-runs", 0, "compatibility alias for --max-workers")
	maxTicks := flags.Int("max-ticks", 0, "stop after this many daemon ticks")
	stateDir := flags.String("state-dir", daemon.DefaultStateDir, "daemon state directory")
	interval := flags.Duration("interval", 5*time.Minute, "daemon tick interval")
	if err := flags.Parse(args); err != nil {
		return err
	}
	workers := *maxWorkers
	if workers <= 0 {
		workers = *maxRuns
	}
	snapshot, err := daemon.Run(ctx, daemon.Options{
		RepoRoot:    repoRoot,
		StateDir:    *stateDir,
		Phase:       *phaseID,
		PhaseBranch: *phaseBranch,
		Interval:    *interval,
		MaxWorkers:  workers,
		MaxTicks:    *maxTicks,
		DryRun:      *dryRun,
		JSON:        *jsonOnly,
	})
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if *jsonOnly {
		fmt.Println(string(encoded))
		return nil
	}
	fmt.Println(string(encoded))
	return nil
}

func runStatus(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	jsonOnly := flags.Bool("json", false, "print JSON")
	watch := flags.Bool("watch", false, "watch status once per second")
	stateDir := flags.String("state-dir", daemon.DefaultStateDir, "daemon state directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	for {
		snapshot, err := state.ReadSnapshot(*stateDir)
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return err
		}
		if *jsonOnly {
			fmt.Println(string(encoded))
		} else {
			fmt.Printf("Symphony %s: %s\n", snapshot.Controller.Lifecycle, snapshot.Controller.LastStatus)
		}
		if !*watch {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func runControl(args []string) error {
	flags := flag.NewFlagSet("control", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	stateDir := flags.String("state-dir", daemon.DefaultStateDir, "daemon state directory")
	concurrency := flags.Int("concurrency", 0, "worker count for set-concurrency")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() == 0 {
		return fmt.Errorf("usage: symphony control <pause|resume|drain|stop|cancel|set-concurrency> [target]")
	}
	action := flags.Arg(0)
	target := ""
	if action == "cancel" && flags.NArg() > 1 {
		target = flags.Arg(1)
	}
	if action == "set-concurrency" && *concurrency == 0 && flags.NArg() > 1 {
		value, err := strconv.Atoi(flags.Arg(1))
		if err != nil {
			return err
		}
		*concurrency = value
	}
	command, err := control.New(action, target, *concurrency)
	if err != nil {
		return err
	}
	path, err := control.WriteCommand(*stateDir, command)
	if err != nil {
		return err
	}
	fmt.Printf("queued %s command at %s\n", action, path)
	return nil
}

func runTUI(args []string) error {
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	stateDir := flags.String("state-dir", daemon.DefaultStateDir, "daemon state directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	return symphonytui.Run(*stateDir)
}

func runLegacyPassthrough(ctx context.Context, repoRoot string, args []string) error {
	command := exec.CommandContext(ctx, "node", append([]string{"scripts/symphony_runner.mjs"}, args...)...)
	command.Dir = repoRoot
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	return command.Run()
}

func runTests(ctx context.Context, repoRoot string) error {
	legacy := exec.CommandContext(ctx, "node", "scripts/symphony_runner.mjs", "test")
	legacy.Dir = repoRoot
	legacy.Stdout = os.Stdout
	legacy.Stderr = os.Stderr
	legacy.Stdin = os.Stdin
	if err := legacy.Run(); err != nil {
		return fmt.Errorf("legacy symphony self-test: %w", err)
	}
	packages := []string{"test", "./internal/symphony/...", "./cmd/symphony"}
	command := exec.CommandContext(ctx, "go", packages...)
	command.Dir = repoRoot
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = goCacheEnv(os.Environ(), repoRoot)
	if err := command.Run(); err != nil {
		return err
	}
	fmt.Println("go symphony tests passed")
	return nil
}

func goCacheEnv(base []string, repoRoot string) []string {
	env := append([]string{}, base...)
	if !hasEnvKey(env, "GOCACHE") {
		env = append(env, "GOCACHE="+filepath.Join(repoRoot, ".gocache"))
	}
	if !hasEnvKey(env, "GOMODCACHE") {
		env = append(env, "GOMODCACHE="+filepath.Join(repoRoot, ".gomodcache"))
	}
	return env
}

func hasEnvKey(env []string, key string) bool {
	prefix := key + "="
	for _, value := range env {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from current directory")
		}
		dir = parent
	}
}
