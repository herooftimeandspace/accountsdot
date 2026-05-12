package core

type SyncSubjectType string

const (
	SyncSubjectTypeStaff   SyncSubjectType = "staff"
	SyncSubjectTypeStudent SyncSubjectType = "student"
)

type SyncPhase string

const (
	SyncPhaseIngested        SyncPhase = "ingested"
	SyncPhasePhotoProcessed  SyncPhase = "photo_processed"
	SyncPhaseIIQMatched      SyncPhase = "iiq_matched"
	SyncPhaseRoomMapped      SyncPhase = "room_mapped"
	SyncPhaseZoomProvisioned SyncPhase = "zoom_provisioned"
)

type SyncOverallStatus string

const (
	SyncOverallStatusPending      SyncOverallStatus = "pending"
	SyncOverallStatusInProgress   SyncOverallStatus = "in_progress"
	SyncOverallStatusManualAction SyncOverallStatus = "manual_action"
	SyncOverallStatusCompleted    SyncOverallStatus = "completed"
)

type SyncIssueCode string

const (
	SyncIssueCodeRoomMappingRequired SyncIssueCode = "room_mapping_required"
	SyncIssueCodeLicensingError      SyncIssueCode = "licensing_error"
	SyncIssueCodePrimaryConflict     SyncIssueCode = "primary_conflict"
	SyncIssueCodeMissingAsset        SyncIssueCode = "missing_asset"
	SyncIssueCodeRolloverWait        SyncIssueCode = "rollover_wait"
)

type SyncProjectionInput struct {
	SubjectType            SyncSubjectType
	PhotoProcessed         bool
	IIQMatched             bool
	RoomMapped             bool
	ZoomMembershipVerified bool
	PhoneAssignmentChecked bool
	Issues                 []SyncIssueCode
}

// Valid documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v SyncSubjectType) Valid() bool {
	switch v {
	case SyncSubjectTypeStaff, SyncSubjectTypeStudent:
		return true
	default:
		return false
	}
}

// Valid documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v SyncPhase) Valid() bool {
	switch v {
	case SyncPhaseIngested,
		SyncPhasePhotoProcessed,
		SyncPhaseIIQMatched,
		SyncPhaseRoomMapped,
		SyncPhaseZoomProvisioned:
		return true
	default:
		return false
	}
}

// Valid documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v SyncOverallStatus) Valid() bool {
	switch v {
	case SyncOverallStatusPending,
		SyncOverallStatusInProgress,
		SyncOverallStatusManualAction,
		SyncOverallStatusCompleted:
		return true
	default:
		return false
	}
}

// Valid documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (v SyncIssueCode) Valid() bool {
	switch v {
	case SyncIssueCodeRoomMappingRequired,
		SyncIssueCodeLicensingError,
		SyncIssueCodePrimaryConflict,
		SyncIssueCodeMissingAsset,
		SyncIssueCodeRolloverWait:
		return true
	default:
		return false
	}
}

// ProjectSyncProgress documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func ProjectSyncProgress(input SyncProjectionInput) (SyncPhase, SyncOverallStatus) {
	phase := SyncPhaseIngested
	status := SyncOverallStatusPending

	if input.PhotoProcessed {
		phase = SyncPhasePhotoProcessed
		status = SyncOverallStatusInProgress
	}
	if input.IIQMatched {
		phase = SyncPhaseIIQMatched
		status = SyncOverallStatusInProgress
	}

	switch input.SubjectType {
	case SyncSubjectTypeStaff:
		if input.RoomMapped {
			phase = SyncPhaseRoomMapped
			status = SyncOverallStatusInProgress
		}
		if input.ZoomMembershipVerified && input.PhoneAssignmentChecked {
			phase = SyncPhaseZoomProvisioned
			status = SyncOverallStatusCompleted
		}
	case SyncSubjectTypeStudent:
		if input.PhotoProcessed && input.IIQMatched {
			phase = SyncPhaseIIQMatched
			status = SyncOverallStatusCompleted
		}
	}

	for _, issue := range input.Issues {
		if issue == SyncIssueCodeRolloverWait {
			return phase, SyncOverallStatusInProgress
		}
		if issue.Valid() {
			return phase, SyncOverallStatusManualAction
		}
	}

	return phase, status
}

// AnnualResetDisposition documents the data flow for internal/core/sync.go. Domain logic, orchestrator code, and tests reach this function; debug it by checking enum validity, projection inputs, and expected workflow state outputs. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func AnnualResetDisposition(status SyncOverallStatus) (archive bool, clearOverrides bool) {
	return status == SyncOverallStatusCompleted, true
}
