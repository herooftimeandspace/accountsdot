package symphony_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony"
)

func TestScanMarkdownCorpusPrioritizesAgentsAndDocs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Project\n")
	writeFile(t, root, ".agents/WORKFLOW.md", "# Workflow\nTarget branch: phase-0-platform-foundation\n- [ ] Dispatch work\n")
	writeFile(t, root, ".agents/skills/example/SKILL.md", "# Skill\nSafety rules matter.\n")
	writeFile(t, root, "docs/planning/implementation-plan.md", "# Phase 0\n## Acceptance Criteria\n- [ ] Ship\n`npm run symphony:test`\n")
	writeFile(t, root, "node_modules/pkg/README.md", "# ignored\n")
	writeFile(t, root, "frontend/dist/generated.md", "# ignored\n")
	writeFile(t, root, "generated/report.md", "# ignored\n")

	corpus, err := symphony.ScanMarkdownCorpus(root)
	if err != nil {
		t.Fatalf("ScanMarkdownCorpus returned error: %v", err)
	}
	if corpus.TotalFiles != 4 {
		t.Fatalf("expected 4 repo-authored markdown files, got %d: %#v", corpus.TotalFiles, corpus.Sources)
	}
	if corpus.Sources[0].Path != "README.md" {
		t.Fatalf("expected README.md to have highest priority, got %s", corpus.Sources[0].Path)
	}
	if corpus.PriorityFiles != 4 {
		t.Fatalf("expected all included files to be priority files, got %d", corpus.PriorityFiles)
	}
}

func TestScanMarkdownCorpusExtractsPlanningFacts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/planning/implementation-plan.md", "# Phase 0\nTarget branch: phase-0-platform-foundation\nFixes #292\n## Acceptance Criteria\n- [ ] Work graph is non-blocking\nRun `go test ./internal/symphony/...`.\nSafety: no production writes.\n")

	corpus, err := symphony.ScanMarkdownCorpus(root)
	if err != nil {
		t.Fatalf("ScanMarkdownCorpus returned error: %v", err)
	}
	source := corpus.Sources[0]
	if !source.HasAcceptanceCriteria {
		t.Fatal("expected acceptance criteria to be detected")
	}
	if len(source.PhaseReferences) != 1 || source.PhaseReferences[0] != "phase 0" {
		t.Fatalf("unexpected phase references: %#v", source.PhaseReferences)
	}
	if len(source.IssueReferences) != 1 || source.IssueReferences[0] != 292 {
		t.Fatalf("unexpected issue references: %#v", source.IssueReferences)
	}
	if len(source.TargetBranches) != 1 || source.TargetBranches[0] != "phase-0-platform-foundation" {
		t.Fatalf("unexpected target branches: %#v", source.TargetBranches)
	}
	if len(source.VerificationCommandIDs) == 0 {
		t.Fatal("expected verification command extraction")
	}
	if source.SafetyRuleCount == 0 {
		t.Fatal("expected safety rule extraction")
	}
}

func TestExtractPhaseSlicesRequiresAcceptanceCriteria(t *testing.T) {
	corpus := symphony.SourceCorpus{Sources: []symphony.MarkdownSource{
		{Path: "docs/planning/implementation-plan.md", Headings: []string{"Phase 0"}, PhaseReferences: []string{"phase 0"}, HasAcceptanceCriteria: true, TargetBranches: []string{"phase-0-platform-foundation"}},
		{Path: "docs/product/product-requirements.md", Headings: []string{"Phase 0"}, PhaseReferences: []string{"phase 0"}},
	}}
	slices := symphony.ExtractPhaseSlices(corpus, "Phase 0")
	if len(slices) != 1 {
		t.Fatalf("expected 1 materializable phase slice, got %d", len(slices))
	}
	if slices[0].SourcePath != "docs/planning/implementation-plan.md" {
		t.Fatalf("unexpected slice source: %s", slices[0].SourcePath)
	}
}

func TestExtractPhaseSlicesAppliesBranchOverride(t *testing.T) {
	corpus := symphony.SourceCorpus{Sources: []symphony.MarkdownSource{
		{Path: "docs/planning/implementation-plan.md", Headings: []string{"Phase 0"}, PhaseReferences: []string{"phase 0"}, HasAcceptanceCriteria: true, TargetBranches: []string{"phase-0-platform-foundation"}},
	}}
	slices := symphony.ExtractPhaseSlices(corpus, "Phase 0", "phase-1-next")
	if len(slices) != 1 {
		t.Fatalf("expected materialized slice, got %d", len(slices))
	}
	if slices[0].TargetBranch != "phase-1-next" {
		t.Fatalf("expected branch override, got %q", slices[0].TargetBranch)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
