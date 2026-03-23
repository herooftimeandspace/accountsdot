package core

type WorkflowType string

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
		WorkflowTypeContextRefresh:
		return true
	default:
		return false
	}
}

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

func (v ProviderKind) Valid() bool {
	switch v {
	case ProviderKindHRSFTP,
		ProviderKindAeries,
		ProviderKindZoom,
		ProviderKindGoogleSheets,
		ProviderKindInternal:
		return true
	default:
		return false
	}
}
