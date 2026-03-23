package core

import "github.com/google/uuid"

type PersonState string

const (
	PersonStateIntakePending           PersonState = "intake_pending"
	PersonStateNormalized              PersonState = "normalized"
	PersonStateReconciled              PersonState = "reconciled"
	PersonStateProvisionPendingContext PersonState = "provision_pending_context"
	PersonStateAwaitingReview          PersonState = "awaiting_review"
	PersonStatePreProvisionReady       PersonState = "preprovision_ready"
	PersonStatePreProvisioning         PersonState = "preprovisioning"
	PersonStateProvisionReady          PersonState = "provision_ready"
	PersonStateProvisioning            PersonState = "provisioning"
	PersonStateActive                  PersonState = "active"
	PersonStateTransferPending         PersonState = "transfer_pending"
	PersonStateLeavePending            PersonState = "leave_pending"
	PersonStateDeprovisioning          PersonState = "deprovisioning"
	PersonStateTerminated              PersonState = "terminated"
	PersonStateFailed                  PersonState = "failed"
	PersonStateOnHold                  PersonState = "on_hold"
)

type JobState string

const (
	JobStateQueued        JobState = "queued"
	JobStateRunning       JobState = "running"
	JobStateRecovering    JobState = "recovering"
	JobStateBlocked       JobState = "blocked"
	JobStateWaitingManual JobState = "waiting_manual"
	JobStateSucceeded     JobState = "succeeded"
	JobStateFailed        JobState = "failed"
	JobStateSkipped       JobState = "skipped"
	JobStateCanceled      JobState = "canceled"
)

type DuplicateReasonCode string

const (
	DuplicateReasonMatchNameNoDOB         DuplicateReasonCode = "MATCH_NAME_NO_DOB"
	DuplicateReasonMatchNameEmailConflict DuplicateReasonCode = "MATCH_NAME_EMAIL_CONFLICT"
	DuplicateReasonMatchSourceIDReuse     DuplicateReasonCode = "MATCH_SOURCE_ID_REUSE"
	DuplicateReasonContextTimeout         DuplicateReasonCode = "CONTEXT_TIMEOUT"
)

func NewPersonUUID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func (s PersonState) Valid() bool {
	switch s {
	case PersonStateIntakePending,
		PersonStateNormalized,
		PersonStateReconciled,
		PersonStateProvisionPendingContext,
		PersonStateAwaitingReview,
		PersonStatePreProvisionReady,
		PersonStatePreProvisioning,
		PersonStateProvisionReady,
		PersonStateProvisioning,
		PersonStateActive,
		PersonStateTransferPending,
		PersonStateLeavePending,
		PersonStateDeprovisioning,
		PersonStateTerminated,
		PersonStateFailed,
		PersonStateOnHold:
		return true
	default:
		return false
	}
}

func (s JobState) Valid() bool {
	switch s {
	case JobStateQueued,
		JobStateRunning,
		JobStateRecovering,
		JobStateBlocked,
		JobStateWaitingManual,
		JobStateSucceeded,
		JobStateFailed,
		JobStateSkipped,
		JobStateCanceled:
		return true
	default:
		return false
	}
}

func (c DuplicateReasonCode) Valid() bool {
	switch c {
	case DuplicateReasonMatchNameNoDOB,
		DuplicateReasonMatchNameEmailConflict,
		DuplicateReasonMatchSourceIDReuse,
		DuplicateReasonContextTimeout:
		return true
	default:
		return false
	}
}
