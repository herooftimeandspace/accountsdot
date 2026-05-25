package symphony_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
)

func TestBuildWorkGraphDoesNotLetReviewWaitConsumeRunnableIssueSlots(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{
				map[string]any{
					"number":   319.0,
					"title":    "Waiting PR",
					"head_ref": "codex/waiting",
					"status":   "waiting_for_codex_review",
					"notes":    []any{"waiting for Codex Review response"},
				},
			},
		},
		"selected_issues": []any{
			map[string]any{"number": 292.0, "title": "Fix Symphony", "branch": "codex/issue-292-fix-symphony", "status": "eligible"},
			map[string]any{"number": 228.0, "title": "Add report", "branch": "codex/issue-228-report", "status": "eligible"},
		},
	}

	graph, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "dispatches_available" {
		t.Fatalf("expected runnable work to drive top-level status, got %q", status)
	}
	if len(capacity.RunnableWork) != 2 {
		t.Fatalf("expected 2 runnable issue workers, got %d from graph %#v", len(capacity.RunnableWork), graph)
	}
	if len(capacity.ExternalWaits) != 1 {
		t.Fatalf("expected 1 external review wait, got %d", len(capacity.ExternalWaits))
	}
}

func TestBuildWorkGraphUsesDispatchOutcomeForSelectedIssues(t *testing.T) {
	legacy := map[string]any{
		"selected_issues": []any{
			map[string]any{"number": 292.0, "title": "Fix Symphony", "branch": "codex/issue-292-fix-symphony", "status": "eligible"},
		},
		"dispatches": []any{
			map[string]any{"number": 292.0, "status": "failed", "reason": "agent runner failed"},
		},
	}

	_, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "blocked_actionable" {
		t.Fatalf("expected failed dispatch to produce actionable blocker status, got %q", status)
	}
	if len(capacity.RunnableWork) != 0 {
		t.Fatalf("expected failed dispatch not to remain runnable, got %#v", capacity.RunnableWork)
	}
}

func TestBuildWorkGraphTreatsSucceededDispatchAsCompleted(t *testing.T) {
	legacy := map[string]any{
		"selected_issues": []any{
			map[string]any{"number": 292.0, "title": "Fix Symphony", "branch": "codex/issue-292-fix-symphony", "status": "eligible"},
		},
		"dispatches": []any{
			map[string]any{"number": 292.0, "status": "succeeded", "reason": "agent runner complete"},
		},
	}

	graph, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "idle" {
		t.Fatalf("expected succeeded dispatch not to create runnable status, got %q", status)
	}
	if len(capacity.RunnableWork) != 0 {
		t.Fatalf("expected succeeded dispatch not to consume capacity, got %#v", capacity.RunnableWork)
	}
	if len(graph.Items) != 1 || graph.Items[0].State != symphony.WorkStateMerged {
		t.Fatalf("expected completed issue item, got %#v", graph.Items)
	}
}

func TestBuildWorkGraphReportsBlockedActionableInsteadOfIdle(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{
				map[string]any{
					"number":   319.0,
					"title":    "Blocked PR",
					"head_ref": "codex/blocked",
					"status":   "blocked",
					"blockers": []any{"merge state DIRTY"},
				},
			},
		},
	}

	_, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "blocked_actionable" {
		t.Fatalf("expected blocked actionable status, got %q", status)
	}
	if len(capacity.RunnableWork) != 0 || len(capacity.ExternalWaits) != 0 {
		t.Fatalf("expected no runnable or external waits, got %#v", capacity)
	}
}

func TestBuildWorkGraphReportsBlockedActionableBeforeReviewWait(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{
				map[string]any{
					"number":   319.0,
					"title":    "Waiting PR",
					"head_ref": "codex/waiting",
					"status":   "waiting_for_codex_review",
					"notes":    []any{"waiting for external Codex Review"},
				},
				map[string]any{
					"number":   320.0,
					"title":    "Blocked PR",
					"head_ref": "codex/blocked",
					"status":   "blocked",
					"blockers": []any{"merge state DIRTY"},
				},
			},
		},
	}

	_, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "blocked_actionable" {
		t.Fatalf("expected blocked actionable status before review wait, got %q", status)
	}
	if len(capacity.ExternalWaits) != 1 {
		t.Fatalf("expected external wait to remain visible, got %#v", capacity.ExternalWaits)
	}
}

func TestBuildWorkGraphKeepsBlockedPRRemediationOutOfCapacity(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{
				map[string]any{
					"number":                          319.0,
					"title":                           "Blocked remediation",
					"head_ref":                        "codex/remediation",
					"status":                          "blocked",
					"unresolved_codex_review_threads": 1.0,
					"unresolved_codex_review_thread_summaries": []any{"Fix capacity accounting"},
				},
			},
			"review_remediations": []any{
				map[string]any{"number": 319.0, "status": "blocked", "reason": "workspace is on the wrong branch"},
			},
		},
	}

	graph, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "blocked_actionable" {
		t.Fatalf("expected blocked remediation to produce actionable blocker, got %q", status)
	}
	if len(capacity.RunnableWork) != 0 {
		t.Fatalf("expected blocked remediation not to consume capacity, got %#v", capacity.RunnableWork)
	}
	if len(graph.Items) != 1 || graph.Items[0].State != symphony.WorkStateBlockedActionable {
		t.Fatalf("expected blocked PR item, got %#v", graph.Items)
	}
}

func TestBuildWorkGraphCopiesPRRemediationWorkspaceDiagnostics(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{
				map[string]any{
					"number":                          328.0,
					"title":                           "TUI diagnostics",
					"head_ref":                        "codex/issue-327-symphony-tui-diagnostics",
					"status":                          "blocked",
					"unresolved_codex_review_threads": 1.0,
				},
			},
			"review_remediations": []any{
				map[string]any{
					"number":     328.0,
					"status":     "would-remediate",
					"workspace":  "/private/tmp/accountsdot-symphony/pr-328",
					"state_path": "/private/tmp/accountsdot-symphony/pr-328/state.json",
				},
			},
		},
	}

	graph, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 6)
	if status != "dispatches_available" {
		t.Fatalf("expected runnable remediation status, got %q", status)
	}
	if len(capacity.RunnableWork) != 1 {
		t.Fatalf("expected one runnable remediation, got %#v", capacity.RunnableWork)
	}
	item := graph.Items[0]
	if item.Workspace != "/private/tmp/accountsdot-symphony/pr-328" {
		t.Fatalf("expected remediation workspace on PR item, got %#v", item)
	}
	if item.LogPath != "/private/tmp/accountsdot-symphony/pr-328/state.json" {
		t.Fatalf("expected remediation state path as log pointer, got %#v", item)
	}
}

func TestBuildWorkGraphPromotesSelfHealingBlockers(t *testing.T) {
	legacy := map[string]any{
		"pull_request_queue": map[string]any{
			"items": []any{},
			"merged_workspace_states": []any{
				map[string]any{"issue_number": 248.0, "status": "blocked", "reason": "fatal: not a git repository"},
			},
		},
		"selected_issues": []any{
			map[string]any{"number": 292.0, "title": "Fix Symphony", "branch": "codex/issue-292-fix-symphony", "status": "eligible"},
		},
	}

	_, capacity, status := symphony.BuildWorkGraph(legacy, symphony.SourceCorpus{TotalFiles: 1}, 1)
	if status != "self_healing_dispatchable" {
		t.Fatalf("expected self-healing status, got %q", status)
	}
	if len(capacity.RunnableWork) != 1 || capacity.RunnableWork[0].Kind != "self_healing" {
		t.Fatalf("expected self-healing to take first capacity slot, got %#v", capacity.RunnableWork)
	}
}
