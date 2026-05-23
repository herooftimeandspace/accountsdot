package core

import (
	"sort"
	"strings"
	"time"
	"unicode"
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

type InformedK12SiteSignalStatus string

const (
	InformedK12SiteSignalClear       InformedK12SiteSignalStatus = "clear"
	InformedK12SiteSignalMissing     InformedK12SiteSignalStatus = "missing"
	InformedK12SiteSignalAmbiguous   InformedK12SiteSignalStatus = "ambiguous"
	InformedK12SiteSignalStale       InformedK12SiteSignalStatus = "stale"
	InformedK12SiteSignalConflicting InformedK12SiteSignalStatus = "conflicting"
)

type InformedK12SiteSignalConfidence string

const (
	InformedK12SiteSignalConfidenceHigh   InformedK12SiteSignalConfidence = "high"
	InformedK12SiteSignalConfidenceMedium InformedK12SiteSignalConfidence = "medium"
	InformedK12SiteSignalConfidenceLow    InformedK12SiteSignalConfidence = "low"
	InformedK12SiteSignalConfidenceNone   InformedK12SiteSignalConfidence = "none"
)

type InformedK12SiteSignalFieldType string

const (
	InformedK12SiteSignalFieldSite       InformedK12SiteSignalFieldType = "site"
	InformedK12SiteSignalFieldLocation   InformedK12SiteSignalFieldType = "location"
	InformedK12SiteSignalFieldDepartment InformedK12SiteSignalFieldType = "department"
	InformedK12SiteSignalFieldPosition   InformedK12SiteSignalFieldType = "position"
	InformedK12SiteSignalFieldTransfer   InformedK12SiteSignalFieldType = "transfer"
	InformedK12SiteSignalFieldSupervisor InformedK12SiteSignalFieldType = "supervisor"
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

type InformedK12SiteAlias struct {
	RawValue string
	SiteID   string
	SiteName string
}

type InformedK12EscapeSiteSnapshot struct {
	SiteID   string
	SiteName string
}

type InformedK12SiteSignalFieldRef struct {
	Key         string
	Label       string
	FieldType   InformedK12SiteSignalFieldType
	RawValue    string
	Sensitivity InformedK12FieldSensitivity
}

type InformedK12SiteSignalInput struct {
	Attachment  InformedK12Attachment
	SiteAliases []InformedK12SiteAlias
	EscapeSite  InformedK12EscapeSiteSnapshot
	Now         time.Time
	StaleAfter  time.Duration
}

type InformedK12SiteChangeSignal struct {
	Status          InformedK12SiteSignalStatus
	Confidence      InformedK12SiteSignalConfidence
	SourceFormID    string
	FormType        string
	SubmittedAt     time.Time
	AttachmentState InformedK12AttachmentState
	EvidenceUse     InformedK12EvidenceUse
	ParsedSiteID    string
	ParsedSiteName  string
	RawSiteValues   []string
	FieldRefs       []InformedK12SiteSignalFieldRef
	ReviewReasons   []string
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

// BuildInformedK12SiteChangeSignal derives the issue #252 review signal from an
// already-linked form attachment. It is called by future employee and contractor
// detail projections after #253 has associated source forms with people; the
// function only classifies retained form fields against caller-supplied site
// aliases and Escape context, so it performs no provider reads, writes, or local
// persistence.
func BuildInformedK12SiteChangeSignal(input InformedK12SiteSignalInput) InformedK12SiteChangeSignal {
	signal := InformedK12SiteChangeSignal{
		Status:          InformedK12SiteSignalMissing,
		Confidence:      InformedK12SiteSignalConfidenceNone,
		SourceFormID:    input.Attachment.Form.SourceFormID,
		FormType:        input.Attachment.Form.FormType,
		SubmittedAt:     input.Attachment.Form.SubmittedAt,
		AttachmentState: input.Attachment.State,
		EvidenceUse:     input.Attachment.EvidenceUse,
	}
	if !isActiveInformedK12Attachment(input.Attachment) {
		signal.ReviewReasons = append(signal.ReviewReasons, "attachment is not active evidence")
		return signal
	}

	refs := collectInformedK12SiteFieldRefs(input.Attachment.Form.SourceFields)
	signal.FieldRefs = refs
	signal.RawSiteValues = rawSiteValues(refs)
	if len(refs) == 0 {
		signal.ReviewReasons = append(signal.ReviewReasons, "no supported site-bearing InformedK12 fields were retained")
		return signal
	}

	matches := matchInformedK12SiteAliases(refs, input.SiteAliases)
	if len(matches) == 0 {
		signal.Status = InformedK12SiteSignalAmbiguous
		signal.Confidence = InformedK12SiteSignalConfidenceLow
		signal.ReviewReasons = append(signal.ReviewReasons, "supported fields were present but no documented site alias matched their raw values")
		return signal
	}
	if len(matches) > 1 {
		signal.Status = InformedK12SiteSignalAmbiguous
		signal.Confidence = InformedK12SiteSignalConfidenceLow
		signal.ReviewReasons = append(signal.ReviewReasons, "supported fields matched multiple different site aliases")
		return signal
	}

	signal.ParsedSiteID = matches[0].SiteID
	signal.ParsedSiteName = matches[0].SiteName
	signal.Status = InformedK12SiteSignalClear
	signal.Confidence = siteSignalConfidence(refs)
	if input.StaleAfter > 0 && !input.Now.IsZero() && !input.Attachment.Form.SubmittedAt.IsZero() && input.Attachment.Form.SubmittedAt.Add(input.StaleAfter).Before(input.Now) {
		signal.Status = InformedK12SiteSignalStale
		signal.Confidence = InformedK12SiteSignalConfidenceLow
		signal.ReviewReasons = append(signal.ReviewReasons, "source form is older than the configured InformedK12 freshness window")
		return signal
	}
	if input.EscapeSite.SiteID != "" && matches[0].SiteID != "" && normalizeMatchValue(input.EscapeSite.SiteID) != normalizeMatchValue(matches[0].SiteID) {
		signal.Status = InformedK12SiteSignalConflicting
		signal.Confidence = InformedK12SiteSignalConfidenceMedium
		signal.ReviewReasons = append(signal.ReviewReasons, "InformedK12 site signal conflicts with current Escape site data")
	}
	return signal
}

// LatestInformedK12SiteChangeSignal selects the newest active attachment signal
// for employee and contractor detail projections. Detached and superseded forms
// remain visible through attachment history, but they do not displace a newer or
// older active site-change signal on the person detail surface.
func LatestInformedK12SiteChangeSignal(attachments []InformedK12Attachment, aliases []InformedK12SiteAlias, escapeSite InformedK12EscapeSiteSnapshot, now time.Time, staleAfter time.Duration) InformedK12SiteChangeSignal {
	var latest InformedK12Attachment
	for _, attachment := range attachments {
		if !isActiveInformedK12Attachment(attachment) {
			continue
		}
		if latest.Form.SourceFormID == "" || attachment.Form.SubmittedAt.After(latest.Form.SubmittedAt) {
			latest = attachment
		}
	}
	if latest.Form.SourceFormID == "" {
		return InformedK12SiteChangeSignal{
			Status:        InformedK12SiteSignalMissing,
			Confidence:    InformedK12SiteSignalConfidenceNone,
			ReviewReasons: []string{"no active InformedK12 form attachment was available"},
		}
	}
	return BuildInformedK12SiteChangeSignal(InformedK12SiteSignalInput{
		Attachment:  latest,
		SiteAliases: aliases,
		EscapeSite:  escapeSite,
		Now:         now,
		StaleAfter:  staleAfter,
	})
}

func isActiveInformedK12Attachment(attachment InformedK12Attachment) bool {
	return attachment.State == InformedK12AttachmentStateAttached
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

// collectInformedK12SiteFieldRefs filters retained form excerpts down to the
// site-bearing fields that #252 allows employee and contractor detail surfaces
// to evaluate. It returns exact raw values and sensitivity labels so callers can
// display or redact the source evidence without reparsing the full form.
func collectInformedK12SiteFieldRefs(fields []InformedK12SourceField) []InformedK12SiteSignalFieldRef {
	refs := []InformedK12SiteSignalFieldRef{}
	for _, field := range fields {
		fieldType, ok := informedK12SiteFieldType(field)
		if !ok || strings.TrimSpace(field.Value) == "" {
			continue
		}
		refs = append(refs, InformedK12SiteSignalFieldRef{
			Key:         field.Key,
			Label:       field.Label,
			FieldType:   fieldType,
			RawValue:    field.Value,
			Sensitivity: field.Sensitivity,
		})
	}
	return refs
}

// informedK12SiteFieldType classifies supported site-bearing form fields by key
// and label only. It deliberately ignores notes and comments so free-text
// personnel narrative does not become a site signal by keyword accident.
func informedK12SiteFieldType(field InformedK12SourceField) (InformedK12SiteSignalFieldType, bool) {
	name := normalizeMatchValue(field.Key + " " + field.Label)
	if containsAny(name, "note", "notes", "comment", "comments", "remark", "remarks") {
		return "", false
	}
	switch {
	case containsAny(name, "site", "school", "campus"):
		return InformedK12SiteSignalFieldSite, true
	case containsAny(name, "location", "building"):
		return InformedK12SiteSignalFieldLocation, true
	case containsAny(name, "department", "dept"):
		return InformedK12SiteSignalFieldDepartment, true
	case containsAny(name, "position", "job title", "classification"):
		return InformedK12SiteSignalFieldPosition, true
	case containsAny(name, "transfer", "new assignment", "assignment change"):
		return InformedK12SiteSignalFieldTransfer, true
	case containsAny(name, "supervisor", "principal", "manager"):
		return InformedK12SiteSignalFieldSupervisor, true
	}
	return "", false
}

// rawSiteValues returns de-duplicated source values for review surfaces while
// preserving the first retained spelling, punctuation, and capitalization from
// InformedK12.
func rawSiteValues(refs []InformedK12SiteSignalFieldRef) []string {
	values := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		normalized := normalizeMatchValue(ref.RawValue)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		values = append(values, ref.RawValue)
	}
	return values
}

// matchInformedK12SiteAliases compares retained raw field values with
// documented site aliases supplied by the caller. It returns unique parsed site
// targets and leaves no-match or multi-match cases for the public builder to
// classify as review states.
func matchInformedK12SiteAliases(refs []InformedK12SiteSignalFieldRef, aliases []InformedK12SiteAlias) []InformedK12SiteAlias {
	matchesBySite := map[string]InformedK12SiteAlias{}
	for _, ref := range refs {
		for _, alias := range aliases {
			if normalizeMatchValue(ref.RawValue) != normalizeMatchValue(alias.RawValue) {
				continue
			}
			key := normalizeMatchValue(firstNonEmpty(alias.SiteID, alias.SiteName))
			if key == "" {
				continue
			}
			matchesBySite[key] = alias
		}
	}
	matches := make([]InformedK12SiteAlias, 0, len(matchesBySite))
	for _, match := range matchesBySite {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool {
		return firstNonEmpty(matches[i].SiteID, matches[i].SiteName) < firstNonEmpty(matches[j].SiteID, matches[j].SiteName)
	})
	return matches
}

// siteSignalConfidence ranks direct site and location fields above contextual
// position, department, transfer, or supervisor fields so the signal can explain
// why one parsed site is stronger evidence than another reviewed source.
func siteSignalConfidence(refs []InformedK12SiteSignalFieldRef) InformedK12SiteSignalConfidence {
	for _, ref := range refs {
		if ref.FieldType == InformedK12SiteSignalFieldSite || ref.FieldType == InformedK12SiteSignalFieldLocation {
			return InformedK12SiteSignalConfidenceHigh
		}
	}
	return InformedK12SiteSignalConfidenceMedium
}

// containsAny keeps the field classifier compact and deterministic; it performs
// no normalization because callers pass values that already went through
// normalizeMatchValue.
func containsAny(value string, needles ...string) bool {
	tokens := normalizedTermTokens(value)
	for _, needle := range needles {
		if containsNormalizedTerms(tokens, normalizedTermTokens(needle)) {
			return true
		}
	}
	return false
}

func containsNormalizedTerms(tokens []string, phrase []string) bool {
	if len(phrase) == 0 || len(phrase) > len(tokens) {
		return false
	}
	for i := 0; i <= len(tokens)-len(phrase); i++ {
		matched := true
		for j, term := range phrase {
			if tokens[i+j] != term {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func normalizedTermTokens(value string) []string {
	return strings.FieldsFunc(normalizeMatchValue(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
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
