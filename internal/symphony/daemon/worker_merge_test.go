package daemon

import (
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/symphony/state"
)

func TestMergeWorkerObservationsPreservesWorkersWhenNoNewWorkIsRunnable(t *testing.T) {
	started := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	existing := []state.WorkerState{{
		ID:         "worker-1",
		WorkItemID: "issue-292",
		Kind:       "issue",
		Status:     "runnable",
		StartedAt:  started,
	}}
	merged := mergeWorkerObservations(existing, nil, started.Add(30*time.Second), time.Minute)
	if len(merged) != 1 || merged[0].WorkItemID != "issue-292" || !merged[0].StartedAt.Equal(started) {
		t.Fatalf("expected existing worker to be preserved, got %#v", merged)
	}
}

func TestMergeWorkerObservationsDropsStaleWorkersWhenNoNewWorkIsRunnable(t *testing.T) {
	started := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	existing := []state.WorkerState{{
		ID:         "worker-1",
		WorkItemID: "issue-292",
		Kind:       "issue",
		Status:     "runnable",
		StartedAt:  started,
	}}
	merged := mergeWorkerObservations(existing, nil, started.Add(2*time.Minute), time.Minute)
	if len(merged) != 0 {
		t.Fatalf("expected stale worker to be dropped, got %#v", merged)
	}
}

func TestMergeWorkerObservationsPreservesWorkerStartTime(t *testing.T) {
	started := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	observedAt := started.Add(time.Minute)
	existing := []state.WorkerState{{
		ID:         "worker-1",
		WorkItemID: "issue-292",
		Kind:       "issue",
		Status:     "runnable",
		StartedAt:  started,
	}}
	observed := []state.WorkerState{{
		ID:          "worker-1",
		WorkItemID:  "issue-292",
		Kind:        "issue",
		Status:      "runnable",
		StartedAt:   observedAt,
		LatestEvent: "still runnable",
	}}
	merged := mergeWorkerObservations(existing, observed, observedAt, time.Minute)
	if len(merged) != 1 || !merged[0].StartedAt.Equal(started) || merged[0].LatestEvent != "still runnable" {
		t.Fatalf("expected observed worker update with original start time, got %#v", merged)
	}
}
