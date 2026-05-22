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

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
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
		return fmt.Errorf("usage: symphony <report|sync|ui-monitor|record-browser-results|test> [options]")
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	switch args[0] {
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
	if *phaseBranch != "" && *phaseID == "" {
		return fmt.Errorf("--phase-branch requires --phase so the override applies only to phase materialization")
	}
	if *phaseBranch != "" && !*dryRun {
		return fmt.Errorf("--phase-branch is supported for dry-run phase materialization only until native Go dispatch replaces the legacy adapter")
	}

	corpus, err := symphony.ScanMarkdownCorpus(repoRoot)
	if err != nil {
		return fmt.Errorf("scan markdown source corpus: %w", err)
	}
	legacyArgs := []string{"sync"}
	if *dryRun {
		legacyArgs = append(legacyArgs, "--dry-run")
	}
	legacyArgs = append(legacyArgs, "--json")
	if *maxRuns > 0 {
		legacyArgs = append(legacyArgs, "--max-runs", strconv.Itoa(*maxRuns))
	}
	legacy, _, err := symphony.RunLegacyNodeRunner(ctx, repoRoot, legacyArgs...)
	if err != nil {
		return err
	}
	effectiveMaxRuns := *maxRuns
	if effectiveMaxRuns <= 0 {
		effectiveMaxRuns = intFromLegacy(legacy, "dispatcher", "max_concurrent_runs")
		if effectiveMaxRuns <= 0 {
			effectiveMaxRuns = 1
		}
	}
	result := symphony.WrapLegacySyncResult(legacy, corpus, effectiveMaxRuns, *dryRun)
	result.IssueMaterialization = symphony.ExtractPhaseSlices(corpus, *phaseID, *phaseBranch)
	if *phaseBranch != "" {
		result.LegacyStatus["phase_branch_override"] = *phaseBranch
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
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

func intFromLegacy(value map[string]any, path ...string) int {
	current := any(value)
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		current = object[key]
	}
	switch typed := current.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}
