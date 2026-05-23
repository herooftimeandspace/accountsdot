package core_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

// TestWorkflowTypesValidate exercises and documents internal/core/workflow_types_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestWorkflowTypesValidate(t *testing.T) {
	valid := []core.WorkflowType{
		core.WorkflowTypePersonOnboard,
		core.WorkflowTypePersonSiteTransfer,
		core.WorkflowTypeRoomCoverage,
		core.WorkflowTypeDirectoryPublish,
		core.WorkflowTypeStaffSyncDryRun,
		core.WorkflowTypeStudentSyncDryRun,
		core.WorkflowTypeSyncRecheck,
		core.WorkflowTypeAnnualResetArchive,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Fatalf("expected workflow type %q to be valid", value)
		}
	}
	if core.WorkflowType("bogus").Valid() {
		t.Fatal("expected bogus workflow type to be invalid")
	}
}

// TestWorkflowRunStatesValidate exercises and documents internal/core/workflow_types_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestWorkflowRunStatesValidate(t *testing.T) {
	valid := []core.WorkflowRunState{
		core.WorkflowRunStatePlanned,
		core.WorkflowRunStateDeferred,
		core.WorkflowRunStateWaitingManual,
		core.WorkflowRunStateRecovering,
		core.WorkflowRunStateSucceeded,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Fatalf("expected workflow state %q to be valid", value)
		}
	}
	if core.WorkflowRunState("bogus").Valid() {
		t.Fatal("expected bogus workflow state to be invalid")
	}
}

// TestWorkflowChangeReasonsValidate exercises and documents internal/core/workflow_types_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestWorkflowChangeReasonsValidate(t *testing.T) {
	valid := []core.WorkflowChangeReason{
		core.WorkflowChangeReasonAssignmentAdd,
		core.WorkflowChangeReasonRoleChange,
		core.WorkflowChangeReasonSameSiteTransfer,
		core.WorkflowChangeReasonSiteTransfer,
		core.WorkflowChangeReasonReactivateSameRole,
		core.WorkflowChangeReasonReactivateRoleChange,
		core.WorkflowChangeReasonReactivateNonEscape,
		core.WorkflowChangeReasonEmployeeContractorContinuation,
		core.WorkflowChangeReasonActiveEscapeContractorCollision,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Fatalf("expected workflow change reason %q to be valid", value)
		}
	}
	if core.WorkflowChangeReason("bogus").Valid() {
		t.Fatal("expected bogus workflow change reason to be invalid")
	}
}

// TestApprovalStatesValidate exercises and documents internal/core/workflow_types_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestApprovalStatesValidate(t *testing.T) {
	valid := []core.ApprovalState{
		core.ApprovalStateNotRequired,
		core.ApprovalStatePending,
		core.ApprovalStateApproved,
		core.ApprovalStateRejected,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Fatalf("expected approval state %q to be valid", value)
		}
	}
	if core.ApprovalState("bogus").Valid() {
		t.Fatal("expected bogus approval state to be invalid")
	}
}

// TestProviderKindsValidate exercises and documents internal/core/workflow_types_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestProviderKindsValidate(t *testing.T) {
	valid := []core.ProviderKind{
		core.ProviderKindHRSFTP,
		core.ProviderKindAeries,
		core.ProviderKindZoom,
		core.ProviderKindGoogleSheets,
		core.ProviderKindInternal,
		core.ProviderKindIncidentIQ,
		core.ProviderKindPhoto,
	}
	for _, value := range valid {
		if !value.Valid() {
			t.Fatalf("expected provider kind %q to be valid", value)
		}
	}
	if core.ProviderKind("bogus").Valid() {
		t.Fatal("expected bogus provider kind to be invalid")
	}
}
