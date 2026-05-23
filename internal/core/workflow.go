package core

type WorkflowType string

type WorkflowChangeReason string

const (
	WorkflowTypePersonOnboard          WorkflowType = "person_onboard"
	WorkflowTypePersonUpdate           WorkflowType = "person_update"
	WorkflowTypePersonSameSiteTransfer WorkflowType = "person_same_site_transfer"
	WorkflowTypePersonSiteTransfer     WorkflowType = "person_site_transfer"
	WorkflowTypePersonLeave            WorkflowType = "person_leave"
	WorkflowTypePersonTerminate        WorkflowType = "person_terminate"
	WorkflowTypeRoomCoverage           WorkflowType = "room_coverage"
	WorkflowTypeDirectoryPublish       WorkflowType = "directory_publish"
	WorkflowTypeContextRefresh         WorkflowType = "context_refresh"
	WorkflowTypeStaffSyncDryRun        WorkflowType = "staff_sync_dry_run"
	WorkflowTypeStudentSyncDryRun      WorkflowType = "student_sync_dry_run"
	WorkflowTypeSyncRecheck            WorkflowType = "sync_recheck"
	WorkflowTypeAnnualResetArchive     WorkflowType = "annual_reset_archive"
)

const (
	WorkflowChangeReasonAssignmentAdd                   WorkflowChangeReason = "assignment_add"
	WorkflowChangeReasonRoleChange                      WorkflowChangeReason = "role_change"
	WorkflowChangeReasonSameSiteTransfer                WorkflowChangeReason = "same_site_transfer"
	WorkflowChangeReasonSiteTransfer                    WorkflowChangeReason = "site_transfer"
	WorkflowChangeReasonReactivateSameRole              WorkflowChangeReason = "reactivate_same_role"
	WorkflowChangeReasonReactivateRoleChange            WorkflowChangeReason = "reactivate_role_change"
	WorkflowChangeReasonReactivateNonEscape             WorkflowChangeReason = "reactivate_non_escape"
	WorkflowChangeReasonEmployeeContractorContinuation  WorkflowChangeReason = "employee_contractor_continuation"
	WorkflowChangeReasonActiveEscapeContractorCollision WorkflowChangeReason = "active_escape_contractor_collision"
)

type WorkflowRunState string

const (
	WorkflowRunStatePlanned       WorkflowRunState = "planned"
	WorkflowRunStateDeferred      WorkflowRunState = "deferred"
	WorkflowRunStateRunning       WorkflowRunState = "running"
	WorkflowRunStateWaitingManual WorkflowRunState = "waiting_manual"
	WorkflowRunStateBlocked       WorkflowRunState = "blocked"
	WorkflowRunStateRecovering    WorkflowRunState = "recovering"
	WorkflowRunStateSucceeded     WorkflowRunState = "succeeded"
	WorkflowRunStateFailed        WorkflowRunState = "failed"
	WorkflowRunStateCanceled      WorkflowRunState = "canceled"
)

type ApprovalState string

const (
	ApprovalStateNotRequired ApprovalState = "not_required"
	ApprovalStatePending     ApprovalState = "pending"
	ApprovalStateApproved    ApprovalState = "approved"
	ApprovalStateRejected    ApprovalState = "rejected"
	ApprovalStateExpired     ApprovalState = "expired"
)

type ProviderKind string

const (
	ProviderKindHRSFTP       ProviderKind = "hr_sftp"
	ProviderKindAeries       ProviderKind = "aeries"
	ProviderKindZoom         ProviderKind = "zoom"
	ProviderKindGoogleSheets ProviderKind = "google_sheets"
	ProviderKindInternal     ProviderKind = "internal"
	ProviderKindIncidentIQ   ProviderKind = "incident_iq"
	ProviderKindPhoto        ProviderKind = "photo"
)

type SubjectKind string

const (
	SubjectKindPerson   SubjectKind = "person"
	SubjectKindRoom     SubjectKind = "room"
	SubjectKindWorkbook SubjectKind = "workbook"
)

type WorkflowJob struct {
	StepKey          string
	Provider         ProviderKind
	Operation        string
	DependsOnStepKey string
	ApprovalRequired bool
}

// Valid reports whether a workflow type is one of the planner-supported
// lifecycle, directory, or sync workflows. Planner tests and route handlers use
// this guard before accepting operator intent, so adding a workflow type must
// update this switch before jobs can be planned or persisted.
func (v WorkflowType) Valid() bool {
	switch v {
	case WorkflowTypePersonOnboard,
		WorkflowTypePersonUpdate,
		WorkflowTypePersonSameSiteTransfer,
		WorkflowTypePersonSiteTransfer,
		WorkflowTypePersonLeave,
		WorkflowTypePersonTerminate,
		WorkflowTypeRoomCoverage,
		WorkflowTypeDirectoryPublish,
		WorkflowTypeContextRefresh,
		WorkflowTypeStaffSyncDryRun,
		WorkflowTypeStudentSyncDryRun,
		WorkflowTypeSyncRecheck,
		WorkflowTypeAnnualResetArchive:
		return true
	default:
		return false
	}
}

// Valid reports whether a change reason can be stored with a planned workflow
// or DEV mock action. These values explain why a workflow exists, so invalid
// strings are rejected before they can hide assignment, transfer, reactivation,
// or contractor-collision decisions from tests and operator diagnostics.
func (v WorkflowChangeReason) Valid() bool {
	switch v {
	case WorkflowChangeReasonAssignmentAdd,
		WorkflowChangeReasonRoleChange,
		WorkflowChangeReasonSameSiteTransfer,
		WorkflowChangeReasonSiteTransfer,
		WorkflowChangeReasonReactivateSameRole,
		WorkflowChangeReasonReactivateRoleChange,
		WorkflowChangeReasonReactivateNonEscape,
		WorkflowChangeReasonEmployeeContractorContinuation,
		WorkflowChangeReasonActiveEscapeContractorCollision:
		return true
	default:
		return false
	}
}

// Valid reports whether a workflow run status can be persisted or returned from
// database orchestration helpers. Scheduled-run overlap protection depends on
// `deferred` being valid so a cadence tick can record work that must wait
// without mutating the still-running family owner.
func (v WorkflowRunState) Valid() bool {
	switch v {
	case WorkflowRunStatePlanned,
		WorkflowRunStateDeferred,
		WorkflowRunStateRunning,
		WorkflowRunStateWaitingManual,
		WorkflowRunStateBlocked,
		WorkflowRunStateRecovering,
		WorkflowRunStateSucceeded,
		WorkflowRunStateFailed,
		WorkflowRunStateCanceled:
		return true
	default:
		return false
	}
}

// Valid reports whether an approval state belongs to the workflow approval
// contract used by planner output, approval routes, and database rows. Rejected
// values keep tests and handlers from treating unknown approval text as an
// executable workflow decision.
func (v ApprovalState) Valid() bool {
	switch v {
	case ApprovalStateNotRequired,
		ApprovalStatePending,
		ApprovalStateApproved,
		ApprovalStateRejected,
		ApprovalStateExpired:
		return true
	default:
		return false
	}
}

// Valid reports whether a provider key is part of the current planner and job
// execution contract. The scheduler, planner tests, and external-write
// inventory use this list to separate internal work from provider-specific
// planned operations before any live SDK write exists.
func (v ProviderKind) Valid() bool {
	switch v {
	case ProviderKindHRSFTP,
		ProviderKindAeries,
		ProviderKindZoom,
		ProviderKindGoogleSheets,
		ProviderKindInternal,
		ProviderKindIncidentIQ,
		ProviderKindPhoto:
		return true
	default:
		return false
	}
}
