package symphony

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BuildWorkGraph converts legacy runner JSON plus the Markdown corpus into the
// Go orchestration model. During migration this lets the Go CLI correct capacity
// and status decisions before every GitHub mutation path has been ported.
func BuildWorkGraph(legacy map[string]any, corpus SourceCorpus, maxRuns int) (WorkGraph, CapacityPlan, string) {
	var graph WorkGraph
	graph.Items = append(graph.Items, pullRequestItems(legacy)...)
	graph.Items = append(graph.Items, issueItems(legacy)...)
	graph.Items = append(graph.Items, selfHealingWorkItems(legacy)...)
	sort.SliceStable(graph.Items, func(i, j int) bool {
		if graph.Items[i].State != graph.Items[j].State {
			return stateRank(graph.Items[i].State) < stateRank(graph.Items[j].State)
		}
		if graph.Items[i].Priority != graph.Items[j].Priority {
			return graph.Items[i].Priority < graph.Items[j].Priority
		}
		return graph.Items[i].ID < graph.Items[j].ID
	})
	capacity := CapacityPlan{MaxRuns: maxRuns, RunnableCapacity: maxRuns}
	for _, item := range graph.Items {
		switch item.State {
		case WorkStateRunnable:
			if capacity.Used < maxRuns {
				capacity.RunnableWork = append(capacity.RunnableWork, item)
				capacity.Used++
			}
		case WorkStateWaitingExternalReview:
			capacity.ExternalWaits = append(capacity.ExternalWaits, item)
		case WorkStateSkippedWithReason:
			capacity.Skipped = append(capacity.Skipped, item)
		}
	}
	return graph, capacity, topLevelStatus(graph, capacity, corpus)
}

// WrapLegacySyncResult adds the Go source corpus, work graph, and corrected
// top-level status around the existing Node runner output.
func WrapLegacySyncResult(legacy map[string]any, corpus SourceCorpus, maxRuns int, dryRun bool) TickResult {
	graph, capacity, status := BuildWorkGraph(legacy, corpus, maxRuns)
	return TickResult{
		Command:        "sync",
		DryRun:         dryRun,
		GeneratedAt:    time.Now().UTC(),
		SourceCorpus:   corpus,
		WorkGraph:      graph,
		Capacity:       capacity,
		RunnableWork:   capacity.RunnableWork,
		ExternalWaits:  capacity.ExternalWaits,
		SelfHealing:    extractSelfHealing(legacy),
		TopLevelStatus: status,
		LegacyStatus:   legacy,
	}
}

func pullRequestItems(legacy map[string]any) []WorkItem {
	queue := objectAt(legacy, "pull_request_queue")
	items := arrayAt(queue, "items")
	work := make([]WorkItem, 0, len(items))
	for index, raw := range items {
		item := rawObject(raw)
		status := stringAt(item, "status")
		number := intAt(item, "number")
		title := stringAt(item, "title")
		state := WorkStateSkippedWithReason
		reason := strings.Join(stringArrayAt(item, "notes"), "; ")
		switch status {
		case "ready_to_merge":
			state = WorkStateRunnable
			reason = "PR is merge-ready"
		case "waiting_for_codex_review":
			state = WorkStateWaitingExternalReview
			if reason == "" {
				reason = "waiting for external Codex Review"
			}
		case "blocked":
			if intAt(item, "unresolved_codex_review_threads") > 0 {
				state = WorkStateRunnable
				reason = "Codex Review remediation is actionable"
			} else {
				state = WorkStateBlockedActionable
				reason = strings.Join(stringArrayAt(item, "blockers"), "; ")
			}
		}
		work = append(work, WorkItem{
			ID:       fmt.Sprintf("pr-%d", number),
			Kind:     "pull_request",
			State:    state,
			Number:   number,
			Title:    title,
			Branch:   stringAt(item, "head_ref"),
			Reason:   reason,
			Source:   "legacy.pull_request_queue.items",
			Priority: 100 + index,
		})
	}
	return work
}

func issueItems(legacy map[string]any) []WorkItem {
	selected := arrayAt(legacy, "selected_issues")
	if len(selected) == 0 {
		selected = arrayAt(legacy, "dispatches")
	}
	work := make([]WorkItem, 0, len(selected))
	for index, raw := range selected {
		item := rawObject(raw)
		number := intAt(item, "number")
		status := stringAt(item, "status")
		state := WorkStateRunnable
		reason := stringAt(item, "reason")
		if status != "" && status != "eligible" && status != "would-prepare" && status != "prepared" && status != "succeeded" {
			state = WorkStateSkippedWithReason
		}
		work = append(work, WorkItem{
			ID:       fmt.Sprintf("issue-%d", number),
			Kind:     "issue",
			State:    state,
			Number:   number,
			Title:    stringAt(item, "title"),
			Branch:   stringAt(item, "branch"),
			Reason:   reason,
			Source:   "legacy.selected_issues",
			Priority: 1000 + index,
		})
	}
	return work
}

func selfHealingWorkItems(legacy map[string]any) []WorkItem {
	queue := objectAt(legacy, "pull_request_queue")
	mergedStates := arrayAt(queue, "merged_workspace_states")
	work := []WorkItem{}
	for _, raw := range mergedStates {
		item := rawObject(raw)
		if stringAt(item, "status") != "blocked" {
			continue
		}
		number := intAt(item, "issue_number")
		work = append(work, WorkItem{
			ID:       fmt.Sprintf("self-heal-issue-%d", number),
			Kind:     "self_healing",
			State:    WorkStateRunnable,
			Number:   number,
			Title:    "Repair Symphony workspace reconciliation blocker",
			Reason:   stringAt(item, "reason"),
			Source:   "legacy.pull_request_queue.merged_workspace_states",
			Priority: 0,
		})
	}
	return work
}

func extractSelfHealing(legacy map[string]any) []SelfHealingIssue {
	items := selfHealingWorkItems(legacy)
	results := make([]SelfHealingIssue, 0, len(items))
	for _, item := range items {
		results = append(results, SelfHealingIssue{
			Fingerprint: item.ID,
			Title:       item.Title,
			State:       item.State,
			Reason:      item.Reason,
		})
	}
	return results
}

func topLevelStatus(graph WorkGraph, capacity CapacityPlan, corpus SourceCorpus) string {
	if len(extractRunnableByKind(graph, "self_healing")) > 0 {
		return "self_healing_dispatchable"
	}
	if len(capacity.RunnableWork) > 0 {
		return "dispatches_available"
	}
	if len(capacity.ExternalWaits) > 0 {
		return "waiting_for_codex_review"
	}
	if corpus.TotalFiles == 0 {
		return "blocked_no_markdown_corpus"
	}
	return "idle"
}

func extractRunnableByKind(graph WorkGraph, kind string) []WorkItem {
	items := []WorkItem{}
	for _, item := range graph.Items {
		if item.Kind == kind && item.State == WorkStateRunnable {
			items = append(items, item)
		}
	}
	return items
}

func stateRank(state WorkState) int {
	switch state {
	case WorkStateRunnable:
		return 0
	case WorkStateBlockedActionable:
		return 1
	case WorkStateWaitingExternalReview:
		return 2
	case WorkStateBlockedHuman:
		return 3
	case WorkStateSkippedWithReason:
		return 4
	case WorkStateMerged:
		return 5
	default:
		return 9
	}
}

func objectAt(value map[string]any, key string) map[string]any {
	return rawObject(value[key])
}

func arrayAt(value map[string]any, key string) []any {
	raw, _ := value[key].([]any)
	return raw
}

func rawObject(value any) map[string]any {
	object, _ := value.(map[string]any)
	if object == nil {
		return map[string]any{}
	}
	return object
}

func stringAt(value map[string]any, key string) string {
	raw, _ := value[key].(string)
	return raw
}

func intAt(value map[string]any, key string) int {
	switch raw := value[key].(type) {
	case float64:
		return int(raw)
	case int:
		return raw
	default:
		return 0
	}
}

func stringArrayAt(value map[string]any, key string) []string {
	raw, _ := value[key].([]any)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
