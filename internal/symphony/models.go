package symphony

import "time"

// WorkState describes whether a Symphony unit can consume automation capacity
// during the current tick. The Go runner uses this state instead of a static
// queue so external waits and skipped work do not occupy worker slots.
type WorkState string

const (
	WorkStateRunnable              WorkState = "runnable"
	WorkStateWaitingExternalReview WorkState = "waiting_external_review"
	WorkStateBlockedActionable     WorkState = "blocked_actionable"
	WorkStateBlockedHuman          WorkState = "blocked_human"
	WorkStateMerged                WorkState = "merged"
	WorkStateSkippedWithReason     WorkState = "skipped_with_reason"
)

// MarkdownSource is one repo-authored Markdown file in the Symphony source
// corpus. Phase planning, prompt rendering, verification selection, and
// self-healing reports use these extracted facts before consulting GitHub.
type MarkdownSource struct {
	Path                   string   `json:"path"`
	Priority               int      `json:"priority"`
	Headings               []string `json:"headings,omitempty"`
	PhaseReferences        []string `json:"phase_references,omitempty"`
	IssueReferences        []int    `json:"issue_references,omitempty"`
	HasAcceptanceCriteria  bool     `json:"has_acceptance_criteria"`
	TargetBranches         []string `json:"target_branches,omitempty"`
	SafetyRuleCount        int      `json:"safety_rule_count"`
	VerificationCommandIDs []string `json:"verification_command_ids,omitempty"`
}

// SourceCorpus summarizes the Markdown files the Go Symphony runner read before
// it planned a tick. The paths are intentionally surfaced in JSON so operators
// can audit which checked-in docs influenced automation.
type SourceCorpus struct {
	Root          string           `json:"root"`
	GeneratedAt   time.Time        `json:"generated_at"`
	TotalFiles    int              `json:"total_files"`
	PriorityFiles int              `json:"priority_files"`
	Sources       []MarkdownSource `json:"sources"`
	Conflicts     []SourceConflict `json:"conflicts,omitempty"`
}

// SourceConflict records source-of-truth disagreements discovered while
// indexing Markdown. The first Go slice only reports branch-target conflicts,
// but the type leaves room for later safety and verification conflicts.
type SourceConflict struct {
	Kind     string   `json:"kind"`
	Value    string   `json:"value"`
	Paths    []string `json:"paths"`
	Decision string   `json:"decision"`
}

// PhaseSlice is a candidate implementation-plan section that can become one or
// more GitHub issues when it has enough documented scope and acceptance criteria.
type PhaseSlice struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	SourcePath         string   `json:"source_path"`
	Headings           []string `json:"headings,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	TargetBranch       string   `json:"target_branch,omitempty"`
	Verification       []string `json:"verification,omitempty"`
}

// WorkItem is the typed unit selected by the work graph. It can represent an
// issue implementation, PR remediation, merge action, workspace repair, or
// self-healing bug fix.
type WorkItem struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`
	State    WorkState `json:"state"`
	Number   int       `json:"number,omitempty"`
	Title    string    `json:"title,omitempty"`
	Branch   string    `json:"branch,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Source   string    `json:"source,omitempty"`
	Priority int       `json:"priority"`
}

// WorkGraph is recomputed every sync tick. Capacity is selected from this graph
// rather than from a one-time static queue.
type WorkGraph struct {
	Items []WorkItem `json:"items"`
}

// CapacityPlan explains how many runnable items may start in the current tick
// and which non-runnable items were deliberately kept out of worker slots.
type CapacityPlan struct {
	MaxRuns          int        `json:"max_runs"`
	RunnableCapacity int        `json:"runnable_capacity"`
	Used             int        `json:"used"`
	RunnableWork     []WorkItem `json:"runnable_work,omitempty"`
	ExternalWaits    []WorkItem `json:"external_waits,omitempty"`
	Skipped          []WorkItem `json:"skipped,omitempty"`
}

// WorkspaceState records the safety classification for a prepared branch
// workspace. Wrong-branch or ambiguous workspaces block; same-branch dirty
// workspaces can be handed back to a worker with context.
type WorkspaceState struct {
	Path       string    `json:"path"`
	Branch     string    `json:"branch,omitempty"`
	State      WorkState `json:"state"`
	DirtyFiles []string  `json:"dirty_files,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// PullRequestState is the PR-side input to the Go work graph.
type PullRequestState struct {
	Number       int         `json:"number"`
	Title        string      `json:"title"`
	HeadRef      string      `json:"head_ref"`
	TargetBranch string      `json:"target_branch"`
	State        WorkState   `json:"state"`
	Review       ReviewState `json:"review"`
	Reason       string      `json:"reason,omitempty"`
}

// ReviewState captures thread-aware Codex Review status as data that can be
// prioritized without blocking unrelated runnable work.
type ReviewState struct {
	PendingExternal bool   `json:"pending_external"`
	UnresolvedCount int    `json:"unresolved_count"`
	BotSignal       string `json:"bot_signal,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// SelfHealingIssue describes a concrete Symphony bug that can be deduplicated
// and promoted ahead of ordinary phase work.
type SelfHealingIssue struct {
	Fingerprint string    `json:"fingerprint"`
	Title       string    `json:"title"`
	State       WorkState `json:"state"`
	Reason      string    `json:"reason"`
}

// TickResult is the public JSON envelope emitted by the Go Symphony entrypoint.
// Legacy runner output is retained while the Go core takes over planning.
type TickResult struct {
	Command              string             `json:"command"`
	DryRun               bool               `json:"dry_run"`
	GeneratedAt          time.Time          `json:"generated_at"`
	SourceCorpus         SourceCorpus       `json:"source_corpus"`
	WorkGraph            WorkGraph          `json:"work_graph"`
	Capacity             CapacityPlan       `json:"capacity"`
	RunnableWork         []WorkItem         `json:"runnable_work,omitempty"`
	ExternalWaits        []WorkItem         `json:"external_waits,omitempty"`
	SelfHealing          []SelfHealingIssue `json:"self_healing,omitempty"`
	IssueMaterialization []PhaseSlice       `json:"issue_materialization,omitempty"`
	ReviewLoops          []WorkItem         `json:"review_loops,omitempty"`
	MergeResults         []WorkItem         `json:"merge_results,omitempty"`
	TopLevelStatus       string             `json:"top_level_status"`
	LegacyStatus         map[string]any     `json:"legacy_status,omitempty"`
}
