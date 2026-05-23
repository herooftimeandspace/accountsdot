package core_test

import (
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

func TestInformedK12ExactEmployeeIDMatch(t *testing.T) {
	form := informedK12TestForm()
	form.EmployeeID = "103118"
	candidates := []core.InformedK12PersonCandidate{
		{PeopleUUID: "person-other", EmployeeID: "103772", PrimaryEmail: "jamie.reed@wusd.org"},
		{PeopleUUID: "person-alex", EmployeeID: "103118", PrimaryEmail: "alex.ramirez@wusd.org"},
	}

	decision := core.MatchInformedK12Form(form, candidates)

	if decision.Status != core.InformedK12MatchExact || decision.PersonUUID != "person-alex" {
		t.Fatalf("decision = %#v, want exact person-alex", decision)
	}
	if len(decision.Evidence) != 1 || decision.Evidence[0].Rule != "employee_id" {
		t.Fatalf("evidence = %#v, want employee_id rule", decision.Evidence)
	}
}

func TestInformedK12ExactGeneratedContractorIDMatch(t *testing.T) {
	form := informedK12TestForm()
	form.EmployeeID = "6601042"
	form.PersonEmail = ""
	form.RequestorEmail = ""
	candidates := []core.InformedK12PersonCandidate{
		{PeopleUUID: "person-contractor", RecordKind: "contractor", GeneratedEmployeeID: "6601042", PrimaryEmail: "contractor@example.invalid"},
	}

	decision := core.MatchInformedK12Form(form, candidates)

	if decision.Status != core.InformedK12MatchExact || decision.PersonUUID != "person-contractor" {
		t.Fatalf("decision = %#v, want exact contractor match", decision)
	}
}

func TestInformedK12AmbiguousEmployeeIDMatchNeedsReview(t *testing.T) {
	form := informedK12TestForm()
	form.EmployeeID = "6612345"
	candidates := []core.InformedK12PersonCandidate{
		{PeopleUUID: "person-one", EmployeeID: "6612345"},
		{PeopleUUID: "person-two", GeneratedEmployeeID: "6612345"},
	}

	decision := core.MatchInformedK12Form(form, candidates)

	if decision.Status != core.InformedK12MatchAmbiguous {
		t.Fatalf("status = %q, want ambiguous", decision.Status)
	}
	if decision.PersonUUID != "" {
		t.Fatalf("ambiguous decision attached to %q", decision.PersonUUID)
	}
	if len(decision.Evidence) != 2 {
		t.Fatalf("evidence = %#v, want both conflicting candidates", decision.Evidence)
	}
}

func TestInformedK12LegalNameWithoutDateDoesNotAutoAttach(t *testing.T) {
	form := informedK12TestForm()
	form.EmployeeID = ""
	form.PersonEmail = ""
	form.RequestorEmail = ""
	form.EffectiveDate = time.Time{}
	candidates := []core.InformedK12PersonCandidate{
		{PeopleUUID: "person-alex", LegalFirstName: "Alex", LegalLastName: "Ramirez", StartDate: date("2026-07-01")},
	}

	decision := core.MatchInformedK12Form(form, candidates)

	if decision.Status != core.InformedK12MatchNeedsReview {
		t.Fatalf("status = %q, want needs_review", decision.Status)
	}
	if decision.PersonUUID != "" {
		t.Fatalf("review decision attached to %q", decision.PersonUUID)
	}
}

func TestInformedK12ManualAttachReviewRecordsEvidenceUse(t *testing.T) {
	form := informedK12TestForm()
	person := core.InformedK12PersonCandidate{PeopleUUID: "person-alex", SiteID: "clover-hs"}
	reviewedAt := dateTime("2026-05-22T15:30:00Z")

	attachment := core.ReviewInformedK12Attachment(form, person, core.InformedK12EvidenceUsePrimarySiteDecision, "hr-reviewer", reviewedAt, "Escape has two highest-category assignments; form confirms Clover High School.")

	if attachment.State != core.InformedK12AttachmentStateAttached {
		t.Fatalf("state = %q, want attached", attachment.State)
	}
	if attachment.EvidenceUse != core.InformedK12EvidenceUsePrimarySiteDecision {
		t.Fatalf("evidence use = %q, want primary site decision", attachment.EvidenceUse)
	}
	if attachment.ReviewedBy != "hr-reviewer" || !attachment.ReviewedAt.Equal(reviewedAt) {
		t.Fatalf("review metadata = %#v", attachment)
	}
}

func TestInformedK12DetachedAndSupersededKeepSourceForm(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUseRelatedOnly,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)

	detached := core.DetachInformedK12Attachment(attachment, "it-admin", dateTime("2026-05-22T16:00:00Z"), "attached to the wrong person")
	if detached.State != core.InformedK12AttachmentStateDetached || detached.Form.SourceFormID != attachment.Form.SourceFormID {
		t.Fatalf("detached attachment = %#v", detached)
	}

	superseded := core.SupersedeInformedK12Attachment(attachment, "ik12-2002", "hr-reviewer", dateTime("2026-05-23T16:00:00Z"), "newer site-change form")
	if superseded.State != core.InformedK12AttachmentStateSuperseded || superseded.SupersededBy != "ik12-2002" {
		t.Fatalf("superseded attachment = %#v", superseded)
	}
	if superseded.Form.SourceFields[0].Value != "Clover High School" {
		t.Fatalf("source fields were normalized or lost: %#v", superseded.Form.SourceFields)
	}
}

func TestInformedK12PersonaRedaction(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUseLifecycleDecision,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)

	full := core.RedactInformedK12AttachmentForPersona(attachment, core.InformedK12VisibilityContext{PersonaRole: core.PermissionRoleHumanResources, AttachmentSite: "clover-hs"})
	if full.Visibility != core.InformedK12VisibilityFull || full.SourceURL == "" || len(full.VisibleFields) != 3 {
		t.Fatalf("full visibility = %#v", full)
	}

	summary := core.RedactInformedK12AttachmentForPersona(attachment, core.InformedK12VisibilityContext{PersonaRole: core.PermissionRoleSiteSecretary, VisibleSiteIDs: []string{"clover-hs"}, AttachmentSite: "clover-hs"})
	if summary.Visibility != core.InformedK12VisibilitySummary {
		t.Fatalf("summary visibility = %#v", summary)
	}
	if summary.SourceURL != "" {
		t.Fatalf("summary should not expose source URL, got %q", summary.SourceURL)
	}
	if len(summary.VisibleFields) != 1 || summary.VisibleFields[0].Key != "site" {
		t.Fatalf("summary visible fields = %#v, want only public site field", summary.VisibleFields)
	}
	if len(summary.RedactedFields) != 2 {
		t.Fatalf("summary redactions = %#v, want personnel and sensitive fields", summary.RedactedFields)
	}

	hidden := core.RedactInformedK12AttachmentForPersona(attachment, core.InformedK12VisibilityContext{PersonaRole: core.PermissionRoleFacultyStaff, AttachmentSite: "clover-hs"})
	if hidden.Visibility != core.InformedK12VisibilityHidden || hidden.SourceFormID != "" {
		t.Fatalf("hidden visibility leaked metadata: %#v", hidden)
	}
}

func TestInformedK12ClearSiteChangeSignalPreservesSourceValues(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUsePrimarySiteDecision,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
		},
		EscapeSite: core.InformedK12EscapeSiteSnapshot{SiteID: "CLA", SiteName: "Clover High School"},
		Now:        dateTime("2026-05-23T15:30:00Z"),
		StaleAfter: 7 * 24 * time.Hour,
	})

	if signal.Status != core.InformedK12SiteSignalClear {
		t.Fatalf("status = %q, want clear: %#v", signal.Status, signal)
	}
	if signal.Confidence != core.InformedK12SiteSignalConfidenceHigh {
		t.Fatalf("confidence = %q, want high", signal.Confidence)
	}
	if signal.SourceFormID != "ik12-1001" || signal.ParsedSiteID != "CLA" {
		t.Fatalf("signal identifiers = %#v", signal)
	}
	if len(signal.FieldRefs) != 1 || signal.FieldRefs[0].RawValue != "Clover High School" {
		t.Fatalf("field refs = %#v, want exact raw site value", signal.FieldRefs)
	}
}

func TestInformedK12MissingSiteChangeSignal(t *testing.T) {
	form := informedK12TestForm()
	form.SourceFields = []core.InformedK12SourceField{
		{Key: "salary_step", Label: "Salary Step", Value: "T3", Sensitivity: core.InformedK12FieldSensitive},
	}
	attachment := core.ReviewInformedK12Attachment(form, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUseRelatedOnly, "hr-reviewer", dateTime("2026-05-22T15:30:00Z"), "manual review")

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{Attachment: attachment})

	if signal.Status != core.InformedK12SiteSignalMissing {
		t.Fatalf("status = %q, want missing: %#v", signal.Status, signal)
	}
	if signal.Confidence != core.InformedK12SiteSignalConfidenceNone {
		t.Fatalf("confidence = %q, want none", signal.Confidence)
	}
	if len(signal.FieldRefs) != 0 || len(signal.ReviewReasons) == 0 {
		t.Fatalf("signal = %#v, want no refs and a review reason", signal)
	}
}

func TestInformedK12AmbiguousSiteChangeSignal(t *testing.T) {
	form := informedK12TestForm()
	form.SourceFields = []core.InformedK12SourceField{
		{Key: "site", Label: "Current Site", Value: "Clover High School", Sensitivity: core.InformedK12FieldPublic},
		{Key: "transfer_site", Label: "Transfer Site", Value: "North County Campus", Sensitivity: core.InformedK12FieldPublic},
	}
	attachment := core.ReviewInformedK12Attachment(form, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUsePrimarySiteDecision, "hr-reviewer", dateTime("2026-05-22T15:30:00Z"), "manual review")

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
			{RawValue: "North County Campus", SiteID: "NCC", SiteName: "North County Campus"},
		},
	})

	if signal.Status != core.InformedK12SiteSignalAmbiguous {
		t.Fatalf("status = %q, want ambiguous: %#v", signal.Status, signal)
	}
	if len(signal.RawSiteValues) != 2 {
		t.Fatalf("raw values = %#v, want both source values", signal.RawSiteValues)
	}
}

func TestInformedK12StaleSiteChangeSignal(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUsePrimarySiteDecision,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
		},
		Now:        dateTime("2026-06-01T00:00:00Z"),
		StaleAfter: 7 * 24 * time.Hour,
	})

	if signal.Status != core.InformedK12SiteSignalStale {
		t.Fatalf("status = %q, want stale: %#v", signal.Status, signal)
	}
	if signal.ParsedSiteID != "CLA" {
		t.Fatalf("parsed site = %q, want CLA", signal.ParsedSiteID)
	}
}

func TestInformedK12SiteChangeSignalConflictsWithEscape(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUsePrimarySiteDecision,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
		},
		EscapeSite: core.InformedK12EscapeSiteSnapshot{SiteID: "DO", SiteName: "District Office"},
		Now:        dateTime("2026-05-23T15:30:00Z"),
		StaleAfter: 7 * 24 * time.Hour,
	})

	if signal.Status != core.InformedK12SiteSignalConflicting {
		t.Fatalf("status = %q, want conflicting: %#v", signal.Status, signal)
	}
	if signal.Confidence != core.InformedK12SiteSignalConfidenceMedium {
		t.Fatalf("confidence = %q, want medium", signal.Confidence)
	}
}

func TestLatestInformedK12SiteChangeSignalUsesNewestActiveAttachment(t *testing.T) {
	olderForm := informedK12TestForm()
	olderForm.SourceFormID = "ik12-older"
	olderForm.SubmittedAt = dateTime("2026-05-21T14:00:00Z")
	newerForm := informedK12TestForm()
	newerForm.SourceFormID = "ik12-newer"
	newerForm.SubmittedAt = dateTime("2026-05-23T14:00:00Z")
	older := core.ReviewInformedK12Attachment(olderForm, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUseRelatedOnly, "hr-reviewer", dateTime("2026-05-22T15:30:00Z"), "manual review")
	newer := core.ReviewInformedK12Attachment(newerForm, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUsePrimarySiteDecision, "hr-reviewer", dateTime("2026-05-23T15:30:00Z"), "manual review")

	signal := core.LatestInformedK12SiteChangeSignal([]core.InformedK12Attachment{older, newer}, []core.InformedK12SiteAlias{
		{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
	}, core.InformedK12EscapeSiteSnapshot{SiteID: "CLA", SiteName: "Clover High School"}, dateTime("2026-05-24T00:00:00Z"), 7*24*time.Hour)

	if signal.SourceFormID != "ik12-newer" || signal.Status != core.InformedK12SiteSignalClear {
		t.Fatalf("latest signal = %#v, want clear ik12-newer", signal)
	}
}

func TestInformedK12ReviewAttachmentDoesNotBecomeActiveSiteSignal(t *testing.T) {
	attachment := core.ReviewInformedK12Attachment(
		informedK12TestForm(),
		core.InformedK12PersonCandidate{PeopleUUID: "person-alex"},
		core.InformedK12EvidenceUsePrimarySiteDecision,
		"hr-reviewer",
		dateTime("2026-05-22T15:30:00Z"),
		"manual review",
	)
	attachment.State = core.InformedK12AttachmentStateReview

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
		},
	})

	if signal.Status != core.InformedK12SiteSignalMissing || len(signal.FieldRefs) != 0 {
		t.Fatalf("review attachment signal = %#v, want missing without field refs", signal)
	}
}

func TestLatestInformedK12SiteChangeSignalSkipsReviewAttachment(t *testing.T) {
	olderForm := informedK12TestForm()
	olderForm.SourceFormID = "ik12-approved"
	olderForm.SubmittedAt = dateTime("2026-05-21T14:00:00Z")
	reviewForm := informedK12TestForm()
	reviewForm.SourceFormID = "ik12-review"
	reviewForm.SubmittedAt = dateTime("2026-05-23T14:00:00Z")
	reviewForm.SourceFields[0].Value = "North County Campus"
	approved := core.ReviewInformedK12Attachment(olderForm, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUseRelatedOnly, "hr-reviewer", dateTime("2026-05-22T15:30:00Z"), "manual review")
	review := core.ReviewInformedK12Attachment(reviewForm, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUsePrimarySiteDecision, "hr-reviewer", dateTime("2026-05-23T15:30:00Z"), "manual review")
	review.State = core.InformedK12AttachmentStateReview

	signal := core.LatestInformedK12SiteChangeSignal([]core.InformedK12Attachment{approved, review}, []core.InformedK12SiteAlias{
		{RawValue: "Clover High School", SiteID: "CLA", SiteName: "Clover High School"},
		{RawValue: "North County Campus", SiteID: "NCC", SiteName: "North County Campus"},
	}, core.InformedK12EscapeSiteSnapshot{}, dateTime("2026-05-24T00:00:00Z"), 7*24*time.Hour)

	if signal.SourceFormID != "ik12-approved" || signal.ParsedSiteID != "CLA" {
		t.Fatalf("latest signal = %#v, want approved attachment to remain active", signal)
	}
}

func TestInformedK12SiteFieldClassifierUsesWholeTerms(t *testing.T) {
	form := informedK12TestForm()
	form.SourceFields = []core.InformedK12SourceField{
		{Key: "website_url", Label: "Onsite orientation website", Value: "https://example.invalid/clover", Sensitivity: core.InformedK12FieldPublic},
	}
	attachment := core.ReviewInformedK12Attachment(form, core.InformedK12PersonCandidate{PeopleUUID: "person-alex"}, core.InformedK12EvidenceUseRelatedOnly, "hr-reviewer", dateTime("2026-05-22T15:30:00Z"), "manual review")

	signal := core.BuildInformedK12SiteChangeSignal(core.InformedK12SiteSignalInput{
		Attachment: attachment,
		SiteAliases: []core.InformedK12SiteAlias{
			{RawValue: "https://example.invalid/clover", SiteID: "CLA", SiteName: "Clover High School"},
		},
	})

	if signal.Status != core.InformedK12SiteSignalMissing || len(signal.FieldRefs) != 0 {
		t.Fatalf("website/onsite field signal = %#v, want missing without site refs", signal)
	}
}

func informedK12TestForm() core.InformedK12FormRecord {
	return core.InformedK12FormRecord{
		SourceFormID:   "ik12-1001",
		FormType:       "Position Change",
		SubmittedAt:    dateTime("2026-05-22T14:00:00Z"),
		Status:         "submitted",
		SubmitterEmail: "principal@wusd.org",
		RequestorEmail: "alex.ramirez@wusd.org",
		PersonEmail:    "alex.ramirez@wusd.org",
		LegalFirstName: "Alex",
		LegalLastName:  "Ramirez",
		EffectiveDate:  date("2026-07-01"),
		SourceURL:      "https://informedk12.example.invalid/forms/ik12-1001",
		SourceFields: []core.InformedK12SourceField{
			{Key: "site", Label: "Site", Value: "Clover High School", Sensitivity: core.InformedK12FieldPublic},
			{Key: "salary_step", Label: "Salary Step", Value: "T3", Sensitivity: core.InformedK12FieldSensitive},
			{Key: "supervisor_notes", Label: "Supervisor Notes", Value: "Preserve exactly as written", Sensitivity: core.InformedK12FieldPersonnel},
		},
	}
}

func date(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func dateTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
