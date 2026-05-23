package web

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

const (
	onboardingManualDraftStatusIncomplete                  = "Incomplete Data"
	onboardingManualDraftStatusReady                       = "Ready to Provision"
	onboardingManualDraftStatusInvalid                     = "Invalid"
	onboardingManualDraftStatusNeedsReview                 = "Needs Review"
	onboardingManualDraftTTL                               = 30 * 24 * time.Hour
	onboardingAdjacentConversionGapDays                    = 14
	onboardingValidityStateValid                           = "valid"
	onboardingValidityStateInvalid                         = "invalid"
	onboardingValidityStateReviewRequired                  = "review_required"
	onboardingInvalidReasonActiveEscapeContractorCollision = "active_escape_contractor_collision"
	onboardingInvalidReasonContinuationMismatch            = "continuation_identity_mismatch"
	onboardingInvalidReasonContinuationUnknownEndDate      = "continuation_unknown_end_date"
	onboardingInvalidReasonContinuationDateOverlap         = "continuation_date_overlap"
	onboardingInvalidReasonContinuationMissingIdentity     = "continuation_missing_identity"
)

var (
	onboardingPersonalEmailPattern  = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	onboardingLast4Pattern          = regexp.MustCompile(`^\d{4}$`)
	onboardingPersonalPhonePattern  = regexp.MustCompile(`^\d{10}$`)
	onboardingFormattedPhonePattern = regexp.MustCompile(`^\(\d{3}\) \d{3}-\d{4}$`)
	devOnboardingStore              = newDevOnboardingStore()
)

type onboardingPagePayload struct {
	PageID      string                    `json:"page_id"`
	Persona     devPersona                `json:"persona"`
	Shell       devShellPayload           `json:"shell"`
	GeneratedAt string                    `json:"generated_at"`
	Page        onboardingPageContent     `json:"page"`
	Form        onboardingFormOptions     `json:"form"`
	Hotspots    map[string]hotspotPayload `json:"hotspots"`
}

type onboardingPageContent struct {
	Title                 string                         `json:"title"`
	Description           string                         `json:"description"`
	LastRefreshed         string                         `json:"last_refreshed"`
	CurrentDate           string                         `json:"current_date"`
	CanManageManual       bool                           `json:"can_manage_manual"`
	Rows                  []onboardingRowPayload         `json:"rows"`
	Drafts                []onboardingManualDraftPayload `json:"drafts"`
	ManualDraftRetention  string                         `json:"manual_draft_retention"`
	ManualAutosaveSeconds int                            `json:"manual_autosave_seconds"`
}

type onboardingRowPayload struct {
	ID                   string                               `json:"id"`
	Kind                 string                               `json:"kind"`
	DateAdded            string                               `json:"date_added"`
	DateAddedReason      string                               `json:"date_added_reason"`
	StartDate            string                               `json:"start_date"`
	LeadTimeWarning      bool                                 `json:"lead_time_warning,omitempty"`
	EffectiveDate        string                               `json:"effective_date,omitempty"`
	Person               string                               `json:"person"`
	SiteID               string                               `json:"site_id"`
	Site                 string                               `json:"site"`
	RoomID               string                               `json:"room_id,omitempty"`
	RoomName             string                               `json:"room_name,omitempty"`
	CanUpdateRoom        bool                                 `json:"can_update_room,omitempty"`
	CurrentStep          string                               `json:"current_step"`
	IssueAction          string                               `json:"issue_action"`
	WorkflowStatus       string                               `json:"workflow_status"`
	ChangeReason         string                               `json:"change_reason,omitempty"`
	LateStart            bool                                 `json:"late_start,omitempty"`
	ScheduledFor         string                               `json:"scheduled_for,omitempty"`
	ValidityState        string                               `json:"validity_state,omitempty"`
	InvalidReason        string                               `json:"invalid_reason,omitempty"`
	LinkedEscapeRecord   *onboardingLinkedEscapeRecordPayload `json:"linked_escape_record,omitempty"`
	ConversionDecision   *onboardingConversionDecisionPayload `json:"conversion_decision,omitempty"`
	EmployeeTimeline     []onboardingTimelineEntryPayload     `json:"employee_timeline,omitempty"`
	AuditEvents          []onboardingAuditEventPayload        `json:"audit_events,omitempty"`
	CanDeleteManualEntry bool                                 `json:"can_delete_manual_entry,omitempty"`
	AssignedEmail        string                               `json:"assigned_email,omitempty"`
	EmployeeNumber       string                               `json:"employee_number,omitempty"`
	ManualDraftID        string                               `json:"manual_draft_id,omitempty"`
	IncidentIQ           string                               `json:"incident_iq,omitempty"`
	AeriesTicket         string                               `json:"aeries_ticket,omitempty"`
	VerkadaTicket        string                               `json:"verkada_ticket,omitempty"`
	WorkflowSteps        []onboardingWorkflowStep             `json:"workflow_steps,omitempty"`
}

type onboardingWorkflowStep struct {
	Name    string                     `json:"name"`
	Status  string                     `json:"status"`
	Detail  string                     `json:"detail"`
	Actions []onboardingWorkflowAction `json:"actions,omitempty"`
}

type onboardingWorkflowAction struct {
	Label      string `json:"label"`
	Resolution string `json:"resolution"`
	System     string `json:"system"`
	Href       string `json:"href"`
}

type onboardingLinkedEscapeRecordPayload struct {
	ID             string `json:"id"`
	Person         string `json:"person"`
	Site           string `json:"site"`
	AssignedEmail  string `json:"assigned_email,omitempty"`
	EmployeeNumber string `json:"employee_number,omitempty"`
	StartDate      string `json:"start_date,omitempty"`
	EndDate        string `json:"end_date,omitempty"`
	CurrentStep    string `json:"current_step,omitempty"`
	WorkflowStatus string `json:"workflow_status,omitempty"`
}

type onboardingConversionDecisionPayload struct {
	Decision           string `json:"decision"`
	Reason             string `json:"reason"`
	GapDays            int    `json:"gap_days"`
	AdjacentGapDays    int    `json:"adjacent_gap_days"`
	GapWarning         bool   `json:"gap_warning"`
	AccountEligible    bool   `json:"account_eligible"`
	LifecycleOwner     string `json:"lifecycle_owner"`
	ActorID            string `json:"actor_id,omitempty"`
	DecidedAt          string `json:"decided_at,omitempty"`
	LinkedEmployeeID   string `json:"linked_employee_id,omitempty"`
	LinkedContractorID string `json:"linked_contractor_id,omitempty"`
}

type onboardingTimelineEntryPayload struct {
	Kind           string `json:"kind"`
	Label          string `json:"label"`
	Source         string `json:"source"`
	Owner          string `json:"owner"`
	StartDate      string `json:"start_date,omitempty"`
	EndDate        string `json:"end_date,omitempty"`
	RecordID       string `json:"record_id,omitempty"`
	LinkedRecordID string `json:"linked_record_id,omitempty"`
	Warning        string `json:"warning,omitempty"`
}

type onboardingAuditEventPayload struct {
	EventType  string `json:"event_type"`
	ActorID    string `json:"actor_id"`
	OccurredAt string `json:"occurred_at"`
	Summary    string `json:"summary"`
}

type onboardingFormOptions struct {
	EmployeeTypes         []string                   `json:"employee_types"`
	Classifications       []string                   `json:"classifications"`
	JobTitles             []string                   `json:"job_titles"`
	Sites                 []devSiteContext           `json:"sites"`
	PreferredDevices      []string                   `json:"preferred_devices"`
	RequestedAeriesAccess []string                   `json:"requested_aeries_access"`
	ReplacingEmployees    []onboardingEmployeeOption `json:"replacing_employees"`
	ContinuationEmployees []onboardingEmployeeOption `json:"continuation_employees"`
	Rooms                 []onboardingRoomOption     `json:"rooms"`
}

type onboardingEmployeeOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type onboardingRoomOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SiteID   string `json:"site_id"`
	SiteName string `json:"site_name"`
}

type onboardingManualDraftRequest struct {
	StartDate              string `json:"start_date"`
	SSNLast4               string `json:"ssn_last4"`
	EmployeeType           string `json:"employee_type"`
	Classification         string `json:"classification"`
	FirstName              string `json:"first_name"`
	LastName               string `json:"last_name"`
	JobTitle               string `json:"job_title"`
	SiteID                 string `json:"site_id"`
	PersonalEmail          string `json:"personal_email"`
	PersonalPhone          string `json:"personal_phone"`
	PreferredDevice        string `json:"preferred_device"`
	RequestedAeriesAccess  string `json:"requested_aeries_access"`
	ReplacingEmployeeID    string `json:"replacing_employee_id"`
	ContinuationEmployeeID string `json:"continuation_employee_id"`
	RoomID                 string `json:"room_id"`
	Notes                  string `json:"notes"`
}

type onboardingRoomUpdateRequest struct {
	RoomID string `json:"room_id"`
}

type onboardingManualDraftPayload struct {
	ID                        string                               `json:"id"`
	Status                    string                               `json:"status"`
	StartDate                 string                               `json:"start_date"`
	EffectiveDate             string                               `json:"effective_date,omitempty"`
	SSNLast4                  string                               `json:"ssn_last4,omitempty"`
	EmployeeType              string                               `json:"employee_type"`
	Classification            string                               `json:"classification"`
	FirstName                 string                               `json:"first_name"`
	LastName                  string                               `json:"last_name"`
	JobTitle                  string                               `json:"job_title"`
	SiteID                    string                               `json:"site_id"`
	SiteName                  string                               `json:"site_name"`
	PersonalEmail             string                               `json:"personal_email"`
	PersonalPhone             string                               `json:"personal_phone,omitempty"`
	PreferredDevice           string                               `json:"preferred_device"`
	RequestedAeriesAccess     string                               `json:"requested_aeries_access"`
	ReplacingEmployeeID       string                               `json:"replacing_employee_id,omitempty"`
	ReplacingEmployeeName     string                               `json:"replacing_employee_name,omitempty"`
	ReplacingEmployeeEmail    string                               `json:"replacing_employee_email,omitempty"`
	ContinuationEmployeeID    string                               `json:"continuation_employee_id,omitempty"`
	ContinuationEmployeeName  string                               `json:"continuation_employee_name,omitempty"`
	ContinuationEmployeeEmail string                               `json:"continuation_employee_email,omitempty"`
	RoomID                    string                               `json:"room_id,omitempty"`
	RoomName                  string                               `json:"room_name,omitempty"`
	Notes                     string                               `json:"notes,omitempty"`
	GeneratedEmail            string                               `json:"generated_email,omitempty"`
	GeneratedEmployeeID       string                               `json:"generated_employee_id,omitempty"`
	ChangeReason              string                               `json:"change_reason,omitempty"`
	LateStart                 bool                                 `json:"late_start,omitempty"`
	ScheduledFor              string                               `json:"scheduled_for,omitempty"`
	ValidityState             string                               `json:"validity_state,omitempty"`
	InvalidReason             string                               `json:"invalid_reason,omitempty"`
	LinkedEscapeRecord        *onboardingLinkedEscapeRecordPayload `json:"linked_escape_record,omitempty"`
	ConversionDecision        *onboardingConversionDecisionPayload `json:"conversion_decision,omitempty"`
	EmployeeTimeline          []onboardingTimelineEntryPayload     `json:"employee_timeline,omitempty"`
	AuditEvents               []onboardingAuditEventPayload        `json:"audit_events,omitempty"`
	CanDeleteManualEntry      bool                                 `json:"can_delete_manual_entry,omitempty"`
	MissingFields             []string                             `json:"missing_fields"`
	CreatedAt                 string                               `json:"created_at"`
	UpdatedAt                 string                               `json:"updated_at"`
	FinalizedAt               string                               `json:"finalized_at,omitempty"`
}

type onboardingManualDraftResponse struct {
	Draft onboardingManualDraftPayload `json:"draft"`
	Rows  []onboardingRowPayload       `json:"rows,omitempty"`
}

type devOnboardingStoreState struct {
	mu            sync.Mutex
	nextDraft     int
	nextEmployee  int
	drafts        map[string]*onboardingManualDraft
	roomOverrides map[string]string
}

type onboardingManualDraft struct {
	ID                     string
	Status                 string
	StartDate              string
	SSNLast4               string
	EmployeeType           string
	Classification         string
	FirstName              string
	LastName               string
	JobTitle               string
	SiteID                 string
	PersonalEmail          string
	PersonalPhone          string
	PreferredDevice        string
	RequestedAeriesAccess  string
	ReplacingEmployeeID    string
	ContinuationEmployeeID string
	RoomID                 string
	Notes                  string
	GeneratedEmail         string
	GeneratedEmployeeID    string
	ChangeReason           core.WorkflowChangeReason
	ValidityState          string
	InvalidReason          string
	LinkedEscapePersonID   string
	ContinuationActorID    string
	ContinuationMarkedAt   *time.Time
	AuditEvents            []onboardingAuditEventPayload
	ScheduledFor           *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	FinalizedAt            *time.Time
	DeletedAt              *time.Time
	DeletedBy              string
}

type devEscapeEmploymentRecord struct {
	ID             string
	FirstName      string
	LastName       string
	SiteID         string
	SiteName       string
	AssignedEmail  string
	EmployeeNumber string
	StartDate      string
	EndDate        string
	CurrentStep    string
	WorkflowStatus string
	Active         bool
}

var devEscapeEmploymentRecords = []devEscapeEmploymentRecord{
	{ID: "escape-jordan-miles", FirstName: "Jordan", LastName: "Miles", SiteID: "clover-hs", SiteName: "Clover HS", AssignedEmail: "jordan.miles@wusd.org", EmployeeNumber: "100241", StartDate: "2025-05-06", CurrentStep: "Google pending", WorkflowStatus: "In Progress", Active: true},
	{ID: "escape-nia-brooks", FirstName: "Nia", LastName: "Brooks", SiteID: "district-office", SiteName: "District Office", AssignedEmail: "nia.brooks@wusd.org", EmployeeNumber: "100842", StartDate: "2025-05-08", CurrentStep: "Sync dry-run", WorkflowStatus: "Needs Review", Active: true},
	{ID: "escape-evan-ruiz", FirstName: "Evan", LastName: "Ruiz", SiteID: "franklin-ms", SiteName: "Franklin MS", AssignedEmail: "evan.ruiz@wusd.org", EmployeeNumber: "101106", StartDate: "2025-05-12", CurrentStep: "HR intake", WorkflowStatus: "Blocked", Active: true},
	{ID: "escape-mika-ito", FirstName: "Mika", LastName: "Ito", SiteID: "desert-view", SiteName: "Desert View", AssignedEmail: "mika.ito@wusd.org", EmployeeNumber: "101441", StartDate: "2025-05-13", CurrentStep: "Ready", WorkflowStatus: "Ready", Active: true},
	{ID: "escape-harper-sloan", FirstName: "Harper", LastName: "Sloan", SiteID: "business-office", SiteName: "Business Office", AssignedEmail: "harper.sloan@wusd.org", EmployeeNumber: "104812", StartDate: "2024-08-12", EndDate: "2026-04-30", CurrentStep: "Inactive in Escape", WorkflowStatus: "Inactive", Active: false},
	{ID: "escape-riley-morgan", FirstName: "Riley", LastName: "Morgan", SiteID: "business-office", SiteName: "Business Office", AssignedEmail: "riley.morgan@wusd.org", EmployeeNumber: "104923", StartDate: "2024-08-12", CurrentStep: "Inactive in Escape", WorkflowStatus: "Inactive", Active: false},
	{ID: "escape-cameron-lee", FirstName: "Cameron", LastName: "Lee", SiteID: "business-office", SiteName: "Business Office", StartDate: "2024-08-12", EndDate: "2026-04-30", CurrentStep: "Inactive in Escape", WorkflowStatus: "Inactive", Active: false},
}

func newDevOnboardingStore() *devOnboardingStoreState {
	return &devOnboardingStoreState{
		nextDraft:     1,
		nextEmployee:  1,
		drafts:        map[string]*onboardingManualDraft{},
		roomOverrides: map[string]string{},
	}
}

// ResetDevOnboardingStateForTest restores the package-level DEV onboarding
// store so web tests can assert site scoping and drawer mutations without
// inheriting manual drafts or room overrides from earlier subtests.
func ResetDevOnboardingStateForTest() {
	devOnboardingStore = newDevOnboardingStore()
}

func findEscapeEmploymentRecord(firstName string, lastName string) *devEscapeEmploymentRecord {
	normalizedFirst := strings.ToLower(normalizeSpaces(firstName))
	normalizedLast := strings.ToLower(normalizeSpaces(lastName))
	if normalizedFirst == "" || normalizedLast == "" {
		return nil
	}
	for index := range devEscapeEmploymentRecords {
		record := &devEscapeEmploymentRecords[index]
		if strings.ToLower(record.FirstName) == normalizedFirst && strings.ToLower(record.LastName) == normalizedLast {
			return record
		}
	}
	return nil
}

func linkedEscapePayloadByID(id string) *onboardingLinkedEscapeRecordPayload {
	for index := range devEscapeEmploymentRecords {
		record := &devEscapeEmploymentRecords[index]
		if record.ID != id {
			continue
		}
		return &onboardingLinkedEscapeRecordPayload{
			ID:             record.ID,
			Person:         strings.TrimSpace(record.FirstName + " " + record.LastName),
			Site:           record.SiteName,
			AssignedEmail:  record.AssignedEmail,
			EmployeeNumber: record.EmployeeNumber,
			StartDate:      record.StartDate,
			EndDate:        record.EndDate,
			CurrentStep:    record.CurrentStep,
			WorkflowStatus: record.WorkflowStatus,
		}
	}
	return nil
}

func inactiveEscapeRecordByID(id string) *devEscapeEmploymentRecord {
	for index := range devEscapeEmploymentRecords {
		record := &devEscapeEmploymentRecords[index]
		if record.ID == id && !record.Active {
			return record
		}
	}
	return nil
}

func sameEscapePerson(record *devEscapeEmploymentRecord, firstName string, lastName string) bool {
	if record == nil {
		return false
	}
	return strings.EqualFold(record.FirstName, normalizeSpaces(firstName)) &&
		strings.EqualFold(record.LastName, normalizeSpaces(lastName))
}

func onboardingTimeLocation() *time.Location {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.UTC
	}
	return location
}

func parseOnboardingStartDate(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, onboardingTimeLocation())
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func isLateStart(startDate string, now time.Time) bool {
	parsed, ok := parseOnboardingStartDate(startDate)
	if !ok {
		return false
	}
	current := now.In(onboardingTimeLocation())
	currentDate := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, onboardingTimeLocation())
	daysUntilStart := int(parsed.Sub(currentDate).Hours() / 24)
	return daysUntilStart >= 0 && daysUntilStart <= 3
}

func nextAvailableWorkflowCycle(now time.Time) time.Time {
	cadence := 30 * time.Second
	next := now.UTC().Truncate(cadence).Add(cadence)
	if !next.After(now.UTC()) {
		next = next.Add(cadence)
	}
	return next
}

func formatOnboardingDateTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.In(onboardingTimeLocation()).Format("Jan 2, 2006 3:04 PM MST")
}

func handleDevOnboardingPage(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can view this page.",
		})
		return
	}
	if !routeAllowed(r.Context(), config, "/onboarding") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Onboarding is not available for this role.",
			"persona": config.Persona,
		})
		return
	}
	config = onboardingConfigWithRequestedSite(config, strings.TrimSpace(r.URL.Query().Get("site_id")))

	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, onboardingPagePayload{
		PageID:      "onboarding",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: onboardingPageContent{
			Title:                 "Onboarding Dashboard",
			Description:           "Upcoming onboarding processes by person, with blockers, workflow state, and external IIQ follow-up status.",
			LastRefreshed:         "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			CurrentDate:           now.Format("2006-01-02"),
			CanManageManual:       canManageManualOnboarding(config),
			Rows:                  devOnboardingStore.rows(now, config),
			Drafts:                devOnboardingStore.draftPayloads(now, config),
			ManualDraftRetention:  "30 days",
			ManualAutosaveSeconds: 60,
		},
		Form:     devOnboardingFormOptions(config),
		Hotspots: map[string]hotspotPayload{"add_manual": {NodeID: "f109", Label: "Add Non-Escape Record"}},
	})
}

// handleDevOnboardingRoomUpdate applies the DEV-only onboarding drawer room
// override. Site-scoped personas reach this through /onboarding row drawers and
// may submit only room_id for rows in the effective site scope; HR and IT keep
// their broader documented onboarding visibility while sharing the same mock
// room-write boundary.
func handleDevOnboardingRoomUpdate(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can update onboarding rooms.",
		})
		return
	}
	if !routeAllowed(r.Context(), config, "/onboarding") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Onboarding is not available for this role.",
			"persona": config.Persona,
		})
		return
	}
	config = onboardingConfigWithRequestedSite(config, strings.TrimSpace(r.URL.Query().Get("site_id")))

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dev/onboarding/rows/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "room" {
		http.NotFound(w, r)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_request",
			"message": "Onboarding room update request body is invalid.",
		})
		return
	}
	if isSiteScopedOnboardingPersona(config) {
		for key := range raw {
			if key != "room_id" {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"code":    "forbidden_field",
					"message": "Site Admin and Site Secretary can update only Room from the onboarding drawer.",
					"field":   key,
				})
				return
			}
		}
	}
	var request onboardingRoomUpdateRequest
	if value, ok := raw["room_id"]; ok {
		if err := json.Unmarshal(value, &request.RoomID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_request",
				"message": "Room must be submitted as a room id.",
			})
			return
		}
	}

	now := time.Now().UTC()
	row, status, errors := devOnboardingStore.updateRoom(parts[0], request, now, config)
	if status != http.StatusOK {
		code := "validation_failed"
		message := "Onboarding room update was rejected."
		if status == http.StatusNotFound {
			code = "not_found"
			message = "Onboarding row was not found in this persona's site scope."
		}
		writeJSON(w, status, map[string]any{
			"code":    code,
			"message": message,
			"errors":  errors,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"row":  row,
		"rows": devOnboardingStore.rows(now, config),
	})
}

func handleDevOnboardingManualDrafts(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	config, ok := requireManualOnboardingManager(w, r)
	if !ok {
		return
	}
	now := time.Now().UTC()
	draft := devOnboardingStore.create(now, config)
	writeJSON(w, http.StatusCreated, onboardingManualDraftResponse{Draft: draft.toPayload(now)})
}

func handleDevOnboardingManualDraft(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}
	config, ok := requireManualOnboardingManager(w, r)
	if !ok {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dev/onboarding/manual-drafts/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	draftID := parts[0]
	finalize := len(parts) == 2 && parts[1] == "finalize"
	if len(parts) > 2 || (len(parts) == 2 && !finalize) {
		http.NotFound(w, r)
		return
	}

	switch {
	case r.Method == http.MethodPut && !finalize:
		var request onboardingManualDraftRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_request",
				"message": "Manual onboarding draft request body is invalid.",
			})
			return
		}
		now := time.Now().UTC()
		draft, found, validationErrors := devOnboardingStore.update(draftID, request, now, config)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "not_found",
				"message": "Manual onboarding draft was not found.",
			})
			return
		}
		if len(validationErrors) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "validation_failed",
				"message": "Manual onboarding draft contains invalid fields.",
				"errors":  validationErrors,
				"draft":   draft.toPayload(now),
			})
			return
		}
		writeJSON(w, http.StatusOK, onboardingManualDraftResponse{Draft: draft.toPayload(now)})
	case r.Method == http.MethodPost && finalize:
		now := time.Now().UTC()
		draft, found, blocked := devOnboardingStore.finalize(draftID, now)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "not_found",
				"message": "Manual onboarding draft was not found.",
			})
			return
		}
		if blocked {
			code := "continuation_review_required"
			message := "Manual onboarding draft requires HR or IT review before workflow creation."
			switch draft.InvalidReason {
			case onboardingInvalidReasonActiveEscapeContractorCollision:
				code = "unsupported_overlap"
				message = "Active Escape employees cannot be hired as manual contractors. Delete the manual entry to resolve this collision."
			case onboardingInvalidReasonContinuationMismatch:
				code = "continuation_identity_mismatch"
				message = "The selected continuation link does not match this manual contractor. Correct or clear the continuation link before workflow creation."
			case onboardingInvalidReasonContinuationUnknownEndDate:
				code = "continuation_unknown_end_date"
				message = "The former employee end date is unavailable, so the employee-to-contractor gap cannot be verified before workflow creation."
			case onboardingInvalidReasonContinuationDateOverlap:
				code = "continuation_date_overlap"
				message = "The contractor start date must be after the former employee end date before workflow creation."
			case onboardingInvalidReasonContinuationMissingIdentity:
				code = "continuation_missing_identity"
				message = "The former employee record is missing the district email or employee number required for account continuity."
			}
			writeJSON(w, http.StatusConflict, map[string]any{
				"code":    code,
				"message": message,
				"draft":   draft.toPayload(now),
			})
			return
		}
		writeJSON(w, http.StatusOK, onboardingManualDraftResponse{
			Draft: draft.toPayload(now),
			Rows:  devOnboardingStore.rows(now, config),
		})
	case r.Method == http.MethodDelete && !finalize:
		now := time.Now().UTC()
		draft, found := devOnboardingStore.softDelete(draftID, config.Persona.ID, now)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "not_found",
				"message": "Manual onboarding draft was not found.",
			})
			return
		}
		writeJSON(w, http.StatusOK, onboardingManualDraftResponse{
			Draft: draft.toPayload(now),
			Rows:  devOnboardingStore.rows(now, config),
		})
	default:
		http.NotFound(w, r)
	}
}

func requireManualOnboardingManager(w http.ResponseWriter, r *http.Request) (devPersonaConfig, bool) {
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can manage manual onboarding records.",
		})
		return devPersonaConfig{}, false
	}
	if !canManageManualOnboarding(config) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Only Human Resources and IT Admin can manage manual onboarding records.",
			"persona": config.Persona,
		})
		return devPersonaConfig{}, false
	}
	if !routeAllowed(r.Context(), config, "/onboarding") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Onboarding is not available for this role.",
			"persona": config.Persona,
		})
		return devPersonaConfig{}, false
	}
	return config, true
}

func canManageManualOnboarding(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

func devOnboardingFormOptions(config devPersonaConfig) onboardingFormOptions {
	sites := config.VisibleSites
	if canManageManualOnboarding(config) {
		sites = sitesByID("district-office", "clover-hs", "desert-view", "highland-es", "franklin-ms", "business-office")
	}
	if isSiteScopedOnboardingPersona(config) {
		sites = []devSiteContext{onboardingEffectiveSite(config, "")}
	}
	rooms := []onboardingRoomOption{
		{ID: "iiq-room-cla-101", Name: "CLA Room 101", SiteID: "clover-hs", SiteName: "Clover High School"},
		{ID: "iiq-room-cla-108", Name: "CLA Room 108", SiteID: "clover-hs", SiteName: "Clover High School"},
		{ID: "iiq-room-dve-c118", Name: "DVE Room C118", SiteID: "desert-view", SiteName: "Desert View"},
		{ID: "iiq-room-do-hr", Name: "District Office HR", SiteID: "district-office", SiteName: "District Office"},
		{ID: "iiq-room-fms-a101", Name: "FMS Room A101", SiteID: "franklin-ms", SiteName: "Franklin Middle School"},
	}
	if isSiteScopedOnboardingPersona(config) {
		site := onboardingEffectiveSite(config, "")
		rooms = slices.DeleteFunc(rooms, func(room onboardingRoomOption) bool {
			return room.SiteID != site.ID
		})
	}
	return onboardingFormOptions{
		EmployeeTypes:         []string{"Contractor", "Volunteer", "Intern", "Student Teacher"},
		Classifications:       []string{"Certificated", "Classified", "Contractor", "Volunteer"},
		JobTitles:             []string{"Counselor", "Instructional Aide", "Program Instructional Assistant", "Student Teacher", "Substitute", "Yard Duty"},
		Sites:                 sites,
		PreferredDevices:      []string{"Mac", "Windows"},
		RequestedAeriesAccess: []string{"Teacher", "Staff", "Counselor", "Secretary", "Registrar"},
		ReplacingEmployees: []onboardingEmployeeOption{
			{ID: "person-clover-morgan-slate", Name: "Morgan Slate", Email: "morgan.slate@mock.wusd.invalid"},
			{ID: "person-clover-riley-vale", Name: "Riley Vale", Email: "riley.vale@mock.wusd.invalid"},
			{ID: "person-district-jules-rowan", Name: "Jules Rowan", Email: "jules.rowan@mock.wusd.invalid"},
		},
		ContinuationEmployees: []onboardingEmployeeOption{
			{ID: "escape-harper-sloan", Name: "Harper Sloan", Email: "harper.sloan@wusd.org"},
			{ID: "escape-riley-morgan", Name: "Riley Morgan", Email: "riley.morgan@wusd.org"},
			{ID: "escape-cameron-lee", Name: "Cameron Lee", Email: ""},
		},
		Rooms: rooms,
	}
}

func isSiteScopedOnboardingPersona(config devPersonaConfig) bool {
	return config.Persona.ID == "site_admin" || config.Persona.ID == "site_secretary"
}

func onboardingEffectiveSite(config devPersonaConfig, requestedSiteID string) devSiteContext {
	if requestedSiteID != "" {
		for _, site := range config.VisibleSites {
			if site.ID == requestedSiteID {
				return site
			}
		}
	}
	if config.CurrentSite.ID != "" {
		return config.CurrentSite
	}
	return config.DefaultSite
}

func onboardingConfigWithRequestedSite(config devPersonaConfig, requestedSiteID string) devPersonaConfig {
	if !isSiteScopedOnboardingPersona(config) {
		return config
	}
	site := onboardingEffectiveSite(config, requestedSiteID)
	config.CurrentSite = site
	return config
}

func onboardingVisibleForConfig(row onboardingRowPayload, config devPersonaConfig) bool {
	if !isSiteScopedOnboardingPersona(config) {
		return true
	}
	site := onboardingEffectiveSite(config, "")
	return row.SiteID == site.ID
}

func (s *devOnboardingStoreState) create(now time.Time, config devPersonaConfig) *onboardingManualDraft {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	id := "manual-draft-" + strconv.Itoa(s.nextDraft)
	s.nextDraft++
	siteID := config.CurrentSite.ID
	if canManageManualOnboarding(config) {
		siteID = "district-office"
	}
	draft := &onboardingManualDraft{
		ID:            id,
		Status:        onboardingManualDraftStatusIncomplete,
		SiteID:        siteID,
		ValidityState: onboardingValidityStateValid,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.drafts[id] = draft
	return cloneOnboardingDraft(draft)
}

func (s *devOnboardingStoreState) update(id string, request onboardingManualDraftRequest, now time.Time, config devPersonaConfig) (*onboardingManualDraft, bool, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	draft, ok := s.drafts[id]
	if !ok {
		return nil, false, nil
	}
	if draft.DeletedAt != nil {
		return nil, false, nil
	}
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonActiveEscapeContractorCollision {
		return cloneOnboardingDraft(draft), true, map[string]string{
			"form": "Active Escape contractor collisions cannot be edited. Delete the manual entry to resolve the collision.",
		}
	}
	if draft.FinalizedAt != nil {
		return cloneOnboardingDraft(draft), true, map[string]string{
			"form": "Finalized manual onboarding records are read-only in the DEV workflow.",
		}
	}

	clean, validationErrors := sanitizeManualDraftRequest(request, config)
	if len(validationErrors) > 0 {
		return cloneOnboardingDraft(draft), true, validationErrors
	}
	draft.StartDate = clean.StartDate
	draft.SSNLast4 = clean.SSNLast4
	draft.EmployeeType = clean.EmployeeType
	draft.Classification = clean.Classification
	draft.FirstName = clean.FirstName
	draft.LastName = clean.LastName
	draft.JobTitle = clean.JobTitle
	draft.SiteID = clean.SiteID
	draft.PersonalEmail = clean.PersonalEmail
	draft.PersonalPhone = clean.PersonalPhone
	draft.PreferredDevice = clean.PreferredDevice
	draft.RequestedAeriesAccess = clean.RequestedAeriesAccess
	draft.ReplacingEmployeeID = clean.ReplacingEmployeeID
	if draft.ContinuationEmployeeID != clean.ContinuationEmployeeID {
		if clean.ContinuationEmployeeID == "" {
			draft.addAuditEventLocked("employee_contractor_link_removed", config.Persona.ID, now, "Explicit former employee continuation link was removed from the manual contractor draft.")
			draft.ContinuationActorID = ""
			draft.ContinuationMarkedAt = nil
		} else {
			draft.addAuditEventLocked("employee_contractor_link_created", config.Persona.ID, now, "An authorized operator marked the manual contractor onboarding as a continuation of a former Escape employee record.")
			draft.ContinuationActorID = config.Persona.ID
			markedAt := now
			draft.ContinuationMarkedAt = &markedAt
		}
	}
	draft.ContinuationEmployeeID = clean.ContinuationEmployeeID
	draft.RoomID = clean.RoomID
	draft.Notes = clean.Notes
	draft.UpdatedAt = now
	s.applyDerivedDraftStateLocked(draft, now)
	return cloneOnboardingDraft(draft), true, nil
}

func (s *devOnboardingStoreState) finalize(id string, now time.Time) (*onboardingManualDraft, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	draft, ok := s.drafts[id]
	if !ok {
		return nil, false, false
	}
	if draft.DeletedAt != nil {
		return nil, false, false
	}
	if draft.FinalizedAt != nil {
		return cloneOnboardingDraft(draft), true, false
	}
	if draft.ValidityState == onboardingValidityStateInvalid ||
		draft.ValidityState == onboardingValidityStateReviewRequired {
		return cloneOnboardingDraft(draft), true, true
	}
	if len(draft.missingFields()) > 0 {
		draft.Status = onboardingManualDraftStatusIncomplete
		draft.UpdatedAt = now
		return cloneOnboardingDraft(draft), true, false
	}
	if draft.ChangeReason == core.WorkflowChangeReasonReactivateNonEscape {
		if record := linkedEscapePayloadByID(draft.LinkedEscapePersonID); record != nil {
			if draft.GeneratedEmployeeID == "" {
				draft.GeneratedEmployeeID = record.EmployeeNumber
			}
			if draft.GeneratedEmail == "" {
				draft.GeneratedEmail = record.AssignedEmail
			}
		}
	}
	if draft.ChangeReason == core.WorkflowChangeReasonEmployeeContractorContinuation {
		if record := linkedEscapePayloadByID(draft.LinkedEscapePersonID); record != nil {
			draft.GeneratedEmployeeID = record.EmployeeNumber
			draft.GeneratedEmail = record.AssignedEmail
		}
		draft.addAuditEventLocked("account_continuity_decision", draft.ContinuationActorID, now, "Account remains open for explicit employee-to-contractor continuation; provider writes stay disabled in DEV.")
	}
	if draft.GeneratedEmployeeID == "" {
		draft.GeneratedEmployeeID = "66" + leftPadInt(s.nextEmployee, 5)
		s.nextEmployee++
	}
	if draft.GeneratedEmail == "" {
		draft.GeneratedEmail = s.generatedEmailLocked(draft)
	}
	if isLateStart(draft.StartDate, now) {
		scheduledFor := nextAvailableWorkflowCycle(now)
		draft.ScheduledFor = &scheduledFor
	}
	finalizedAt := now
	draft.FinalizedAt = &finalizedAt
	draft.Status = onboardingManualDraftStatusReady
	draft.UpdatedAt = now
	return cloneOnboardingDraft(draft), true, false
}

func (s *devOnboardingStoreState) softDelete(id string, actor string, now time.Time) (*onboardingManualDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	draft, ok := s.drafts[id]
	if !ok {
		return nil, false
	}
	if draft.DeletedAt != nil {
		return cloneOnboardingDraft(draft), true
	}
	deletedAt := now
	draft.DeletedAt = &deletedAt
	draft.DeletedBy = actor
	draft.UpdatedAt = now
	return cloneOnboardingDraft(draft), true
}

func (s *devOnboardingStoreState) rows(now time.Time, config devPersonaConfig) []onboardingRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	rows := devSeedOnboardingRows(now, s.roomOverrides)
	drafts := s.activeDraftsLocked(now)
	for _, draft := range drafts {
		rows = append(rows, draft.toRowPayload(now))
	}
	return slices.DeleteFunc(rows, func(row onboardingRowPayload) bool {
		return !onboardingVisibleForConfig(row, config)
	})
}

func (s *devOnboardingStoreState) draftPayloads(now time.Time, config devPersonaConfig) []onboardingManualDraftPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	drafts := s.activeDraftsLocked(now)
	payload := make([]onboardingManualDraftPayload, 0, len(drafts))
	for _, draft := range drafts {
		if !onboardingVisibleForConfig(draft.toRowPayload(now), config) {
			continue
		}
		payload = append(payload, draft.toPayload(now))
	}
	return payload
}

func (s *devOnboardingStoreState) updateRoom(id string, request onboardingRoomUpdateRequest, now time.Time, config devPersonaConfig) (onboardingRowPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	if !isRoomAllowedForOnboardingConfig(request.RoomID, config) {
		return onboardingRowPayload{}, http.StatusBadRequest, map[string]string{"room_id": "Room is not an allowed option."}
	}
	rows := devSeedOnboardingRows(now, s.roomOverrides)
	for _, draft := range s.activeDraftsLocked(now) {
		rows = append(rows, draft.toRowPayload(now))
	}
	for _, row := range rows {
		if row.ID != id {
			continue
		}
		if !onboardingVisibleForConfig(row, config) {
			return onboardingRowPayload{}, http.StatusNotFound, nil
		}
		if row.Kind == "manual" && row.ManualDraftID != "" {
			draft := s.drafts[row.ManualDraftID]
			if draft == nil || draft.DeletedAt != nil {
				return onboardingRowPayload{}, http.StatusNotFound, nil
			}
			if draft.FinalizedAt != nil {
				return onboardingRowPayload{}, http.StatusBadRequest, map[string]string{"room_id": "Finalized manual onboarding records are read-only."}
			}
			draft.RoomID = request.RoomID
			draft.UpdatedAt = now
			return draft.toRowPayload(now), http.StatusOK, nil
		}
		s.roomOverrides[id] = request.RoomID
		for _, updated := range devSeedOnboardingRows(now, s.roomOverrides) {
			if updated.ID == id {
				return updated, http.StatusOK, nil
			}
		}
		return onboardingRowPayload{}, http.StatusNotFound, nil
	}
	return onboardingRowPayload{}, http.StatusNotFound, nil
}

func isRoomAllowedForOnboardingConfig(roomID string, config devPersonaConfig) bool {
	if strings.TrimSpace(roomID) == "" {
		return true
	}
	for _, room := range devOnboardingFormOptions(config).Rooms {
		if room.ID == roomID {
			return true
		}
	}
	return false
}

func (s *devOnboardingStoreState) activeDraftsLocked(now time.Time) []*onboardingManualDraft {
	drafts := make([]*onboardingManualDraft, 0, len(s.drafts)+1)
	drafts = append(drafts, devLeadTimeReviewDraft(now))
	for _, draft := range s.drafts {
		if draft.DeletedAt != nil {
			continue
		}
		drafts = append(drafts, draft)
	}
	sort.Slice(drafts, func(i, j int) bool {
		return drafts[i].CreatedAt.Before(drafts[j].CreatedAt)
	})
	return drafts
}

func (s *devOnboardingStoreState) purgeExpiredLocked(now time.Time) {
	for id, draft := range s.drafts {
		if draft.FinalizedAt != nil || draft.DeletedAt != nil || draft.ValidityState == onboardingValidityStateInvalid {
			continue
		}
		if now.Sub(draft.UpdatedAt) > onboardingManualDraftTTL {
			delete(s.drafts, id)
		}
	}
}

func (s *devOnboardingStoreState) generatedEmailLocked(draft *onboardingManualDraft) string {
	existing := map[string]bool{
		"jordan.miles@wusd.org": true,
		"nia.brooks@wusd.org":   true,
		"evan.ruiz@wusd.org":    true,
		"mika.ito@wusd.org":     true,
	}
	for _, record := range devEscapeEmploymentRecords {
		if strings.HasSuffix(record.AssignedEmail, "@wusd.org") {
			existing[strings.ToLower(record.AssignedEmail)] = true
		}
	}
	for _, entry := range devPhoneDirectoryEntries {
		if strings.HasSuffix(entry.Email, "@wusd.org") {
			existing[strings.ToLower(entry.Email)] = true
		}
	}
	for _, other := range s.drafts {
		if other.ID == draft.ID || other.GeneratedEmail == "" {
			continue
		}
		existing[strings.ToLower(other.GeneratedEmail)] = true
	}
	first := normalizeEmailNamePart(draft.FirstName)
	last := normalizeEmailNamePart(draft.LastName)
	if first == "" || last == "" {
		return ""
	}
	candidates := []string{
		first[:1] + last + "@wusd.org",
		first + "." + last + "@wusd.org",
		first[:1] + "." + last + "@wusd.org",
	}
	for _, candidate := range candidates {
		if !existing[candidate] {
			return candidate
		}
	}
	for i := 1; ; i++ {
		candidate := first[:1] + last + leftPadInt(i, 2) + "@wusd.org"
		if !existing[candidate] {
			return candidate
		}
	}
}

func (s *devOnboardingStoreState) applyDerivedDraftStateLocked(draft *onboardingManualDraft, now time.Time) {
	draft.ValidityState = onboardingValidityStateValid
	draft.InvalidReason = ""
	draft.LinkedEscapePersonID = ""
	draft.ChangeReason = ""
	draft.ScheduledFor = nil
	draft.GeneratedEmail = ""
	draft.GeneratedEmployeeID = ""
	if len(draft.missingFields()) > 0 {
		draft.Status = onboardingManualDraftStatusIncomplete
		return
	}

	record := findEscapeEmploymentRecord(draft.FirstName, draft.LastName)
	if record != nil {
		draft.LinkedEscapePersonID = record.ID
		if record.Active {
			draft.ValidityState = onboardingValidityStateInvalid
			draft.InvalidReason = onboardingInvalidReasonActiveEscapeContractorCollision
			draft.ChangeReason = core.WorkflowChangeReasonActiveEscapeContractorCollision
			draft.Status = onboardingManualDraftStatusInvalid
			draft.FinalizedAt = nil
			return
		}
		if draft.ContinuationEmployeeID == "" {
			draft.ValidityState = onboardingValidityStateReviewRequired
			draft.Status = onboardingManualDraftStatusNeedsReview
			return
		}
		if draft.ContinuationEmployeeID != record.ID {
			draft.ValidityState = onboardingValidityStateInvalid
			draft.InvalidReason = onboardingInvalidReasonContinuationMismatch
			draft.Status = onboardingManualDraftStatusInvalid
			draft.FinalizedAt = nil
			return
		}
		if invalidReason := employeeContractorGapInvalidReason(record.EndDate, draft.StartDate); invalidReason != "" {
			draft.ValidityState = onboardingValidityStateInvalid
			draft.InvalidReason = invalidReason
			draft.Status = onboardingManualDraftStatusInvalid
			draft.FinalizedAt = nil
			return
		}
		if strings.TrimSpace(record.AssignedEmail) == "" || strings.TrimSpace(record.EmployeeNumber) == "" {
			draft.ValidityState = onboardingValidityStateInvalid
			draft.InvalidReason = onboardingInvalidReasonContinuationMissingIdentity
			draft.Status = onboardingManualDraftStatusInvalid
			draft.FinalizedAt = nil
			return
		}
		draft.ChangeReason = core.WorkflowChangeReasonEmployeeContractorContinuation
		draft.GeneratedEmail = record.AssignedEmail
		draft.GeneratedEmployeeID = record.EmployeeNumber
	} else {
		if draft.ContinuationEmployeeID != "" {
			linked := inactiveEscapeRecordByID(draft.ContinuationEmployeeID)
			if linked == nil || !sameEscapePerson(linked, draft.FirstName, draft.LastName) {
				draft.LinkedEscapePersonID = draft.ContinuationEmployeeID
				draft.ValidityState = onboardingValidityStateInvalid
				draft.InvalidReason = onboardingInvalidReasonContinuationMismatch
				draft.Status = onboardingManualDraftStatusInvalid
				draft.FinalizedAt = nil
				return
			}
		}
		draft.GeneratedEmail = s.generatedEmailLocked(draft)
	}

	if isLateStart(draft.StartDate, now) {
		scheduledFor := nextAvailableWorkflowCycle(now)
		draft.ScheduledFor = &scheduledFor
	}
	draft.Status = onboardingManualDraftStatusIncomplete
}

func sanitizeManualDraftRequest(request onboardingManualDraftRequest, config devPersonaConfig) (onboardingManualDraftRequest, map[string]string) {
	personalPhoneInput := strings.TrimSpace(request.PersonalPhone)
	clean := onboardingManualDraftRequest{
		StartDate:              strings.TrimSpace(request.StartDate),
		SSNLast4:               strings.TrimSpace(request.SSNLast4),
		EmployeeType:           normalizeSpaces(request.EmployeeType),
		Classification:         normalizeSpaces(request.Classification),
		FirstName:              normalizeSpaces(request.FirstName),
		LastName:               normalizeSpaces(request.LastName),
		JobTitle:               normalizeSpaces(request.JobTitle),
		SiteID:                 strings.TrimSpace(request.SiteID),
		PersonalEmail:          strings.ToLower(strings.TrimSpace(request.PersonalEmail)),
		PersonalPhone:          normalizePersonalPhone(personalPhoneInput),
		PreferredDevice:        normalizeSpaces(request.PreferredDevice),
		RequestedAeriesAccess:  normalizeSpaces(request.RequestedAeriesAccess),
		ReplacingEmployeeID:    strings.TrimSpace(request.ReplacingEmployeeID),
		ContinuationEmployeeID: strings.TrimSpace(request.ContinuationEmployeeID),
		RoomID:                 strings.TrimSpace(request.RoomID),
		Notes:                  normalizeSpaces(request.Notes),
	}
	errors := map[string]string{}
	if clean.StartDate != "" {
		if _, err := time.Parse("2006-01-02", clean.StartDate); err != nil {
			errors["start_date"] = "Start date must be a valid date."
		}
	}
	if clean.SSNLast4 != "" && !onboardingLast4Pattern.MatchString(clean.SSNLast4) {
		errors["ssn_last4"] = "Last 4 SSN must contain exactly 4 digits."
	}
	if clean.PersonalEmail != "" && !onboardingPersonalEmailPattern.MatchString(clean.PersonalEmail) {
		errors["personal_email"] = "Personal email must be a valid email address."
	}
	if personalPhoneInput != "" &&
		!onboardingPersonalPhonePattern.MatchString(personalPhoneInput) &&
		!onboardingFormattedPhonePattern.MatchString(personalPhoneInput) {
		errors["personal_phone"] = "Personal phone must use (NNN) NNN-NNNN or 10 digits."
	}
	options := devOnboardingFormOptions(config)
	validateOption(errors, "employee_type", clean.EmployeeType, options.EmployeeTypes)
	validateOption(errors, "classification", clean.Classification, options.Classifications)
	validateOption(errors, "job_title", clean.JobTitle, options.JobTitles)
	validateSiteOption(errors, clean.SiteID, options.Sites)
	validateOption(errors, "preferred_device", clean.PreferredDevice, options.PreferredDevices)
	validateOption(errors, "requested_aeries_access", clean.RequestedAeriesAccess, options.RequestedAeriesAccess)
	validateReplacingEmployee(errors, clean.ReplacingEmployeeID, options.ReplacingEmployees)
	validateContinuationEmployee(errors, clean.ContinuationEmployeeID, options.ContinuationEmployees)
	validateRoom(errors, clean.RoomID, options.Rooms)
	return clean, errors
}

func validateOption(errors map[string]string, field string, value string, options []string) {
	if value == "" {
		return
	}
	if !slices.Contains(options, value) {
		errors[field] = "Value is not an allowed option."
	}
}

func validateSiteOption(errors map[string]string, value string, sites []devSiteContext) {
	if value == "" {
		return
	}
	for _, site := range sites {
		if site.ID == value {
			return
		}
	}
	errors["site_id"] = "Site is not an allowed option."
}

func validateReplacingEmployee(errors map[string]string, value string, employees []onboardingEmployeeOption) {
	if value == "" {
		return
	}
	for _, employee := range employees {
		if employee.ID == value {
			return
		}
	}
	errors["replacing_employee_id"] = "Replacing employee is not an allowed option."
}

func validateContinuationEmployee(errors map[string]string, value string, employees []onboardingEmployeeOption) {
	if value == "" {
		return
	}
	for _, employee := range employees {
		if employee.ID == value {
			return
		}
	}
	errors["continuation_employee_id"] = "Former employee continuation link is not an allowed option."
}

func validateRoom(errors map[string]string, value string, rooms []onboardingRoomOption) {
	if value == "" {
		return
	}
	for _, room := range rooms {
		if room.ID == value {
			return
		}
	}
	errors["room_id"] = "Room is not an allowed option."
}

func (draft *onboardingManualDraft) missingFields() []string {
	required := []struct {
		name  string
		value string
	}{
		{"start_date", draft.StartDate},
		{"ssn_last4", draft.SSNLast4},
		{"employee_type", draft.EmployeeType},
		{"classification", draft.Classification},
		{"first_name", draft.FirstName},
		{"last_name", draft.LastName},
		{"job_title", draft.JobTitle},
		{"site_id", draft.SiteID},
		{"personal_email", draft.PersonalEmail},
		{"personal_phone", draft.PersonalPhone},
		{"preferred_device", draft.PreferredDevice},
		{"requested_aeries_access", draft.RequestedAeriesAccess},
	}
	missing := []string{}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}

func (draft *onboardingManualDraft) toPayload(now time.Time) onboardingManualDraftPayload {
	site := siteByID(draft.SiteID)
	replacing := replacingEmployeeByID(draft.ReplacingEmployeeID)
	continuation := continuationEmployeeByID(draft.ContinuationEmployeeID)
	room := roomByID(draft.RoomID)
	lateStart := isLateStart(draft.StartDate, now)
	linkedEscapeRecord := linkedEscapePayloadByID(draft.LinkedEscapePersonID)
	validityState := draft.ValidityState
	if validityState == "" {
		validityState = onboardingValidityStateValid
	}
	payload := onboardingManualDraftPayload{
		ID:                     draft.ID,
		Status:                 draft.Status,
		StartDate:              draft.StartDate,
		EffectiveDate:          draft.StartDate,
		SSNLast4:               draft.SSNLast4,
		EmployeeType:           draft.EmployeeType,
		Classification:         draft.Classification,
		FirstName:              draft.FirstName,
		LastName:               draft.LastName,
		JobTitle:               draft.JobTitle,
		SiteID:                 draft.SiteID,
		SiteName:               site.Name,
		PersonalEmail:          draft.PersonalEmail,
		PersonalPhone:          draft.PersonalPhone,
		PreferredDevice:        draft.PreferredDevice,
		RequestedAeriesAccess:  draft.RequestedAeriesAccess,
		ReplacingEmployeeID:    draft.ReplacingEmployeeID,
		ContinuationEmployeeID: draft.ContinuationEmployeeID,
		RoomID:                 draft.RoomID,
		Notes:                  draft.Notes,
		GeneratedEmail:         draft.GeneratedEmail,
		GeneratedEmployeeID:    draft.GeneratedEmployeeID,
		ChangeReason:           string(draft.ChangeReason),
		LateStart:              lateStart,
		ValidityState:          validityState,
		InvalidReason:          draft.InvalidReason,
		LinkedEscapeRecord:     linkedEscapeRecord,
		ConversionDecision:     draft.conversionDecisionPayload(),
		EmployeeTimeline:       draft.employeeTimelinePayload(),
		AuditEvents:            append([]onboardingAuditEventPayload(nil), draft.AuditEvents...),
		CanDeleteManualEntry:   draft.InvalidReason == onboardingInvalidReasonActiveEscapeContractorCollision && draft.DeletedAt == nil,
		MissingFields:          draft.missingFields(),
		CreatedAt:              draft.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              draft.UpdatedAt.Format(time.RFC3339),
	}
	if replacing.ID != "" {
		payload.ReplacingEmployeeName = replacing.Name
		payload.ReplacingEmployeeEmail = replacing.Email
	}
	if continuation.ID != "" {
		payload.ContinuationEmployeeName = continuation.Name
		payload.ContinuationEmployeeEmail = continuation.Email
	}
	if room.ID != "" {
		payload.RoomName = room.Name
	}
	if draft.FinalizedAt != nil {
		payload.FinalizedAt = draft.FinalizedAt.Format(time.RFC3339)
	}
	if draft.ScheduledFor != nil {
		payload.ScheduledFor = formatOnboardingDateTime(*draft.ScheduledFor)
	}
	return payload
}

func (draft *onboardingManualDraft) toRowPayload(now time.Time) onboardingRowPayload {
	person := strings.TrimSpace(strings.TrimSpace(draft.FirstName) + " " + strings.TrimSpace(draft.LastName))
	if person == "" {
		person = "Manual Non-Escape Draft"
	}
	site := siteByID(draft.SiteID)
	workflowStatus := draft.Status
	currentStep := "Manual intake"
	issueAction := "Missing required data"
	linkedEscapeRecord := linkedEscapePayloadByID(draft.LinkedEscapePersonID)
	lateStart := isLateStart(draft.StartDate, now)
	validityState := draft.ValidityState
	if validityState == "" {
		validityState = onboardingValidityStateValid
	}
	if draft.ValidityState == onboardingValidityStateInvalid {
		currentStep = "Unsupported contractor collision"
		issueAction = "Delete manual entry"
		switch draft.InvalidReason {
		case onboardingInvalidReasonContinuationMismatch:
			currentStep = "Continuation link mismatch"
			issueAction = "Correct continuation link"
		case onboardingInvalidReasonContinuationUnknownEndDate:
			currentStep = "Unknown employee end date"
			issueAction = "Confirm employee end date"
		case onboardingInvalidReasonContinuationDateOverlap:
			currentStep = "Continuation date overlap"
			issueAction = "Correct contractor start date"
		case onboardingInvalidReasonContinuationMissingIdentity:
			currentStep = "Continuation identity incomplete"
			issueAction = "Confirm former employee identity"
		}
	} else if draft.ValidityState == onboardingValidityStateReviewRequired {
		currentStep = "Needs HR/IT review"
		issueAction = "Mark continuation or revise"
	} else if len(draft.missingFields()) == 0 {
		currentStep = "Ready"
		issueAction = "Manual Non-Escape record"
		if draft.ChangeReason == core.WorkflowChangeReasonReactivateNonEscape {
			issueAction = "Reuse existing identity"
		}
	}
	if draft.FinalizedAt != nil {
		currentStep = "Workflow queued"
		issueAction = "Mock employee + onboarding workflow"
		if draft.ChangeReason == core.WorkflowChangeReasonReactivateNonEscape {
			issueAction = "Reuse existing identity"
		}
		if draft.ScheduledFor != nil {
			currentStep = "Scheduled for next cycle"
			issueAction = "Late-start catch-up"
		}
	}
	return onboardingRowPayload{
		ID:                   "manual-row-" + draft.ID,
		Kind:                 "manual",
		DateAdded:            formatOnboardingDate(draft.CreatedAt),
		DateAddedReason:      "Manual Non-Escape record added",
		StartDate:            draft.StartDate,
		EffectiveDate:        draft.StartDate,
		LeadTimeWarning:      draft.hasLeadTimeWarning(),
		Person:               person,
		SiteID:               draft.SiteID,
		Site:                 site.Name,
		RoomID:               draft.RoomID,
		RoomName:             roomByID(draft.RoomID).Name,
		CanUpdateRoom:        true,
		CurrentStep:          currentStep,
		IssueAction:          issueAction,
		WorkflowStatus:       workflowStatus,
		ChangeReason:         string(draft.ChangeReason),
		LateStart:            lateStart,
		ScheduledFor:         formatOnboardingDateTimePointer(draft.ScheduledFor),
		ValidityState:        validityState,
		InvalidReason:        draft.InvalidReason,
		LinkedEscapeRecord:   linkedEscapeRecord,
		ConversionDecision:   draft.conversionDecisionPayload(),
		EmployeeTimeline:     draft.employeeTimelinePayload(),
		AuditEvents:          append([]onboardingAuditEventPayload(nil), draft.AuditEvents...),
		CanDeleteManualEntry: draft.InvalidReason == onboardingInvalidReasonActiveEscapeContractorCollision && draft.DeletedAt == nil,
		AssignedEmail:        draft.GeneratedEmail,
		EmployeeNumber:       draft.GeneratedEmployeeID,
		ManualDraftID:        draft.ID,
		WorkflowSteps:        draft.workflowSteps(now),
	}
}

func (draft *onboardingManualDraft) addAuditEventLocked(eventType string, actorID string, occurredAt time.Time, summary string) {
	if actorID == "" {
		actorID = "system"
	}
	draft.AuditEvents = append(draft.AuditEvents, onboardingAuditEventPayload{
		EventType:  eventType,
		ActorID:    actorID,
		OccurredAt: occurredAt.Format(time.RFC3339),
		Summary:    summary,
	})
}

func (draft *onboardingManualDraft) conversionDecisionPayload() *onboardingConversionDecisionPayload {
	record := inactiveEscapeRecordByID(draft.LinkedEscapePersonID)
	if record == nil {
		record = inactiveEscapeRecordByID(draft.ContinuationEmployeeID)
	}
	if record == nil {
		return nil
	}
	gapDays := employeeContractorGapDays(record.EndDate, draft.StartDate)
	decision := "review_required"
	reason := "Former employee match requires HR or IT to mark the contractor onboarding as a continuation before the account can remain open."
	accountEligible := record.AssignedEmail != "" &&
		record.EmployeeNumber != "" &&
		draft.InvalidReason == "" &&
		len(draft.missingFields()) == 0
	if draft.ValidityState == onboardingValidityStateInvalid {
		decision = "blocked"
		reason = "The selected former employee record does not safely match this contractor onboarding."
		switch draft.InvalidReason {
		case onboardingInvalidReasonContinuationUnknownEndDate:
			reason = "The former employee end date is unavailable, so the employee-to-contractor gap cannot be verified."
		case onboardingInvalidReasonContinuationDateOverlap:
			reason = "The contractor start date must be after the former employee end date before the account can remain open."
		case onboardingInvalidReasonContinuationMissingIdentity:
			reason = "The former employee record is missing the district email or employee number required for account continuity."
		}
		accountEligible = false
	}
	if draft.ChangeReason == core.WorkflowChangeReasonEmployeeContractorContinuation && accountEligible {
		decision = "keep_account_open"
		reason = "An authorized HR or IT operator explicitly marked this manual contractor onboarding as a continuation and the former account remains eligible for identity reuse."
	}
	decidedAt := ""
	if draft.ContinuationMarkedAt != nil {
		decidedAt = draft.ContinuationMarkedAt.Format(time.RFC3339)
	}
	return &onboardingConversionDecisionPayload{
		Decision:           decision,
		Reason:             reason,
		GapDays:            gapDays,
		AdjacentGapDays:    onboardingAdjacentConversionGapDays,
		GapWarning:         gapDays > onboardingAdjacentConversionGapDays,
		AccountEligible:    accountEligible,
		LifecycleOwner:     "manual_contractor_record",
		ActorID:            draft.ContinuationActorID,
		DecidedAt:          decidedAt,
		LinkedEmployeeID:   record.ID,
		LinkedContractorID: draft.ID,
	}
}

func employeeContractorGapDays(employeeEndDate string, contractorStartDate string) int {
	end, endOK := parseOnboardingStartDate(employeeEndDate)
	start, startOK := parseOnboardingStartDate(contractorStartDate)
	if !endOK || !startOK || !start.After(end) {
		return 0
	}
	return onboardingDateOrdinal(start) - onboardingDateOrdinal(end)
}

func employeeContractorGapInvalidReason(employeeEndDate string, contractorStartDate string) string {
	end, endOK := parseOnboardingStartDate(employeeEndDate)
	if !endOK {
		return onboardingInvalidReasonContinuationUnknownEndDate
	}
	start, startOK := parseOnboardingStartDate(contractorStartDate)
	if !startOK {
		return ""
	}
	if !start.After(end) {
		return onboardingInvalidReasonContinuationDateOverlap
	}
	return ""
}

func onboardingDateOrdinal(value time.Time) int {
	year, month, day := value.In(onboardingTimeLocation()).Date()
	return int(time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / int64(24*time.Hour/time.Second))
}

func (draft *onboardingManualDraft) employeeTimelinePayload() []onboardingTimelineEntryPayload {
	record := inactiveEscapeRecordByID(draft.LinkedEscapePersonID)
	if record == nil {
		record = inactiveEscapeRecordByID(draft.ContinuationEmployeeID)
	}
	if record == nil {
		return nil
	}
	timeline := []onboardingTimelineEntryPayload{{
		Kind:           "escape_employee_period",
		Label:          "Escape employee assignment",
		Source:         "Escape",
		Owner:          "Escape",
		StartDate:      record.StartDate,
		EndDate:        record.EndDate,
		RecordID:       record.ID,
		LinkedRecordID: draft.ID,
	}}
	gapDays := employeeContractorGapDays(record.EndDate, draft.StartDate)
	if gapDays > 0 {
		warning := ""
		if gapDays > onboardingAdjacentConversionGapDays {
			warning = "Gap exceeds the adjacent conversion threshold and requires HR/IT review."
		}
		timeline = append(timeline, onboardingTimelineEntryPayload{
			Kind:           "assignment_gap",
			Label:          "Gap between employee end and contractor start",
			Source:         "Derived",
			Owner:          "HR + IT review",
			StartDate:      record.EndDate,
			EndDate:        draft.StartDate,
			RecordID:       record.ID,
			LinkedRecordID: draft.ID,
			Warning:        warning,
		})
	}
	timeline = append(timeline, onboardingTimelineEntryPayload{
		Kind:           "manual_contractor_period",
		Label:          "Manual Non-Escape contractor assignment",
		Source:         "Local HR-managed record",
		Owner:          "manual_contractor_record",
		StartDate:      draft.StartDate,
		RecordID:       draft.ID,
		LinkedRecordID: record.ID,
	})
	return timeline
}

func (draft *onboardingManualDraft) workflowSteps(now time.Time) []onboardingWorkflowStep {
	missing := draft.missingFields()
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonActiveEscapeContractorCollision {
		return []onboardingWorkflowStep{{
			Name:   "Manual contractor collision",
			Status: onboardingManualDraftStatusInvalid,
			Detail: "Invalid contractor entry. This person is already an active Escape employee. Escape always takes precedence. We cannot hire an active employee as a contractor. Delete the manual entry to resolve this collision.",
		}}
	}
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonContinuationMismatch {
		return []onboardingWorkflowStep{{
			Name:   "Continuation link",
			Status: onboardingManualDraftStatusInvalid,
			Detail: "The selected former Escape employee record does not match this manual contractor name. Correct the contractor identity or clear the continuation link before saving.",
		}}
	}
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonContinuationUnknownEndDate {
		return []onboardingWorkflowStep{{
			Name:   "Continuation end date",
			Status: onboardingManualDraftStatusInvalid,
			Detail: "The former Escape employee record does not have a valid end date, so the employee-to-contractor gap cannot be verified. Confirm the employee end date before keeping the account open.",
		}}
	}
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonContinuationDateOverlap {
		return []onboardingWorkflowStep{{
			Name:   "Continuation dates",
			Status: onboardingManualDraftStatusInvalid,
			Detail: "The contractor start date must be after the former Escape employee end date before this onboarding can keep the account open.",
		}}
	}
	if draft.ValidityState == onboardingValidityStateInvalid && draft.InvalidReason == onboardingInvalidReasonContinuationMissingIdentity {
		return []onboardingWorkflowStep{{
			Name:   "Continuation identity",
			Status: onboardingManualDraftStatusInvalid,
			Detail: "The former Escape employee record is missing the district email or employee number required to keep the account open. Confirm the preserved identity before marking continuation.",
		}}
	}
	if len(missing) > 0 {
		return []onboardingWorkflowStep{{
			Name:   "Manual intake",
			Status: onboardingManualDraftStatusIncomplete,
			Detail: "Required manual onboarding data is missing. Complete the highlighted fields and save again.",
		}}
	}
	if draft.ValidityState == onboardingValidityStateReviewRequired {
		detail := "A former inactive Escape employee matches this manual contractor. HR or IT must explicitly mark this onboarding as a continuation before the existing identity can remain open."
		if decision := draft.conversionDecisionPayload(); decision != nil && decision.GapWarning {
			detail += " The gap is longer than 14 calendar days, so HR/IT review is required before account continuity."
		}
		return []onboardingWorkflowStep{{
			Name:   "Employee-to-contractor continuation",
			Status: onboardingManualDraftStatusNeedsReview,
			Detail: detail,
		}}
	}
	if draft.FinalizedAt == nil {
		detail := "All required fields are present. Save again to finalize the DEV mock employee and queue onboarding."
		if draft.ChangeReason == core.WorkflowChangeReasonReactivateNonEscape {
			detail = "All required fields are present. Save again to reactivate the existing identity as a manual Non-Escape contractor."
		}
		if draft.ChangeReason == core.WorkflowChangeReasonEmployeeContractorContinuation {
			detail = "All required fields are present. Save again to keep the former employee identity open for this manual Non-Escape contractor continuation."
			if decision := draft.conversionDecisionPayload(); decision != nil && decision.GapWarning {
				detail += " The employee-to-contractor gap exceeds 14 calendar days and remains visible for audit review."
			}
		}
		if draft.ScheduledFor != nil {
			detail += " Because the start date is already in the past, the workflow will run on the next available cycle at " + formatOnboardingDateTime(*draft.ScheduledFor) + "."
		}
		return []onboardingWorkflowStep{{
			Name:   "Manual intake",
			Status: "Ready",
			Detail: detail,
		}}
	}
	identityDetail := "The DEV mock employee is ready for baseline onboarding using the generated employee ID and email."
	stepStatus := "Queued"
	if draft.ChangeReason == core.WorkflowChangeReasonReactivateNonEscape {
		identityDetail = "The existing identity is being reused for this manual Non-Escape contractor reactivation. Baseline-first reprovisioning applies and prior extras are not restored automatically."
	}
	if draft.ChangeReason == core.WorkflowChangeReasonEmployeeContractorContinuation {
		identityDetail = "The former employee identity remains open for this explicitly linked manual contractor continuation. The linked employee and contractor records point to each other, and provider writes remain DEV mock-only in this slice."
	}
	if draft.ScheduledFor != nil {
		stepStatus = "Scheduled"
		identityDetail += " The start date is already in the past, so the workflow is scheduled for the next available cycle at " + formatOnboardingDateTime(*draft.ScheduledFor) + "."
	}
	aeriesDetail := "Requested Aeries access is tracked as workflow data. The app links external IncidentIQ follow-up status when it exists."
	aeriesPayload := provider.BuildAeriesUploadPayload(provider.AeriesUploadInput{
		SourceSystem:    provider.AeriesSourceManualNonEscape,
		EmployeeID:      draft.GeneratedEmployeeID,
		FirstName:       draft.FirstName,
		LastName:        draft.LastName,
		PersonalEmail:   draft.PersonalEmail,
		PersonalPhone:   draft.PersonalPhone,
		RequestedAccess: draft.RequestedAeriesAccess,
		PreferredDevice: draft.PreferredDevice,
		ChangeReason:    string(draft.ChangeReason),
	})
	if _, ok := aeriesPayload["personal_phone"]; ok {
		aeriesDetail += " The manual Non-Escape Aeries upload payload has a captured personal phone number; diagnostics and summaries must use redaction rather than the raw value."
	}
	return []onboardingWorkflowStep{
		{
			Name:   "Identity preparation",
			Status: stepStatus,
			Detail: identityDetail,
		},
		{
			Name:   "Aeries access follow-up",
			Status: "External action",
			Detail: aeriesDetail,
			Actions: []onboardingWorkflowAction{{
				Label:      "Open mock Aeries access request",
				Resolution: "Confirm the requested Aeries role and complete the external user-rights task.",
				System:     "IncidentIQ",
				Href:       mockWorkflowHref("incidentiq", "aeries-"+draft.ID),
			}},
		},
	}
}

func (draft *onboardingManualDraft) hasLeadTimeWarning() bool {
	start, ok := parseOnboardingStartDate(draft.StartDate)
	if !ok || draft.CreatedAt.IsZero() {
		return false
	}
	added := draft.CreatedAt.In(onboardingTimeLocation())
	addedDate := time.Date(added.Year(), added.Month(), added.Day(), 0, 0, 0, 0, onboardingTimeLocation())
	days := int(start.Sub(addedDate).Hours() / 24)
	return days >= 0 && days <= 3
}

func cloneOnboardingDraft(draft *onboardingManualDraft) *onboardingManualDraft {
	if draft == nil {
		return nil
	}
	clone := *draft
	clone.AuditEvents = append([]onboardingAuditEventPayload(nil), draft.AuditEvents...)
	return &clone
}

func devSeedOnboardingRows(now time.Time, roomOverrides map[string]string) []onboardingRowPayload {
	scheduledFor := formatOnboardingDateTime(nextAvailableWorkflowCycle(now))
	scheduledForLateStart := func(startDate string) string {
		if isLateStart(startDate, now) {
			return scheduledFor
		}
		return ""
	}
	rows := []onboardingRowPayload{
		{
			ID: "jordan-miles", Kind: "seed", DateAdded: "Apr 29, 2025", DateAddedReason: "First Escape import", StartDate: "2025-05-06", EffectiveDate: "2025-05-06", Person: "Jordan Miles", SiteID: "clover-hs", Site: "Clover HS", RoomID: "iiq-room-cla-101", RoomName: "CLA Room 101", CanUpdateRoom: true, CurrentStep: "Google pending", IssueAction: "Waiting Entra convergence", WorkflowStatus: "In Progress", LateStart: isLateStart("2025-05-06", now), ScheduledFor: scheduledForLateStart("2025-05-06"), AssignedEmail: "jordan.miles@wusd.org", IncidentIQ: "No local write owned by this app. User lookup retries at most once per hour.", AeriesTicket: "IT-12904 Open", VerkadaTicket: "MOT-4412 Waiting",
			WorkflowSteps: []onboardingWorkflowStep{
				{Name: "Google account", Status: "Complete", Detail: "The account exists and baseline profile planning has completed."},
				{Name: "Entra convergence", Status: "Running", Detail: "AD → Entra propagation is still inside the expected one-hour window."},
				{Name: "IncidentIQ user sync", Status: "Waiting", Detail: "IncidentIQ is expected to sync from Google and Aeries on its normal cadence."},
			},
		},
		{
			ID: "nia-brooks", Kind: "seed", DateAdded: "May 1, 2025", DateAddedReason: "Escape inactive employee set active", StartDate: "2025-05-08", EffectiveDate: "2025-05-08", Person: "Nia Brooks", SiteID: "district-office", Site: "District Office", RoomID: "iiq-room-do-hr", RoomName: "District Office HR", CanUpdateRoom: true, CurrentStep: "Sync dry-run", IssueAction: "Room mapping required", WorkflowStatus: "Needs Review", ChangeReason: string(core.WorkflowChangeReasonReactivateSameRole), LateStart: isLateStart("2025-05-08", now), ScheduledFor: scheduledForLateStart("2025-05-08"), AssignedEmail: "nia.brooks@wusd.org", IncidentIQ: "Room assignment mismatch is waiting on district-office review before provisioning resumes.", AeriesTicket: "IT-12941 Needs room mapping", VerkadaTicket: "MOT-4420 Not started",
			WorkflowSteps: []onboardingWorkflowStep{{
				Name:   "Room mapping",
				Status: "Manual action",
				Detail: "The target room does not match the IncidentIQ room inventory. Confirm or override the room before provisioning resumes. The Escape start date remains authoritative even when the static DEV fixture date is in the past.",
				Actions: []onboardingWorkflowAction{{
					Label:      "Resolve room in IncidentIQ",
					Resolution: "Select the correct room inventory item or document a temporary manual override.",
					System:     "IncidentIQ",
					Href:       mockWorkflowHref("incidentiq", "room-mapping-nia-brooks"),
				}},
			}},
		},
		{
			ID: "evan-ruiz", Kind: "seed", DateAdded: "May 2, 2025", DateAddedReason: "First Escape import", StartDate: "2025-05-12", EffectiveDate: "2025-05-12", Person: "Evan Ruiz", SiteID: "franklin-ms", Site: "Franklin MS", RoomID: "iiq-room-fms-a101", RoomName: "FMS Room A101", CanUpdateRoom: true, CurrentStep: "HR intake", IssueAction: "Missing mandatory field", WorkflowStatus: "Blocked", LateStart: isLateStart("2025-05-12", now), ScheduledFor: scheduledForLateStart("2025-05-12"), AssignedEmail: "evan.ruiz@wusd.org", IncidentIQ: "HR intake is missing a required employment field; downstream account work is blocked.", AeriesTicket: "IT-12988 Waiting on HR", VerkadaTicket: "MOT-4434 Waiting",
			WorkflowSteps: []onboardingWorkflowStep{{
				Name:   "HR intake",
				Status: "Blocked",
				Detail: "Missing required field: Employment type. Update the source record, then rerun the next DEV mock sync.",
				Actions: []onboardingWorkflowAction{{
					Label:      "Open mock HR source record",
					Resolution: "Enter Employment type and confirm the source record is active.",
					System:     "Escape",
					Href:       mockWorkflowHref("escape", "hr-intake-evan-ruiz"),
				}},
			}},
		},
		{
			ID: "mika-ito", Kind: "seed", DateAdded: "May 3, 2025", DateAddedReason: "First Escape import", StartDate: "2025-05-13", EffectiveDate: "2025-05-13", Person: "Mika Ito", SiteID: "desert-view", Site: "Desert View", RoomID: "iiq-room-dve-c118", RoomName: "DVE Room C118", CanUpdateRoom: true, CurrentStep: "Ready", IssueAction: "No blockers", WorkflowStatus: "Ready", LateStart: isLateStart("2025-05-13", now), ScheduledFor: scheduledForLateStart("2025-05-13"), AssignedEmail: "mika.ito@wusd.org", IncidentIQ: "Ready for baseline provisioning. No external follow-up is currently required.", AeriesTicket: "IT-13002 Ready", VerkadaTicket: "MOT-4441 Ready",
			WorkflowSteps: []onboardingWorkflowStep{{Name: "Baseline readiness", Status: "Ready", Detail: "All required context is present. No user action is required."}},
		},
	}
	for index := range rows {
		if roomID, ok := roomOverrides[rows[index].ID]; ok {
			room := roomByID(roomID)
			rows[index].RoomID = room.ID
			rows[index].RoomName = room.Name
		}
	}
	return rows
}

func devLeadTimeReviewDraft(now time.Time) *onboardingManualDraft {
	added := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
	start := added.AddDate(0, 0, 2)
	return &onboardingManualDraft{
		ID:                    "manual-draft-lead-time-review",
		Status:                onboardingManualDraftStatusIncomplete,
		StartDate:             start.Format("2006-01-02"),
		SSNLast4:              "4729",
		EmployeeType:          "Contractor",
		Classification:        "Contractor",
		FirstName:             "Casey",
		LastName:              "Quickstart",
		JobTitle:              "Instructional Aide",
		SiteID:                "district-office",
		PersonalEmail:         "casey.quickstart@example.com",
		PersonalPhone:         "7075550198",
		PreferredDevice:       "Mac",
		RequestedAeriesAccess: "Staff",
		Notes:                 "DEV review row: start date is within three calendar days of Date Added so the intake drawer shows the lead-time warning.",
		CreatedAt:             added,
		UpdatedAt:             added,
	}
}

func formatOnboardingDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("Jan 2, 2006")
}

func formatOnboardingDateTimePointer(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatOnboardingDateTime(*value)
}

func mockWorkflowHref(system string, id string) string {
	return "https://mock.wusd.invalid/" + system + "/" + id
}

func replacingEmployeeByID(id string) onboardingEmployeeOption {
	for _, employee := range devOnboardingFormOptions(devPersonaConfigs["it_admin"]).ReplacingEmployees {
		if employee.ID == id {
			return employee
		}
	}
	return onboardingEmployeeOption{}
}

func continuationEmployeeByID(id string) onboardingEmployeeOption {
	for _, employee := range devOnboardingFormOptions(devPersonaConfigs["it_admin"]).ContinuationEmployees {
		if employee.ID == id {
			return employee
		}
	}
	return onboardingEmployeeOption{}
}

func roomByID(id string) onboardingRoomOption {
	for _, room := range devOnboardingFormOptions(devPersonaConfigs["it_admin"]).Rooms {
		if room.ID == id {
			return room
		}
	}
	return onboardingRoomOption{}
}

func normalizeSpaces(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

// normalizePersonalPhone accepts HR-entered manual Non-Escape phone input from the DEV onboarding drawer and returns only the ten digits needed by the planned Aeries upload payload. Formatting characters are intentionally discarded here so web tests and future serializers compare the same canonical value without logging or displaying the raw operator entry.
func normalizePersonalPhone(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		return digits[1:]
	}
	return digits
}

func normalizeEmailNamePart(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func leftPadInt(value int, width int) string {
	raw := strconv.Itoa(value)
	if len(raw) >= width {
		return raw
	}
	return strings.Repeat("0", width-len(raw)) + raw
}
