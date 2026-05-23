package core_test

import (
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

// TestClassifyManualEmployeeChangeLifecycle locks down which employee-affecting
// manual website changes are reviewed at school-year rollover, preserved until
// upstream reconciliation, or excluded from automatic expiration.
func TestClassifyManualEmployeeChangeLifecycle(t *testing.T) {
	tests := []struct {
		name           string
		changeType     core.ManualEmployeeChangeType
		wantPolicy     core.ManualEmployeeChangeExpirationPolicy
		wantReview     bool
		wantProviderOK bool
	}{
		{
			name:           "manual Non-Escape assignment needs HR rollover review",
			changeType:     core.ManualEmployeeChangeManualNonEscapeAssignment,
			wantPolicy:     core.ManualEmployeeChangeExpiresAfterReview,
			wantReview:     true,
			wantProviderOK: true,
		},
		{
			name:       "temporary site override persists until upstream correction",
			changeType: core.ManualEmployeeChangeTemporarySiteOverride,
			wantPolicy: core.ManualEmployeeChangePersistsUntilUpstream,
			wantReview: true,
		},
		{
			name:       "InformedK12 site selection persists until upstream correction",
			changeType: core.ManualEmployeeChangeInformedK12SiteSelection,
			wantPolicy: core.ManualEmployeeChangePersistsUntilUpstream,
			wantReview: true,
		},
		{
			name:       "permission and site scope mapping never auto expires",
			changeType: core.ManualEmployeeChangePermissionSiteScope,
			wantPolicy: core.ManualEmployeeChangeNeverAutoExpires,
		},
		{
			name:       "local employee attributes never auto expire",
			changeType: core.ManualEmployeeChangeLocalEmployeeAttribute,
			wantPolicy: core.ManualEmployeeChangeNeverAutoExpires,
		},
		{
			name:       "sync exception override can be cleared only as local annual reset metadata",
			changeType: core.ManualEmployeeChangeSyncExceptionOverride,
			wantPolicy: core.ManualEmployeeChangeAnnualResetLocalOnly,
			wantReview: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.ClassifyManualEmployeeChangeLifecycle(tt.changeType)
			if got.Policy != tt.wantPolicy {
				t.Fatalf("policy = %q, want %q", got.Policy, tt.wantPolicy)
			}
			if got.SchoolYearReviewRequired != tt.wantReview {
				t.Fatalf("review required = %t, want %t", got.SchoolYearReviewRequired, tt.wantReview)
			}
			if got.ProviderRemovalAllowed != tt.wantProviderOK {
				t.Fatalf("provider removal allowed = %t, want %t", got.ProviderRemovalAllowed, tt.wantProviderOK)
			}
		})
	}
}

// TestBuildManualEmployeeChangeAuditEntry verifies that HR rollover decisions
// cannot be accepted unless they carry the actor, reason, timestamp, and
// extension metadata needed for the future audit-log row.
func TestBuildManualEmployeeChangeAuditEntry(t *testing.T) {
	decidedAt := time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC)
	newExpiration := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)

	entry, ok := core.BuildManualEmployeeChangeAuditEntry(core.ManualEmployeeChangeReviewInput{
		ChangeID:          "manual-1",
		ChangeType:        core.ManualEmployeeChangeManualNonEscapeAssignment,
		Decision:          core.ManualEmployeeChangeDecisionExtend,
		Actor:             "hr@example.org",
		Reason:            "contract continues next school year",
		DecidedAt:         decidedAt,
		NewExpirationDate: newExpiration,
	})
	if !ok {
		t.Fatal("expected valid extension decision")
	}
	if entry.Actor != "hr@example.org" || entry.Reason == "" || !entry.NewExpirationDate.Equal(newExpiration) {
		t.Fatalf("audit entry = %#v, want actor, reason, and new expiration date preserved", entry)
	}

	_, ok = core.BuildManualEmployeeChangeAuditEntry(core.ManualEmployeeChangeReviewInput{
		ChangeID:   "site-1",
		ChangeType: core.ManualEmployeeChangeTemporarySiteOverride,
		Decision:   core.ManualEmployeeChangeDecisionRemove,
		Actor:      "hr@example.org",
		Reason:     "cleanup",
		DecidedAt:  decidedAt,
	})
	if ok {
		t.Fatal("site overrides must not accept a remove decision during rollover review")
	}

	_, ok = core.BuildManualEmployeeChangeAuditEntry(core.ManualEmployeeChangeReviewInput{
		ChangeID:   "manual-2",
		ChangeType: core.ManualEmployeeChangeManualNonEscapeAssignment,
		Decision:   core.ManualEmployeeChangeDecisionExtend,
		Actor:      "hr@example.org",
		Reason:     "missing expiration date",
		DecidedAt:  decidedAt,
	})
	if ok {
		t.Fatal("extension decisions must require a new expiration date")
	}
}

// TestManualEmployeeChangeProviderRemovalAllowed verifies that rollover cleanup
// cannot become a destructive provider plan unless HR review, what-if
// validation, and the Phase 2 pilot allowlist gate have all passed.
func TestManualEmployeeChangeProviderRemovalAllowed(t *testing.T) {
	if core.ManualEmployeeChangeProviderRemovalAllowed(core.ManualEmployeeChangeProviderWriteInput{
		ChangeType:               core.ManualEmployeeChangeManualNonEscapeAssignment,
		Decision:                 core.ManualEmployeeChangeDecisionRemove,
		HRReviewed:               true,
		WhatIfValidated:          true,
		PilotAllowlisted:         true,
		ProviderRemovalRequested: true,
	}) != true {
		t.Fatal("expected reviewed, validated, allowlisted manual assignment removal to be allowed")
	}

	blocked := []core.ManualEmployeeChangeProviderWriteInput{
		{
			ChangeType:               core.ManualEmployeeChangeManualNonEscapeAssignment,
			Decision:                 core.ManualEmployeeChangeDecisionRemove,
			WhatIfValidated:          true,
			PilotAllowlisted:         true,
			ProviderRemovalRequested: true,
		},
		{
			ChangeType:               core.ManualEmployeeChangeManualNonEscapeAssignment,
			Decision:                 core.ManualEmployeeChangeDecisionRemove,
			HRReviewed:               true,
			PilotAllowlisted:         true,
			ProviderRemovalRequested: true,
		},
		{
			ChangeType:               core.ManualEmployeeChangeTemporarySiteOverride,
			Decision:                 core.ManualEmployeeChangeDecisionRemove,
			HRReviewed:               true,
			WhatIfValidated:          true,
			PilotAllowlisted:         true,
			ProviderRemovalRequested: true,
		},
	}
	for _, input := range blocked {
		if core.ManualEmployeeChangeProviderRemovalAllowed(input) {
			t.Fatalf("provider removal allowed for unsafe input %#v", input)
		}
	}
}
