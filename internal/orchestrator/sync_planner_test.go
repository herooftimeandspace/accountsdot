package orchestrator_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/orchestrator"
)

// TestPlanWorkflowStaffSyncDryRun exercises and documents internal/orchestrator/sync_planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowStaffSyncDryRun(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType:              core.WorkflowTypeStaffSyncDryRun,
		SubjectKind:               core.SubjectKindPerson,
		SubjectID:                 "staff-1",
		PrimaryAssignmentRequired: true,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	assertOperations(t, result.Jobs, []string{
		"internal.sync_ingest_subject",
		"photo.check_delta",
		"incident_iq.resolve_room",
		"incident_iq.resolve_room_asset",
		"zoom.validate_room_membership",
		"zoom.validate_primary_phone_assignment",
		"internal.sync_update_projection",
	})
}

// TestPlanWorkflowStudentSyncDryRun exercises and documents internal/orchestrator/sync_planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowStudentSyncDryRun(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypeStudentSyncDryRun,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "student-1",
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	assertOperations(t, result.Jobs, []string{
		"internal.sync_ingest_subject",
		"photo.check_delta",
		"incident_iq.match_person",
		"internal.sync_update_projection",
	})
}

// TestPlanWorkflowSyncRecheck exercises and documents internal/orchestrator/sync_planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowSyncRecheck(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypeSyncRecheck,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "staff-1",
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	assertOperations(t, result.Jobs, []string{
		"internal.sync_recheck_subject",
	})
}

// TestPlanWorkflowAnnualResetArchive exercises and documents internal/orchestrator/sync_planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowAnnualResetArchive(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypeAnnualResetArchive,
		SubjectKind:  core.SubjectKindWorkbook,
		SubjectID:    "school-year-2026",
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	assertOperations(t, result.Jobs, []string{
		"internal.archive_completed_sync_rows",
		"internal.clear_sync_exception_overrides",
	})
}
