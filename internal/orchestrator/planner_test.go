package orchestrator_test

import (
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/orchestrator"
)

// TestPlanWorkflowPersonOnboardWithRoom exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowPersonOnboardWithRoom(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypePersonOnboard,
		ChangeReason: core.WorkflowChangeReasonReactivateRoleChange,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "person-1",
		RoomKnown:    true,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	if result.WorkflowType != core.WorkflowTypePersonOnboard {
		t.Fatalf("expected workflow type %q, got %q", core.WorkflowTypePersonOnboard, result.WorkflowType)
	}
	if result.ChangeReason != core.WorkflowChangeReasonReactivateRoleChange {
		t.Fatalf("expected change reason %q, got %q", core.WorkflowChangeReasonReactivateRoleChange, result.ChangeReason)
	}

	want := []string{
		"zoom.read_user",
		"zoom.create_or_link_user",
		"internal.reserve_extension",
		"zoom.assign_site_extension",
		"zoom.assign_calling_plan",
		"zoom.ensure_room_slg",
		"zoom.add_room_membership",
	}
	assertOperations(t, result.Jobs, want)
	if len(result.FollowUps) != 0 {
		t.Fatalf("expected no follow-up workflows, got %d", len(result.FollowUps))
	}
}

// TestPlanWorkflowPersonOnboardWithoutRoomSkipsRoomSteps exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowPersonOnboardWithoutRoomSkipsRoomSteps(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypePersonOnboard,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "person-1",
		RoomKnown:    false,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"zoom.read_user",
		"zoom.create_or_link_user",
		"internal.reserve_extension",
		"zoom.assign_site_extension",
		"zoom.assign_calling_plan",
	}
	assertOperations(t, result.Jobs, want)
}

// TestPlanWorkflowSameSiteTransferCreatesRoomCoverageFollowUp exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/planning/external-write-inventory.md.
func TestPlanWorkflowSameSiteTransferCreatesRoomCoverageFollowUp(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType:         core.WorkflowTypePersonSameSiteTransfer,
		SubjectKind:          core.SubjectKindPerson,
		SubjectID:            "person-1",
		RoomKnown:            true,
		OldRoomBecomesVacant: true,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"zoom.ensure_room_slg",
		"zoom.add_room_membership",
		"zoom.remove_room_membership",
	}
	assertOperations(t, result.Jobs, want)
	if len(result.FollowUps) != 1 || result.FollowUps[0].WorkflowType != core.WorkflowTypeRoomCoverage {
		t.Fatalf("expected one room coverage follow-up, got %#v", result.FollowUps)
	}
}

// TestPlanWorkflowSiteTransferRequiresApprovalOnCutover exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowSiteTransferRequiresApprovalOnCutover(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypePersonSiteTransfer,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "person-1",
		RoomKnown:    true,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	var cutover *core.WorkflowJob
	for i := range result.Jobs {
		if result.Jobs[i].Operation == "zoom.apply_site_extension_cutover" {
			cutover = &result.Jobs[i]
			break
		}
	}
	if cutover == nil {
		t.Fatal("expected cutover operation to be present")
	}
	if !cutover.ApprovalRequired {
		t.Fatal("expected site transfer cutover to require approval")
	}
}

// TestPlanWorkflowTerminationWithCoverageCreatesCapFirst exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/planning/external-write-inventory.md.
func TestPlanWorkflowTerminationWithCoverageCreatesCapFirst(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType:         core.WorkflowTypePersonTerminate,
		SubjectKind:          core.SubjectKindPerson,
		SubjectID:            "person-1",
		RoomCoverageRequired: true,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"zoom.ensure_room_cap",
		"zoom.verify_room_cap",
		"zoom.remove_room_membership",
		"zoom.remove_phone_assignment",
		"zoom.deprovision_user",
		"internal.release_extension",
	}
	assertOperations(t, result.Jobs, want)
	for _, job := range result.Jobs[2:] {
		if !job.ApprovalRequired {
			t.Fatalf("expected destructive job %q to require approval", job.Operation)
		}
	}
}

// TestPlanWorkflowPersonLeaveWithoutCoverageIsDestructiveOnly exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowPersonLeaveWithoutCoverageIsDestructiveOnly(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypePersonLeave,
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "person-1",
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"zoom.remove_room_membership",
		"zoom.remove_phone_assignment",
		"zoom.deprovision_user",
		"internal.release_extension",
	}
	assertOperations(t, result.Jobs, want)
}

// TestPlanWorkflowDirectoryPublishUsesDebounce exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowDirectoryPublishUsesDebounce(t *testing.T) {
	now := time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC)
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypeDirectoryPublish,
		SubjectKind:  core.SubjectKindWorkbook,
		SubjectID:    "default-workbook",
		Now:          now,
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"google_sheets.stage_workbook",
		"google_sheets.validate_sentinel",
		"google_sheets.apply_pointers",
	}
	assertOperations(t, result.Jobs, want)
	if !result.RunAfter.Equal(now.Add(time.Minute)) {
		t.Fatalf("expected run_after to be debounced by one minute, got %v", result.RunAfter)
	}
}

// TestPlanWorkflowRoomCoverage exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowRoomCoverage(t *testing.T) {
	result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowTypeRoomCoverage,
		SubjectKind:  core.SubjectKindRoom,
		SubjectID:    "room-1",
	})
	if err != nil {
		t.Fatalf("PlanWorkflow returned error: %v", err)
	}

	want := []string{
		"zoom.ensure_room_cap",
		"zoom.verify_room_cap",
	}
	assertOperations(t, result.Jobs, want)
}

// TestPlanWorkflowUpdateAndContextRefresh exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/planning/external-write-inventory.md.
func TestPlanWorkflowUpdateAndContextRefresh(t *testing.T) {
	for _, workflowType := range []core.WorkflowType{
		core.WorkflowTypePersonUpdate,
		core.WorkflowTypeContextRefresh,
	} {
		result, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
			WorkflowType: workflowType,
			SubjectKind:  core.SubjectKindPerson,
			SubjectID:    "person-1",
		})
		if err != nil {
			t.Fatalf("PlanWorkflow(%q) returned error: %v", workflowType, err)
		}
		assertOperations(t, result.Jobs, []string{"internal.reconcile_subject"})
	}
}

// TestPlanWorkflowUnsupportedTypeReturnsError exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestPlanWorkflowUnsupportedTypeReturnsError(t *testing.T) {
	if _, err := orchestrator.PlanWorkflow(orchestrator.PlanInput{
		WorkflowType: core.WorkflowType("bogus"),
		SubjectKind:  core.SubjectKindPerson,
		SubjectID:    "person-1",
	}); err == nil {
		t.Fatal("expected unsupported workflow type to return an error")
	}
}

// TestDefaultLoopSpecs exercises and documents internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestDefaultLoopSpecs(t *testing.T) {
	specs := orchestrator.DefaultLoopSpecs()
	assertLoopCadence(t, specs, "hr_import_loop", 5*time.Minute)
	assertLoopCadence(t, specs, "aeries_sync_loop", 5*time.Minute)
	assertLoopCadence(t, specs, "context_watcher_loop", 5*time.Minute)
	assertLoopCadence(t, specs, "recovery_loop", 30*time.Second)
	assertLoopCadence(t, specs, "janitor_loop", 2*time.Minute)
	assertLoopCadence(t, specs, "workflow_planner_loop", 30*time.Second)
}

// assertOperations documents the data flow for internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func assertOperations(t *testing.T, jobs []core.WorkflowJob, want []string) {
	t.Helper()
	if len(jobs) != len(want) {
		t.Fatalf("expected %d jobs, got %d", len(want), len(jobs))
	}
	for i, job := range jobs {
		if job.Operation != want[i] {
			t.Fatalf("job %d operation = %q, want %q", i, job.Operation, want[i])
		}
	}
}

// assertLoopCadence documents the data flow for internal/orchestrator/planner_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func assertLoopCadence(t *testing.T, specs []orchestrator.LoopSpec, name string, want time.Duration) {
	t.Helper()
	for _, spec := range specs {
		if spec.Name == name {
			if spec.Cadence != want {
				t.Fatalf("loop %s cadence = %v, want %v", name, spec.Cadence, want)
			}
			return
		}
	}
	t.Fatalf("loop %s not found", name)
}
