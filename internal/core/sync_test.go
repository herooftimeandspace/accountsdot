package core_test

import (
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

// TestSyncEnumsValidate exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSyncEnumsValidate(t *testing.T) {
	t.Run("subject types", func(t *testing.T) {
		valid := []core.SyncSubjectType{
			core.SyncSubjectTypeStaff,
			core.SyncSubjectTypeStudent,
		}
		for _, value := range valid {
			if !value.Valid() {
				t.Fatalf("expected sync subject type %q to be valid", value)
			}
		}
		if core.SyncSubjectType("bogus").Valid() {
			t.Fatal("expected bogus sync subject type to be invalid")
		}
	})

	t.Run("phases", func(t *testing.T) {
		valid := []core.SyncPhase{
			core.SyncPhaseIngested,
			core.SyncPhasePhotoProcessed,
			core.SyncPhaseIIQMatched,
			core.SyncPhaseRoomMapped,
			core.SyncPhaseZoomProvisioned,
		}
		for _, value := range valid {
			if !value.Valid() {
				t.Fatalf("expected sync phase %q to be valid", value)
			}
		}
		if core.SyncPhase("bogus").Valid() {
			t.Fatal("expected bogus sync phase to be invalid")
		}
	})

	t.Run("overall statuses", func(t *testing.T) {
		valid := []core.SyncOverallStatus{
			core.SyncOverallStatusPending,
			core.SyncOverallStatusInProgress,
			core.SyncOverallStatusManualAction,
			core.SyncOverallStatusCompleted,
		}
		for _, value := range valid {
			if !value.Valid() {
				t.Fatalf("expected sync overall status %q to be valid", value)
			}
		}
		if core.SyncOverallStatus("bogus").Valid() {
			t.Fatal("expected bogus sync overall status to be invalid")
		}
	})

	t.Run("issue codes", func(t *testing.T) {
		valid := []core.SyncIssueCode{
			core.SyncIssueCodeRoomMappingRequired,
			core.SyncIssueCodeLicensingError,
			core.SyncIssueCodePrimaryConflict,
			core.SyncIssueCodeMissingAsset,
			core.SyncIssueCodeRolloverWait,
		}
		for _, value := range valid {
			if !value.Valid() {
				t.Fatalf("expected sync issue code %q to be valid", value)
			}
		}
		if core.SyncIssueCode("bogus").Valid() {
			t.Fatal("expected bogus sync issue code to be invalid")
		}
	})
}

// TestProjectSyncProgressStaffCompletion exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestProjectSyncProgressStaffCompletion(t *testing.T) {
	phase, status := core.ProjectSyncProgress(core.SyncProjectionInput{
		SubjectType:            core.SyncSubjectTypeStaff,
		PhotoProcessed:         true,
		IIQMatched:             true,
		RoomMapped:             true,
		ZoomMembershipVerified: true,
		PhoneAssignmentChecked: true,
	})

	if phase != core.SyncPhaseZoomProvisioned {
		t.Fatalf("expected final staff phase %q, got %q", core.SyncPhaseZoomProvisioned, phase)
	}
	if status != core.SyncOverallStatusCompleted {
		t.Fatalf("expected completed staff status, got %q", status)
	}
}

// TestProjectSyncProgressStudentCompletion exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestProjectSyncProgressStudentCompletion(t *testing.T) {
	phase, status := core.ProjectSyncProgress(core.SyncProjectionInput{
		SubjectType:    core.SyncSubjectTypeStudent,
		PhotoProcessed: true,
		IIQMatched:     true,
	})

	if phase != core.SyncPhaseIIQMatched {
		t.Fatalf("expected final student phase %q, got %q", core.SyncPhaseIIQMatched, phase)
	}
	if status != core.SyncOverallStatusCompleted {
		t.Fatalf("expected completed student status, got %q", status)
	}
}

// TestProjectSyncProgressManualActionOverridesCompletion exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestProjectSyncProgressManualActionOverridesCompletion(t *testing.T) {
	phase, status := core.ProjectSyncProgress(core.SyncProjectionInput{
		SubjectType:            core.SyncSubjectTypeStaff,
		PhotoProcessed:         true,
		IIQMatched:             true,
		RoomMapped:             true,
		ZoomMembershipVerified: true,
		PhoneAssignmentChecked: true,
		Issues: []core.SyncIssueCode{
			core.SyncIssueCodeLicensingError,
		},
	})

	if phase != core.SyncPhaseZoomProvisioned {
		t.Fatalf("expected progress to remain at %q, got %q", core.SyncPhaseZoomProvisioned, phase)
	}
	if status != core.SyncOverallStatusManualAction {
		t.Fatalf("expected manual action status, got %q", status)
	}
}

// TestProjectSyncProgressRolloverWaitIsInProgress exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestProjectSyncProgressRolloverWaitIsInProgress(t *testing.T) {
	phase, status := core.ProjectSyncProgress(core.SyncProjectionInput{
		SubjectType:            core.SyncSubjectTypeStaff,
		PhotoProcessed:         true,
		IIQMatched:             true,
		RoomMapped:             true,
		ZoomMembershipVerified: true,
		PhoneAssignmentChecked: true,
		Issues: []core.SyncIssueCode{
			core.SyncIssueCodeRolloverWait,
		},
	})

	if phase != core.SyncPhaseZoomProvisioned {
		t.Fatalf("expected progress to remain at %q, got %q", core.SyncPhaseZoomProvisioned, phase)
	}
	if status != core.SyncOverallStatusInProgress {
		t.Fatalf("expected in-progress status for rollover wait, got %q", status)
	}
}

// TestAnnualResetDisposition exercises and documents internal/core/sync_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestAnnualResetDisposition(t *testing.T) {
	archive, clearOverrides := core.AnnualResetDisposition(core.SyncOverallStatusCompleted)
	if !archive {
		t.Fatal("expected completed sync rows to be archived at annual reset")
	}
	if !clearOverrides {
		t.Fatal("expected annual reset to clear per-user exception overrides")
	}

	archive, clearOverrides = core.AnnualResetDisposition(core.SyncOverallStatusManualAction)
	if archive {
		t.Fatal("expected manual-action rows to remain active, not archived")
	}
	if !clearOverrides {
		t.Fatal("expected annual reset to clear per-user exception overrides for active rows too")
	}
}
