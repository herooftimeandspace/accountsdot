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
	WorkflowChangeReasonActiveEscapeContractorCollision WorkflowChangeReason = "active_escape_contractor_collision"
)

type WorkflowRunState string

const (
	WorkflowRunStatePlanned       WorkflowRunState = "planned"
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

// Valid documents the data flow for internal/core/workflow.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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

// Valid documents the data flow for internal/core/workflow.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v WorkflowChangeReason) Valid() bool {
	switch v {
	case WorkflowChangeReasonAssignmentAdd,
		WorkflowChangeReasonRoleChange,
		WorkflowChangeReasonSameSiteTransfer,
		WorkflowChangeReasonSiteTransfer,
		WorkflowChangeReasonReactivateSameRole,
		WorkflowChangeReasonReactivateRoleChange,
		WorkflowChangeReasonReactivateNonEscape,
		WorkflowChangeReasonActiveEscapeContractorCollision:
		return true
	default:
		return false
	}
}

// Valid documents the data flow for internal/core/workflow.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v WorkflowRunState) Valid() bool {
	switch v {
	case WorkflowRunStatePlanned,
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

// Valid documents the data flow for internal/core/workflow.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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

// Valid documents the data flow for internal/core/workflow.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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
