package core

import (
	"sort"
	"strings"
	"time"
)

type InformedK12AttachmentState string

const (
	InformedK12AttachmentStateAttached   InformedK12AttachmentState = "attached"
	InformedK12AttachmentStateReview     InformedK12AttachmentState = "review"
	InformedK12AttachmentStateDetached   InformedK12AttachmentState = "detached"
	InformedK12AttachmentStateSuperseded InformedK12AttachmentState = "superseded"
)

type InformedK12EvidenceUse string

const (
	InformedK12EvidenceUseRelatedOnly         InformedK12EvidenceUse = "related_only"
	InformedK12EvidenceUsePrimarySiteDecision InformedK12EvidenceUse = "primary_site_decision"
	InformedK12EvidenceUseLifecycleDecision   InformedK12EvidenceUse = "lifecycle_decision"
)

type InformedK12MatchStatus string

const (
	InformedK12MatchExact          InformedK12MatchStatus = "exact"
	InformedK12MatchNeedsReview    InformedK12MatchStatus = "needs_review"
	InformedK12MatchAmbiguous      InformedK12MatchStatus = "ambiguous"
	InformedK12MatchNoCandidate    InformedK12MatchStatus = "no_candidate"
	InformedK12MatchManualReviewed InformedK12MatchStatus = "manual_reviewed"
)

type InformedK12VisibilityLevel string

const (
	InformedK12VisibilityHidden  InformedK12VisibilityLevel = "hidden"
	InformedK12VisibilitySummary InformedK12VisibilityLevel = "summary"
	InformedK12VisibilityFull    InformedK12VisibilityLevel = "full"
)

type InformedK12FieldSensitivity string

const (
	InformedK12FieldPublic    InformedK12FieldSensitivity = "public"
	InformedK12FieldPersonnel InformedK12FieldSensitivity = "personnel"
	InformedK12FieldSensitive InformedK12FieldSensitivity = "sensitive"
)

type InformedK12SourceField struct {
	Key         string
	Label       string
	Value       string
	Sensitivity InformedK12FieldSensitivity
}

type InformedK12FormRecord struct {
	SourceFormID   string
	FormType       string
	SubmittedAt    time.Time
	Status         string
	SubmitterEmail string
	RequestorEmail string
	EmployeeID     string
	PersonEmail    string
	LegalFirstName string
	LegalLastName  string
	EffectiveDate  time.Time
	SourceFields   []InformedK12SourceField
	SourceURL      string
}

type InformedK12PersonCandidate struct {
	PeopleUUID          string
	RecordKind          string
	EmployeeID          string
	GeneratedEmployeeID string
	PrimaryEmail        string
	AlternateEmails     []string
	LegalFirstName      string
	LegalLastName       string
	StartDate           time.Time
	ChangeDates         []time.Time
	SiteID              string
}

type InformedK12MatchEvidence struct {
	Rule        string
	PersonUUID  string
	SourceValue string
	PersonValue string
}

type InformedK12MatchDecision struct {
	Status     InformedK12MatchStatus
	PersonUUID string
	Evidence   []InformedK12MatchEvidence
	Conflicts  []string
}

type InformedK12Attachment struct {
	Form         InformedK12FormRecord
	PeopleUUID   string
	State        InformedK12AttachmentState
	EvidenceUse  InformedK12EvidenceUse
	DecisionNote string
	ReviewedBy   string
	ReviewedAt   time.Time
	SupersededBy string
	DetachedBy   string
	DetachedAt   time.Time
	DetachReason string
}

type InformedK12VisibilityContext struct {
	PersonaRole      PermissionRole
	VisibleSiteIDs   []string
	AttachmentSite   string
	CanViewOwnRecord bool
}

type InformedK12VisibleAttachment struct {
	SourceFormID   string
	FormType       string
	SubmittedAt    time.Time
	Status         string
	SourceURL      string
	EvidenceUse    InformedK12EvidenceUse
	State          InformedK12AttachmentState
	Visibility     InformedK12VisibilityLevel
	VisibleFields  []InformedK12SourceField
	RedactedFields []string
}

// MatchInformedK12Form evaluates issue #253 form-to-person attachment rules without
// mutating source records. Employee id and email are strong exact matches; legal
// name alone is never enough because the form must remain reviewable when a
// date or identifier is missing.
func MatchInformedK12Form(form InformedK12FormRecord, candidates []InformedK12PersonCandidate) InformedK12MatchDecision {
	if len(candidates) == 0 {
		return InformedK12MatchDecision{Status: InformedK12MatchNoCandidate, Conflicts: []string{"no candidate records were supplied"}}
	}
	if decision := uniqueStrongMatch(form, candidates, "employee_id", form.EmployeeID, candidateEmployeeIDs); decision.Status != "" {
		return decision
	}
	if decision := uniqueStrongMatch(form, candidates, "email", firstNonEmpty(form.PersonEmail, form.RequestorEmail), candidateEmails); decision.Status != "" {
		return decision
	}

	nameDateMatches := nameAndDateMatches(form, candidates)
	if len(nameDateMatches) == 1 {
		return InformedK12MatchDecision{
			Status:     InformedK12MatchExact,
			PersonUUID: nameDateMatches[0].PeopleUUID,
			Evidence: []InformedK12MatchEvidence{{
				Rule:        "legal_name_and_effective_date",
				PersonUUID:  nameDateMatches[0].PeopleUUID,
				SourceValue: strings.TrimSpace(form.LegalFirstName + " " + form.LegalLastName + " " + formatDate(form.EffectiveDate)),
				PersonValue: strings.TrimSpace(nameDateMatches[0].LegalFirstName + " " + nameDateMatches[0].LegalLastName + " " + candidateDateEvidence(nameDateMatches[0], form.EffectiveDate)),
			}},
		}
	}
	if len(nameDateMatches) > 1 {
		return ambiguousDecision("legal_name_and_effective_date", nameDateMatches)
	}
	if legalNamePresent(form) {
		return InformedK12MatchDecision{Status: InformedK12MatchNeedsReview, Conflicts: []string{"legal name matched no unique candidate with the same start or change date"}}
	}
	return InformedK12MatchDecision{Status: InformedK12MatchNoCandidate, Conflicts: []string{"no employee id, email, or legal-name/date rule matched"}}
}

// ReviewInformedK12Attachment records the policy-level manual review result for a
// form that could not be attached automatically. Callers must persist the
// returned attachment with their own database/audit transaction once the DB
// write path is implemented.
func ReviewInformedK12Attachment(form InformedK12FormRecord, person InformedK12PersonCandidate, evidenceUse InformedK12EvidenceUse, reviewerID string, reviewedAt time.Time, note string) InformedK12Attachment {
	return InformedK12Attachment{
		Form:         form,
		PeopleUUID:   person.PeopleUUID,
		State:        InformedK12AttachmentStateAttached,
		EvidenceUse:  evidenceUse,
		DecisionNote: note,
		ReviewedBy:   reviewerID,
		ReviewedAt:   reviewedAt,
	}
}

// DetachInformedK12Attachment preserves the source form while removing it from
// active evidence. This supports operator cleanup when a form was attached to
// the wrong person or a later form supersedes its decision evidence.
func DetachInformedK12Attachment(attachment InformedK12Attachment, actorID string, detachedAt time.Time, reason string) InformedK12Attachment {
	attachment.State = InformedK12AttachmentStateDetached
	attachment.DetachedBy = actorID
	attachment.DetachedAt = detachedAt
	attachment.DetachReason = reason
	return attachment
}

func SupersedeInformedK12Attachment(attachment InformedK12Attachment, supersedingFormID string, actorID string, supersededAt time.Time, reason string) InformedK12Attachment {
	attachment.State = InformedK12AttachmentStateSuperseded
	attachment.SupersededBy = supersedingFormID
	attachment.DetachedBy = actorID
	attachment.DetachedAt = supersededAt
	attachment.DetachReason = reason
	return attachment
}

// RedactInformedK12AttachmentForPersona enforces the field-level visibility rule
// for employee and contractor records. IT and HR can inspect retained personnel
// excerpts; site-scoped operators see only non-sensitive summaries for their
// site; all other personas receive no form metadata.
func RedactInformedK12AttachmentForPersona(attachment InformedK12Attachment, ctx InformedK12VisibilityContext) InformedK12VisibleAttachment {
	level := informedK12VisibilityLevel(ctx)
	visible := InformedK12VisibleAttachment{
		SourceFormID: attachment.Form.SourceFormID,
		FormType:     attachment.Form.FormType,
		SubmittedAt:  attachment.Form.SubmittedAt,
		Status:       attachment.Form.Status,
		EvidenceUse:  attachment.EvidenceUse,
		State:        attachment.State,
		Visibility:   level,
	}
	if level == InformedK12VisibilityHidden {
		return InformedK12VisibleAttachment{Visibility: InformedK12VisibilityHidden}
	}
	if level == InformedK12VisibilityFull {
		visible.SourceURL = attachment.Form.SourceURL
		visible.VisibleFields = append(visible.VisibleFields, attachment.Form.SourceFields...)
		return visible
	}
	for _, field := range attachment.Form.SourceFields {
		if field.Sensitivity == InformedK12FieldPublic {
			visible.VisibleFields = append(visible.VisibleFields, field)
			continue
		}
		visible.RedactedFields = append(visible.RedactedFields, field.Key)
	}
	sort.Strings(visible.RedactedFields)
	return visible
}

func informedK12VisibilityLevel(ctx InformedK12VisibilityContext) InformedK12VisibilityLevel {
	switch ctx.PersonaRole {
	case PermissionRoleITAdmin, PermissionRoleHumanResources:
		return InformedK12VisibilityFull
	case PermissionRoleSiteAdmin, PermissionRoleSiteSecretary:
		if containsString(ctx.VisibleSiteIDs, ctx.AttachmentSite) {
			return InformedK12VisibilitySummary
		}
	}
	return InformedK12VisibilityHidden
}

func uniqueStrongMatch(form InformedK12FormRecord, candidates []InformedK12PersonCandidate, rule string, sourceValue string, values func(InformedK12PersonCandidate) []string) InformedK12MatchDecision {
	source := normalizeMatchValue(sourceValue)
	if source == "" {
		return InformedK12MatchDecision{}
	}
	matches := make([]InformedK12PersonCandidate, 0, 1)
	evidence := make([]InformedK12MatchEvidence, 0, 1)
	for _, candidate := range candidates {
		for _, value := range values(candidate) {
			if normalizeMatchValue(value) == source {
				matches = append(matches, candidate)
				evidence = append(evidence, InformedK12MatchEvidence{Rule: rule, PersonUUID: candidate.PeopleUUID, SourceValue: sourceValue, PersonValue: value})
				break
			}
		}
	}
	if len(matches) == 1 {
		return InformedK12MatchDecision{Status: InformedK12MatchExact, PersonUUID: matches[0].PeopleUUID, Evidence: evidence}
	}
	if len(matches) > 1 {
		return InformedK12MatchDecision{Status: InformedK12MatchAmbiguous, Evidence: evidence, Conflicts: []string{rule + " matched multiple candidate records"}}
	}
	return InformedK12MatchDecision{}
}

func ambiguousDecision(rule string, matches []InformedK12PersonCandidate) InformedK12MatchDecision {
	evidence := make([]InformedK12MatchEvidence, 0, len(matches))
	for _, candidate := range matches {
		evidence = append(evidence, InformedK12MatchEvidence{Rule: rule, PersonUUID: candidate.PeopleUUID})
	}
	return InformedK12MatchDecision{Status: InformedK12MatchAmbiguous, Evidence: evidence, Conflicts: []string{rule + " matched multiple candidate records"}}
}

func nameAndDateMatches(form InformedK12FormRecord, candidates []InformedK12PersonCandidate) []InformedK12PersonCandidate {
	if !legalNamePresent(form) || form.EffectiveDate.IsZero() {
		return nil
	}
	matches := []InformedK12PersonCandidate{}
	for _, candidate := range candidates {
		if normalizeMatchValue(candidate.LegalFirstName) != normalizeMatchValue(form.LegalFirstName) {
			continue
		}
		if normalizeMatchValue(candidate.LegalLastName) != normalizeMatchValue(form.LegalLastName) {
			continue
		}
		if dateMatches(candidate.StartDate, form.EffectiveDate) || anyDateMatches(candidate.ChangeDates, form.EffectiveDate) {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func candidateEmployeeIDs(candidate InformedK12PersonCandidate) []string {
	return []string{candidate.EmployeeID, candidate.GeneratedEmployeeID}
}

func candidateEmails(candidate InformedK12PersonCandidate) []string {
	return append([]string{candidate.PrimaryEmail}, candidate.AlternateEmails...)
}

func normalizeMatchValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func legalNamePresent(form InformedK12FormRecord) bool {
	return normalizeMatchValue(form.LegalFirstName) != "" && normalizeMatchValue(form.LegalLastName) != ""
}

func anyDateMatches(values []time.Time, target time.Time) bool {
	for _, value := range values {
		if dateMatches(value, target) {
			return true
		}
	}
	return false
}

func dateMatches(left time.Time, right time.Time) bool {
	if left.IsZero() || right.IsZero() {
		return false
	}
	ly, lm, ld := left.Date()
	ry, rm, rd := right.Date()
	return ly == ry && lm == rm && ld == rd
}

func candidateDateEvidence(candidate InformedK12PersonCandidate, target time.Time) string {
	if dateMatches(candidate.StartDate, target) {
		return formatDate(candidate.StartDate)
	}
	for _, value := range candidate.ChangeDates {
		if dateMatches(value, target) {
			return formatDate(value)
		}
	}
	return ""
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
