package orchestrator_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/orchestrator"
)

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
