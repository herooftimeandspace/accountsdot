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
