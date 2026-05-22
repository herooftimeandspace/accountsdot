package symphony

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// SyncOptions is the typed input shared by the one-shot CLI and the daemon
// tick loop. Keeping this in the core package prevents the daemon from
// shelling back into cmd/symphony and gives tests one place to exercise the
// Markdown scan, legacy adapter, work graph, and status envelope.
type SyncOptions struct {
	DryRun      bool
	PhaseID     string
	PhaseBranch string
	MaxRuns     int
}

// RunSyncTick executes one Symphony planning/dispatch tick. During the Go
// migration it still delegates GitHub mutation paths to the legacy Node runner,
// then wraps that output with the Go source corpus, work graph, capacity plan,
// and corrected top-level status.
func RunSyncTick(ctx context.Context, repoRoot string, options SyncOptions) (TickResult, error) {
	if options.PhaseBranch != "" && options.PhaseID == "" {
		return TickResult{}, fmt.Errorf("--phase-branch requires --phase so the override applies only to phase materialization")
	}
	if options.PhaseBranch != "" && !options.DryRun {
		return TickResult{}, fmt.Errorf("--phase-branch is supported for dry-run phase materialization only until native Go dispatch replaces the legacy adapter")
	}

	corpus, err := ScanMarkdownCorpus(repoRoot)
	if err != nil {
		return TickResult{}, fmt.Errorf("scan markdown source corpus: %w", err)
	}
	legacyArgs := []string{"sync"}
	if options.DryRun {
		legacyArgs = append(legacyArgs, "--dry-run")
	}
	legacyArgs = append(legacyArgs, "--json")
	if options.MaxRuns > 0 {
		legacyArgs = append(legacyArgs, "--max-runs", strconv.Itoa(options.MaxRuns))
	}
	legacy, _, err := RunLegacyNodeRunner(ctx, repoRoot, legacyArgs...)
	if err != nil {
		return TickResult{}, err
	}
	effectiveMaxRuns := options.MaxRuns
	if effectiveMaxRuns <= 0 {
		effectiveMaxRuns = IntFromLegacy(legacy, "dispatcher", "max_concurrent_runs")
		if effectiveMaxRuns <= 0 {
			effectiveMaxRuns = 1
		}
	}
	result := WrapLegacySyncResult(legacy, corpus, effectiveMaxRuns, options.DryRun)
	result.IssueMaterialization = ExtractPhaseSlices(corpus, options.PhaseID, options.PhaseBranch)
	if options.PhaseBranch != "" {
		result.LegacyStatus["phase_branch_override"] = options.PhaseBranch
	}
	result.GeneratedAt = time.Now().UTC()
	return result, nil
}

// MarshalTickResult formats the public status envelope in the same stable
// shape used by the CLI and daemon dry-run output.
func MarshalTickResult(result TickResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

// IntFromLegacy reads a nested numeric field from legacy JSON. The command
// package uses this indirectly through RunSyncTick, and tests keep coverage in
// the core package so daemon code does not need to duplicate JSON traversal.
func IntFromLegacy(value map[string]any, path ...string) int {
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
