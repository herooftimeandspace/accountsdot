package core

import "time"

type ManualEmployeeChangeType string

const (
	ManualEmployeeChangeTemporarySiteOverride     ManualEmployeeChangeType = "temporary_site_override"
	ManualEmployeeChangeRoomLocationOverride      ManualEmployeeChangeType = "room_location_override"
	ManualEmployeeChangeManualNonEscapeAssignment ManualEmployeeChangeType = "manual_non_escape_assignment"
	ManualEmployeeChangeInformedK12SiteSelection  ManualEmployeeChangeType = "informedk12_site_selection"
	ManualEmployeeChangePermissionSiteScope       ManualEmployeeChangeType = "permission_site_scope_override"
	ManualEmployeeChangeLocalEmployeeAttribute    ManualEmployeeChangeType = "local_employee_attribute"
	ManualEmployeeChangeSyncExceptionOverride     ManualEmployeeChangeType = "sync_exception_override"
)

type ManualEmployeeChangeExpirationPolicy string

const (
	ManualEmployeeChangeExpiresAfterReview    ManualEmployeeChangeExpirationPolicy = "expires_after_hr_review"
	ManualEmployeeChangePersistsUntilUpstream ManualEmployeeChangeExpirationPolicy = "persists_until_upstream_correction"
	ManualEmployeeChangeNeverAutoExpires      ManualEmployeeChangeExpirationPolicy = "never_auto_expires"
	ManualEmployeeChangeAnnualResetLocalOnly  ManualEmployeeChangeExpirationPolicy = "annual_reset_local_only"
	ManualEmployeeChangeNeedsProductDecision  ManualEmployeeChangeExpirationPolicy = "needs_product_decision"
)

type ManualEmployeeChangeReviewDecision string

const (
	ManualEmployeeChangeDecisionExtend     ManualEmployeeChangeReviewDecision = "extend"
	ManualEmployeeChangeDecisionRemove     ManualEmployeeChangeReviewDecision = "remove"
	ManualEmployeeChangeDecisionReconciled ManualEmployeeChangeReviewDecision = "reconciled"
)

type ManualEmployeeChangeLifecycle struct {
	Policy                   ManualEmployeeChangeExpirationPolicy
	SchoolYearReviewRequired bool
	AllowedDecisions         []ManualEmployeeChangeReviewDecision
	ProviderRemovalAllowed   bool
}

type ManualEmployeeChangeReviewInput struct {
	ChangeID          string
	ChangeType        ManualEmployeeChangeType
	Decision          ManualEmployeeChangeReviewDecision
	Actor             string
	Reason            string
	DecidedAt         time.Time
	NewExpirationDate time.Time
}

type ManualEmployeeChangeAuditEntry struct {
	ChangeID          string
	ChangeType        ManualEmployeeChangeType
	Decision          ManualEmployeeChangeReviewDecision
	Actor             string
	Reason            string
	DecidedAt         time.Time
	NewExpirationDate time.Time
}

type ManualEmployeeChangeProviderWriteInput struct {
	ChangeType               ManualEmployeeChangeType
	Decision                 ManualEmployeeChangeReviewDecision
	HRReviewed               bool
	WhatIfValidated          bool
	PilotAllowlisted         bool
	ProviderRemovalRequested bool
}

// ClassifyManualEmployeeChangeLifecycle returns the first-pass school-year
// lifecycle policy for employee-affecting dashboard changes. Sync planning and
// future review queues call this before deciding whether annual reset can hide
// local metadata or plan a provider-affecting removal.
func ClassifyManualEmployeeChangeLifecycle(changeType ManualEmployeeChangeType) ManualEmployeeChangeLifecycle {
	switch changeType {
	case ManualEmployeeChangeManualNonEscapeAssignment:
		return ManualEmployeeChangeLifecycle{
			Policy:                   ManualEmployeeChangeExpiresAfterReview,
			SchoolYearReviewRequired: true,
			AllowedDecisions: []ManualEmployeeChangeReviewDecision{
				ManualEmployeeChangeDecisionExtend,
				ManualEmployeeChangeDecisionRemove,
				ManualEmployeeChangeDecisionReconciled,
			},
			ProviderRemovalAllowed: true,
		}
	case ManualEmployeeChangeTemporarySiteOverride,
		ManualEmployeeChangeRoomLocationOverride,
		ManualEmployeeChangeInformedK12SiteSelection:
		return ManualEmployeeChangeLifecycle{
			Policy:                   ManualEmployeeChangePersistsUntilUpstream,
			SchoolYearReviewRequired: true,
			AllowedDecisions: []ManualEmployeeChangeReviewDecision{
				ManualEmployeeChangeDecisionExtend,
				ManualEmployeeChangeDecisionReconciled,
			},
		}
	case ManualEmployeeChangeSyncExceptionOverride:
		return ManualEmployeeChangeLifecycle{
			Policy:                   ManualEmployeeChangeAnnualResetLocalOnly,
			SchoolYearReviewRequired: true,
			AllowedDecisions: []ManualEmployeeChangeReviewDecision{
				ManualEmployeeChangeDecisionExtend,
				ManualEmployeeChangeDecisionRemove,
				ManualEmployeeChangeDecisionReconciled,
			},
		}
	case ManualEmployeeChangePermissionSiteScope,
		ManualEmployeeChangeLocalEmployeeAttribute:
		return ManualEmployeeChangeLifecycle{
			Policy: ManualEmployeeChangeNeverAutoExpires,
		}
	default:
		return ManualEmployeeChangeLifecycle{
			Policy:                   ManualEmployeeChangeNeedsProductDecision,
			SchoolYearReviewRequired: true,
		}
	}
}

// BuildManualEmployeeChangeAuditEntry validates a review decision and returns
// the audit payload future DB-backed handlers must persist with the actor,
// timestamp, reason, and extension date before applying rollover decisions.
func BuildManualEmployeeChangeAuditEntry(input ManualEmployeeChangeReviewInput) (ManualEmployeeChangeAuditEntry, bool) {
	if input.ChangeID == "" || input.Actor == "" || input.Reason == "" || input.DecidedAt.IsZero() {
		return ManualEmployeeChangeAuditEntry{}, false
	}
	lifecycle := ClassifyManualEmployeeChangeLifecycle(input.ChangeType)
	if !reviewDecisionAllowed(lifecycle.AllowedDecisions, input.Decision) {
		return ManualEmployeeChangeAuditEntry{}, false
	}
	if input.Decision == ManualEmployeeChangeDecisionExtend && input.NewExpirationDate.IsZero() {
		return ManualEmployeeChangeAuditEntry{}, false
	}
	if input.Decision != ManualEmployeeChangeDecisionExtend && !input.NewExpirationDate.IsZero() {
		return ManualEmployeeChangeAuditEntry{}, false
	}
	return ManualEmployeeChangeAuditEntry{
		ChangeID:          input.ChangeID,
		ChangeType:        input.ChangeType,
		Decision:          input.Decision,
		Actor:             input.Actor,
		Reason:            input.Reason,
		DecidedAt:         input.DecidedAt,
		NewExpirationDate: input.NewExpirationDate,
	}, true
}

// ManualEmployeeChangeProviderRemovalAllowed enforces the destructive-write
// gate for rollover cleanup. Future provider workers can only remove external
// access after HR review chooses removal and the Phase 2 what-if plus pilot
// allowlist gates have both passed.
func ManualEmployeeChangeProviderRemovalAllowed(input ManualEmployeeChangeProviderWriteInput) bool {
	if !input.ProviderRemovalRequested {
		return true
	}
	lifecycle := ClassifyManualEmployeeChangeLifecycle(input.ChangeType)
	return lifecycle.ProviderRemovalAllowed &&
		input.Decision == ManualEmployeeChangeDecisionRemove &&
		input.HRReviewed &&
		input.WhatIfValidated &&
		input.PilotAllowlisted
}

func reviewDecisionAllowed(allowed []ManualEmployeeChangeReviewDecision, decision ManualEmployeeChangeReviewDecision) bool {
	for _, candidate := range allowed {
		if candidate == decision {
			return true
		}
	}
	return false
}
