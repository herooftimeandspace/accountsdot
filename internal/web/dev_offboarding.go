package web

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

var devOffboardingStore = newDevOffboardingStore()

type offboardingPagePayload struct {
	PageID      string                 `json:"page_id"`
	Persona     devPersona             `json:"persona"`
	Shell       devShellPayload        `json:"shell"`
	GeneratedAt string                 `json:"generated_at"`
	Page        offboardingPageContent `json:"page"`
}

type securityIssuesReportPagePayload struct {
	PageID      string                      `json:"page_id"`
	Persona     devPersona                  `json:"persona"`
	Shell       devShellPayload             `json:"shell"`
	GeneratedAt string                      `json:"generated_at"`
	Page        securityIssuesReportContent `json:"page"`
}

type securityIssuesReportContent struct {
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	LastRefreshed string                  `json:"last_refreshed"`
	SummaryCards  []summaryCardPayload    `json:"summary_cards"`
	Rows          []offboardingRowPayload `json:"rows"`
}

type offboardingPageContent struct {
	Title             string                  `json:"title"`
	Description       string                  `json:"description"`
	LastRefreshed     string                  `json:"last_refreshed"`
	CanManageEndDates bool                    `json:"can_manage_end_dates"`
	CanManageManual   bool                    `json:"can_manage_manual"`
	ShowEmployeeIDs   bool                    `json:"show_employee_ids"`
	SummaryCards      []summaryCardPayload    `json:"summary_cards"`
	Rows              []offboardingRowPayload `json:"rows"`
}

type offboardingRowPayload struct {
	ID                string                    `json:"id"`
	Kind              string                    `json:"kind"`
	Person            string                    `json:"person"`
	Email             string                    `json:"email"`
	EmployeeID        string                    `json:"employee_id,omitempty"`
	SiteID            string                    `json:"site_id"`
	Site              string                    `json:"site"`
	EndDate           string                    `json:"end_date"`
	EndDateSource     string                    `json:"end_date_source"`
	EndDateEditable   bool                      `json:"end_date_editable"`
	Status            string                    `json:"status"`
	NextAction        string                    `json:"next_action"`
	AssetWork         string                    `json:"asset_work"`
	Warning           string                    `json:"warning,omitempty"`
	Details           []offboardingDetail       `json:"details"`
	Actions           []offboardingWorkflowStep `json:"actions"`
	ExternalReference string                    `json:"external_reference,omitempty"`
}

type offboardingDetail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type offboardingWorkflowStep struct {
	Name       string                      `json:"name"`
	Owner      string                      `json:"owner"`
	Status     string                      `json:"status"`
	Detail     string                      `json:"detail"`
	Resolution string                      `json:"resolution"`
	Links      []offboardingWorkflowAction `json:"links,omitempty"`
	Metadata   []offboardingDetail         `json:"metadata,omitempty"`
}

type offboardingWorkflowAction struct {
	Label  string `json:"label"`
	System string `json:"system"`
	Href   string `json:"href"`
}

type offboardingEndDateRequest struct {
	EndDate string `json:"end_date"`
}

type offboardingEndDateResponse struct {
	Row offboardingRowPayload `json:"row"`
}

type offboardingCandidatePayload struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Person          string `json:"person"`
	Email           string `json:"email"`
	EmployeeID      string `json:"employee_id"`
	SiteID          string `json:"site_id"`
	Site            string `json:"site"`
	TerminationDate string `json:"termination_date,omitempty"`
	Source          string `json:"source"`
}

type offboardingCandidatesResponse struct {
	Candidates []offboardingCandidatePayload `json:"candidates"`
}

type offboardingEmergencyRequest struct {
	PersonID string `json:"person_id"`
}

type offboardingContractorRequest struct {
	PersonID string `json:"person_id"`
	EndDate  string `json:"end_date"`
}

type offboardingScheduleResponse struct {
	Action offboardingScheduledAction `json:"action"`
}

type offboardingScheduledAction struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	PersonID     string `json:"person_id"`
	Person       string `json:"person"`
	Email        string `json:"email"`
	ScheduledFor string `json:"scheduled_for"`
	ActorID      string `json:"actor_id"`
	CreatedAt    string `json:"created_at"`
	Mode         string `json:"mode"`
	Status       string `json:"status"`
}

type devOffboardingStoreState struct {
	mu               sync.Mutex
	endDates         map[string]string
	scheduledActions []offboardingScheduledAction
}

type offboardingSeedRecord struct {
	ID                string
	Kind              string
	Person            string
	Email             string
	EmployeeID        string
	SiteID            string
	Site              string
	EndDate           string
	EndDateSource     string
	Status            string
	NextAction        string
	AssetWork         string
	Warning           string
	Details           []offboardingDetail
	Actions           []offboardingWorkflowStep
	ExternalReference string
}

// newDevOffboardingStore initializes the process-local DEV offboarding state.
// Page and mutation handlers share this store for local end-date overrides and
// issue #161 mock schedule evidence without creating provider or database
// writes.
func newDevOffboardingStore() *devOffboardingStoreState {
	return &devOffboardingStoreState{endDates: map[string]string{}}
}

// handleDevOffboardingPage serves the DEV Offboarding read model consumed by
// frontend/src/pages/OffboardingPage.jsx. It requires an authenticated persona
// with /offboarding access, returns HR lifecycle rows plus editable end-date
// flags, and deliberately excludes account-security rows because issue #42
// moved recent-activity security risk review to /reports/security-issues.
func handleDevOffboardingPage(w http.ResponseWriter, r *http.Request) {
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
	if !routeAllowed(r.Context(), config, "/offboarding") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Offboarding is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	now := time.Now().UTC()
	rows := devOffboardingStore.rows(config)
	writeJSON(w, http.StatusOK, offboardingPagePayload{
		PageID:      "offboarding",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: offboardingPageContent{
			Title:             "Offboarding Dashboard",
			Description:       "Offboarding status by person across accounts, licenses, assets, and closeout tasks.",
			LastRefreshed:     "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			CanManageEndDates: canManageOffboardingEndDates(config),
			CanManageManual:   canManageOffboardingManualActions(config),
			ShowEmployeeIDs:   canSeeOffboardingEmployeeIDs(config),
			SummaryCards: []summaryCardPayload{
				{Title: "Scheduled Leaves", Count: "58"},
				{Title: "Immediate Terms", Count: "9"},
				{Title: "Asset Retrieval", Count: "37"},
			},
			Rows: rows,
		},
	})
}

// handleDevSecurityIssuesReportPage serves the IT Admin-only DEV report behind
// /reports/security-issues. It reuses the Offboarding seed row shape so the
// migrated security issue keeps the same detail fields, owner/action context,
// and deterministic mock external links without exposing the Offboarding
// end-date mutation path to HR workflows.
func handleDevSecurityIssuesReportPage(w http.ResponseWriter, r *http.Request) {
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
	if !routeAllowed(r.Context(), config, "/reports/security-issues") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Security issue reports are available only to IT Admin.",
			"persona": config.Persona,
		})
		return
	}

	now := time.Now().UTC()
	rows := devOffboardingStore.securityIssueRows(config)
	writeJSON(w, http.StatusOK, securityIssuesReportPagePayload{
		PageID:      "reports-security-issues",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: securityIssuesReportContent{
			Title:         "Security Issues Report",
			Description:   "Account-security issues that need IT Admin review before or during deprovisioning.",
			LastRefreshed: "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			SummaryCards: []summaryCardPayload{
				{Title: "Security Issues", Count: "6"},
				{Title: "Recent Activity", Count: "1"},
				{Title: "Review Owner", Count: "IT Admin"},
			},
			Rows: rows,
		},
	})
}

// handleDevOffboardingRecord applies HR/IT local end-date edits for the
// selected DEV Offboarding row. It requires /offboarding access plus manual
// end-date permission before parsing the row id, then mutates only the in-memory
// override map documented in docs/planning/external-write-inventory.md.
func handleDevOffboardingRecord(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can update this page.",
		})
		return
	}
	if !routeAllowed(r.Context(), config, "/offboarding") || !canManageOffboardingEndDates(config) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "This persona cannot update offboarding end dates.",
		})
		return
	}
	if r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dev/offboarding/records/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "end-date" {
		http.NotFound(w, r)
		return
	}

	var request offboardingEndDateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_json",
			"message": "Request body must be valid JSON.",
		})
		return
	}

	row, status, errors := devOffboardingStore.updateEndDate(parts[0], request.EndDate, config)
	if status != http.StatusOK {
		body := map[string]any{
			"code":    "offboarding_end_date_rejected",
			"message": "The offboarding end date could not be updated.",
		}
		if len(errors) > 0 {
			body["errors"] = errors
		}
		writeJSON(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, offboardingEndDateResponse{Row: row})
}

// handleDevOffboardingCandidates returns the HR/IT-only search corpus used by
// the Emergency Offboarding and Offboard Contractor drawers. It runs the same
// DEV session, route, and persona checks as mutation endpoints before exposing
// employee IDs or contractor records, so direct calls from site-scoped personas
// fail without receiving searchable employment data.
func handleDevOffboardingCandidates(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can search offboarding candidates.",
		})
		return
	}
	if !routeAllowed(r.Context(), config, "/offboarding") || !canManageOffboardingManualActions(config) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Only Human Resources and IT Admin can search manual offboarding candidates.",
		})
		return
	}
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = "emergency"
	}
	candidates, status, errors := devOffboardingStore.candidates(mode)
	if status != http.StatusOK {
		writeJSON(w, status, map[string]any{
			"code":    "offboarding_candidate_search_rejected",
			"message": "The offboarding candidate search could not be loaded.",
			"errors":  errors,
		})
		return
	}
	writeJSON(w, http.StatusOK, offboardingCandidatesResponse{Candidates: candidates})
}

// handleDevOffboardingEmergencyDeprovision records the DEV-only immediate
// deprovision decision submitted from the Emergency Offboarding drawer. It does
// not call Google, Zoom, IncidentIQ, Escape, or a database; it writes only the
// in-memory audit-shaped action list so the UI and tests can verify the future
// workflow boundary without bypassing the Phase 2 live-write pilot gate.
func handleDevOffboardingEmergencyDeprovision(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	config, ok := requireOffboardingManualManager(w, r, "schedule emergency offboarding")
	if !ok {
		return
	}
	var request offboardingEmergencyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_json",
			"message": "Request body must be valid JSON.",
		})
		return
	}
	action, status, errors := devOffboardingStore.scheduleEmergencyDeprovision(request.PersonID, config)
	writeOffboardingScheduleResult(w, action, status, errors)
}

// handleDevOffboardingContractorSchedule records the DEV-only manual
// contractor offboarding decision submitted from the Offboard Contractor drawer.
// The date is saved only when this endpoint accepts the explicit schedule
// request; editing the date field in React never mutates DEV state by itself.
func handleDevOffboardingContractorSchedule(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	config, ok := requireOffboardingManualManager(w, r, "schedule contractor offboarding")
	if !ok {
		return
	}
	var request offboardingContractorRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_json",
			"message": "Request body must be valid JSON.",
		})
		return
	}
	action, status, errors := devOffboardingStore.scheduleContractorOffboarding(request.PersonID, request.EndDate, config)
	writeOffboardingScheduleResult(w, action, status, errors)
}

func requireOffboardingManualManager(w http.ResponseWriter, r *http.Request, action string) (devPersonaConfig, bool) {
	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can " + action + ".",
		})
		return devPersonaConfig{}, false
	}
	if !routeAllowed(r.Context(), config, "/offboarding") || !canManageOffboardingManualActions(config) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Only Human Resources and IT Admin can manage manual offboarding actions.",
		})
		return devPersonaConfig{}, false
	}
	return config, true
}

func writeOffboardingScheduleResult(w http.ResponseWriter, action offboardingScheduledAction, status int, errors map[string]string) {
	if status != http.StatusOK {
		body := map[string]any{
			"code":    "offboarding_schedule_rejected",
			"message": "The offboarding action could not be scheduled.",
		}
		if len(errors) > 0 {
			body["errors"] = errors
		}
		writeJSON(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, offboardingScheduleResponse{Action: action})
}

// canManageOffboardingEndDates is the HR/IT permission gate for non-Escape
// local end-date edits. Page payloads use it for UI affordances, and the
// mutation handler repeats it before touching DEV state.
func canManageOffboardingEndDates(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

// canManageOffboardingManualActions centralizes the HR/IT-only permission used
// by the Offboarding page payload, candidate search API, and mock scheduling
// APIs so direct payload construction cannot bypass the drawer visibility rule.
func canManageOffboardingManualActions(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

// canSeeOffboardingEmployeeIDs keeps employee ID visibility aligned with the
// PRD field-level rule: HR and IT Admin can review the identifier, while
// site-scoped viewers receive row payloads with the field omitted.
func canSeeOffboardingEmployeeIDs(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

// rows returns only the Offboarding-owned records for the DEV page and update
// route. The caller holds no persistent database transaction; this in-memory
// store applies local end-date overrides for eligible orphan records and leaves
// security-risk rows for securityIssueRows so HR workflows cannot edit them.
func (s *devOffboardingStoreState) rows(config devPersonaConfig) []offboardingRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := make([]offboardingRowPayload, 0, len(devOffboardingSeedRecords))
	for _, record := range devOffboardingSeedRecords {
		if !offboardingRecordVisible(record, config) {
			continue
		}
		rows = append(rows, s.rowPayloadLocked(record, config))
	}
	return rows
}

// securityIssueRows returns the IT Admin report projection for orphan-account
// records whose current owner is security review rather than HR Offboarding.
// The endpoint is read-only in this slice, so the returned rows keep detail and
// action context but mark local end dates as non-editable.
func (s *devOffboardingStoreState) securityIssueRows(config devPersonaConfig) []offboardingRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := make([]offboardingRowPayload, 0, len(devOffboardingSeedRecords))
	for _, record := range devOffboardingSeedRecords {
		if !securityIssueRecordVisible(record, config) {
			continue
		}
		row := s.rowPayloadLocked(record, config)
		row.EndDateEditable = false
		rows = append(rows, row)
	}
	return rows
}

// updateEndDate validates and stores a local DEV end-date override for one
// visible Offboarding row. Escape-backed rows are rejected because Escape owns
// those dates; accepted non-Escape/orphan edits update only the in-memory map.
func (s *devOffboardingStoreState) updateEndDate(id string, value string, config devPersonaConfig) (offboardingRowPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, found := findOffboardingSeedRecord(id)
	if !found || !offboardingRecordVisible(record, config) {
		return offboardingRowPayload{}, http.StatusNotFound, nil
	}
	if record.EndDateSource == "Escape" {
		return offboardingRowPayload{}, http.StatusConflict, map[string]string{
			"end_date": "Escape-backed end dates are read-only. Update the end date in Escape.",
		}
	}
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		delete(s.endDates, id)
		return s.rowPayloadLocked(record, config), http.StatusOK, nil
	}
	if _, err := time.ParseInLocation("2006-01-02", normalized, onboardingTimeLocation()); err != nil {
		return offboardingRowPayload{}, http.StatusBadRequest, map[string]string{
			"end_date": "Use YYYY-MM-DD.",
		}
	}
	s.endDates[id] = normalized
	return s.rowPayloadLocked(record, config), http.StatusOK, nil
}

// candidates returns the authorized search corpus for the selected manual
// offboarding drawer. The caller already verified HR/IT access; this helper
// keeps contractor searches limited to active manual Non-Escape contractors
// while emergency searches include both active employees and contractors.
func (s *devOffboardingStoreState) candidates(mode string) ([]offboardingCandidatePayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch mode {
	case "emergency":
		return append([]offboardingCandidatePayload(nil), devOffboardingEmergencyCandidates...), http.StatusOK, nil
	case "contractor":
		contractors := make([]offboardingCandidatePayload, 0, len(devOffboardingEmergencyCandidates))
		for _, candidate := range devOffboardingEmergencyCandidates {
			if candidate.Kind == "contractor" {
				contractors = append(contractors, candidate)
			}
		}
		return contractors, http.StatusOK, nil
	default:
		return nil, http.StatusBadRequest, map[string]string{
			"mode": "Use emergency or contractor.",
		}
	}
}

// scheduleEmergencyDeprovision records the immediate DEV mock deprovision
// intent after the handler validates HR/IT access. It writes only in-memory
// action evidence and deliberately does not call any provider or database.
func (s *devOffboardingStoreState) scheduleEmergencyDeprovision(personID string, config devPersonaConfig) (offboardingScheduledAction, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate, found := findOffboardingCandidate(personID)
	if !found {
		return offboardingScheduledAction{}, http.StatusNotFound, map[string]string{
			"person_id": "Select an active employee or contractor.",
		}
	}
	action := offboardingScheduledAction{
		ID:           "dev-offboarding-emergency-" + candidate.ID,
		Kind:         "emergency_deprovision",
		PersonID:     candidate.ID,
		Person:       candidate.Person,
		Email:        candidate.Email,
		ScheduledFor: "immediate",
		ActorID:      config.Persona.ID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Mode:         "dev_mock_only",
		Status:       "scheduled",
	}
	s.scheduledActions = append(s.scheduledActions, action)
	return action, http.StatusOK, nil
}

// scheduleContractorOffboarding records the dated DEV mock deprovision intent
// for a selected manual Non-Escape contractor. It validates the submitted date
// at save time so changing the frontend date picker never mutates state alone.
func (s *devOffboardingStoreState) scheduleContractorOffboarding(personID string, endDate string, config devPersonaConfig) (offboardingScheduledAction, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate, found := findOffboardingCandidate(personID)
	if !found || candidate.Kind != "contractor" {
		return offboardingScheduledAction{}, http.StatusNotFound, map[string]string{
			"person_id": "Select an active contractor.",
		}
	}
	normalizedDate := strings.TrimSpace(endDate)
	if normalizedDate == "" {
		return offboardingScheduledAction{}, http.StatusBadRequest, map[string]string{
			"end_date": "Choose a termination date.",
		}
	}
	if _, err := time.ParseInLocation("2006-01-02", normalizedDate, onboardingTimeLocation()); err != nil {
		return offboardingScheduledAction{}, http.StatusBadRequest, map[string]string{
			"end_date": "Use YYYY-MM-DD.",
		}
	}
	action := offboardingScheduledAction{
		ID:           "dev-offboarding-contractor-" + candidate.ID,
		Kind:         "contractor_scheduled_deprovision",
		PersonID:     candidate.ID,
		Person:       candidate.Person,
		Email:        candidate.Email,
		ScheduledFor: normalizedDate,
		ActorID:      config.Persona.ID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Mode:         "dev_mock_only",
		Status:       "scheduled",
	}
	s.scheduledActions = append(s.scheduledActions, action)
	return action, http.StatusOK, nil
}

// findOffboardingCandidate resolves an explicit drawer selection by stable DEV
// candidate id. Schedule handlers use it instead of trusting display fields from
// the client payload.
func findOffboardingCandidate(id string) (offboardingCandidatePayload, bool) {
	trimmed := strings.TrimSpace(id)
	for _, candidate := range devOffboardingEmergencyCandidates {
		if candidate.ID == trimmed {
			return candidate, true
		}
	}
	return offboardingCandidatePayload{}, false
}

// rowPayloadLocked projects a seed record plus any local end-date override into
// the JSON shape consumed by OffboardingPage. The caller holds the store lock so
// the end-date map and visibility-sensitive employee ID field stay consistent.
func (s *devOffboardingStoreState) rowPayloadLocked(record offboardingSeedRecord, config devPersonaConfig) offboardingRowPayload {
	endDate := record.EndDate
	if override, ok := s.endDates[record.ID]; ok {
		endDate = override
	}
	payload := offboardingRowPayload{
		ID:                record.ID,
		Kind:              record.Kind,
		Person:            record.Person,
		Email:             record.Email,
		SiteID:            record.SiteID,
		Site:              record.Site,
		EndDate:           endDate,
		EndDateSource:     record.EndDateSource,
		EndDateEditable:   record.EndDateSource != "Escape" && canManageOffboardingEndDates(config),
		Status:            record.Status,
		NextAction:        record.NextAction,
		AssetWork:         record.AssetWork,
		Warning:           record.Warning,
		Details:           append([]offboardingDetail(nil), record.Details...),
		Actions:           append([]offboardingWorkflowStep(nil), record.Actions...),
		ExternalReference: record.ExternalReference,
	}
	if canSeeOffboardingEmployeeIDs(config) {
		payload.EmployeeID = record.EmployeeID
	}
	return payload
}

// offboardingRecordVisible removes security-risk rows from HR Offboarding and
// applies site scope for Site Admin viewers. IT Admin and HR receive the
// non-security district-wide rows; security rows belong to the IT-only report.
func offboardingRecordVisible(record offboardingSeedRecord, config devPersonaConfig) bool {
	if record.Status == "Security risk" {
		return false
	}
	if config.Persona.ID != "site_admin" {
		return true
	}
	if record.SiteID == "" {
		return false
	}
	return slices.ContainsFunc(config.VisibleSites, func(site devSiteContext) bool {
		return site.ID == record.SiteID
	})
}

// securityIssueRecordVisible scopes moved account-security rows to the IT Admin
// Reports surface. Non-IT personas do not receive these rows through the report
// endpoint, and Offboarding filtering remains separate so HR can still manage
// non-security orphan-account end dates.
func securityIssueRecordVisible(record offboardingSeedRecord, config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" && record.Status == "Security risk"
}

// findOffboardingSeedRecord resolves a stable DEV row id before an end-date
// mutation can proceed. Unknown ids fail closed so a valid-looking payload
// cannot create a new offboarding row.
func findOffboardingSeedRecord(id string) (offboardingSeedRecord, bool) {
	for _, record := range devOffboardingSeedRecords {
		if record.ID == id {
			return record, true
		}
	}
	return offboardingSeedRecord{}, false
}

var devOffboardingSeedRecords = []offboardingSeedRecord{
	{
		ID:            "escape-chris-morgan",
		Kind:          "escape",
		Person:        "Chris Morgan",
		Email:         "chris.morgan@wusd.org",
		EmployeeID:    "103118",
		SiteID:        "clover-hs",
		Site:          "Clover HS",
		EndDate:       "2025-05-03",
		EndDateSource: "Escape",
		Status:        "Manual action",
		NextAction:    "Retrieve laptop and badge",
		AssetWork:     "2 items",
		Details: []offboardingDetail{
			{Label: "Source", Value: "Escape"},
			{Label: "End date ownership", Value: "Escape is authoritative. HR or IT must correct the source record in Escape."},
			{Label: "Assigned email", Value: "chris.morgan@wusd.org"},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "Asset retrieval",
				Owner:      "Site Admin",
				Status:     "Manual action",
				Detail:     "Laptop and badge are assigned directly to Chris Morgan and must be collected.",
				Resolution: "Collect the assigned items, update IncidentIQ asset custody, and confirm the closeout ticket.",
				Links: []offboardingWorkflowAction{
					{Label: "Open asset task", System: "IncidentIQ", Href: "https://mock.wusd.local/incidentiq/tickets/IT-13044"},
				},
			},
		},
		ExternalReference: "IT-13044",
	},
	{
		ID:            "escape-taylor-singh",
		Kind:          "escape",
		Person:        "Taylor Singh",
		Email:         "taylor.singh@wusd.org",
		EmployeeID:    "103442",
		SiteID:        "district-office",
		Site:          "District Office",
		EndDate:       "2025-05-09",
		EndDateSource: "Escape",
		Status:        "Scheduled",
		NextAction:    "License reclaim queued",
		AssetWork:     "1 item",
		Details: []offboardingDetail{
			{Label: "Source", Value: "Escape"},
			{Label: "End date ownership", Value: "Escape is authoritative. This dashboard does not override it."},
			{Label: "Assigned email", Value: "taylor.singh@wusd.org"},
			{Label: "Licenses to reclaim", Value: "Zoom Workplace for Education Enterprise Essentials; Zoom Phone Basic"},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "License reclaim",
				Owner:      "Automation",
				Status:     "Scheduled",
				Detail:     "Zoom Workplace for Education Enterprise Essentials and Zoom Phone Basic are queued for reclamation on the Escape end date.",
				Resolution: "No manual action is needed unless the scheduled job reports a failure reclaiming Zoom Workplace for Education Enterprise Essentials, Zoom Phone Basic, or the baseline Google Workspace assignment.",
			},
		},
	},
	{
		ID:            "escape-jamie-reed",
		Kind:          "escape",
		Person:        "Jamie Reed",
		Email:         "jamie.reed@wusd.org",
		EmployeeID:    "103772",
		SiteID:        "desert-view",
		Site:          "Desert View",
		EndDate:       "2025-05-12",
		EndDateSource: "Escape",
		Status:        "Blocked",
		NextAction:    "Exception review needed",
		AssetWork:     "0 items",
		Details: []offboardingDetail{
			{Label: "Source", Value: "Escape"},
			{Label: "End date ownership", Value: "Escape is authoritative. Correct upstream if this date is wrong."},
			{Label: "Assigned email", Value: "jamie.reed@wusd.org"},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "Security exception",
				Owner:      "IT Admin",
				Status:     "Blocked",
				Detail:     "The account has an active exception that must be reviewed before normal deprovisioning can continue.",
				Resolution: "Review the exception owner, reason, and review date before approving the next run.",
				Links: []offboardingWorkflowAction{
					{Label: "Review exception", System: "Admin", Href: "https://mock.wusd.local/admin/offboarding-exceptions/jamie-reed"},
				},
			},
		},
	},
	{
		ID:            "escape-robin-hall",
		Kind:          "escape",
		Person:        "Robin Hall",
		Email:         "robin.hall@wusd.org",
		EmployeeID:    "104012",
		SiteID:        "franklin-ms",
		Site:          "Franklin MS",
		EndDate:       "2025-05-18",
		EndDateSource: "Escape",
		Status:        "Ready",
		NextAction:    "All provider checks passed",
		AssetWork:     "3 items",
		Details: []offboardingDetail{
			{Label: "Source", Value: "Escape"},
			{Label: "End date ownership", Value: "Escape is authoritative. No local override is available."},
			{Label: "Assigned email", Value: "robin.hall@wusd.org"},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "Provider checks",
				Owner:      "Automation",
				Status:     "Ready",
				Detail:     "All provider checks passed and the offboarding run is ready for the scheduled end date.",
				Resolution: "No manual action is needed.",
			},
		},
	},
	{
		ID:            "orphan-avery-cole",
		Kind:          "orphan",
		Person:        "Avery Cole",
		Email:         "avery.cole@wusd.org",
		SiteID:        "district-office",
		Site:          "District Office",
		EndDate:       "2026-06-30",
		EndDateSource: "Local override",
		Status:        "Manual action",
		NextAction:    "Set or confirm end date",
		AssetWork:     "0 items",
		Warning:       "No linked Escape, Aeries, or local employee record. HR/IT must confirm the end date before deprovisioning.",
		Details: []offboardingDetail{
			{Label: "Source", Value: "AD-active orphan account"},
			{Label: "End date ownership", Value: "Local HR/IT-managed date because no authoritative Escape row exists."},
			{Label: "Assigned email", Value: "avery.cole@wusd.org"},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "Confirm orphan account",
				Owner:      "Human Resources",
				Status:     "Manual action",
				Detail:     "The account has no linked Escape, Aeries, or local override record.",
				Resolution: "Confirm whether the account belongs to an active worker. If not, set the end date and allow deprovisioning to proceed.",
				Links: []offboardingWorkflowAction{
					{Label: "Review account", System: "Google Admin", Href: "https://mock.wusd.local/google/users/avery.cole"},
				},
			},
		},
	},
	{
		ID:            "orphan-riley-park",
		Kind:          "orphan",
		Person:        "Riley Park",
		Email:         "riley.park@wusd.org",
		SiteID:        "clover-hs",
		Site:          "Clover HS",
		EndDate:       "2026-06-30",
		EndDateSource: "Local override",
		Status:        "Security risk",
		NextAction:    "Recent activity review",
		AssetWork:     "0 items",
		Warning:       "Recent Google activity after source-system inactivity must be treated as a security risk.",
		Details: []offboardingDetail{
			{Label: "Source", Value: "AD-active orphan account"},
			{Label: "Recent activity", Value: "Google login activity within the last 30 days."},
			{Label: "End date ownership", Value: "Local HR/IT-managed date because no authoritative Escape row exists."},
		},
		Actions: []offboardingWorkflowStep{
			{
				Name:       "Security review",
				Owner:      "IT Admin",
				Status:     "Security risk",
				Detail:     "This orphan account still shows recent Google activity after source-system inactivity.",
				Resolution: "Confirm whether the login is expected. If not, deprovision the account and open a security review ticket.",
				Links: []offboardingWorkflowAction{
					{Label: "Open activity log", System: "Google Admin", Href: "https://mock.wusd.local/google/activity/riley.park"},
				},
			},
		},
	},
}

var devOffboardingEmergencyCandidates = []offboardingCandidatePayload{
	{
		ID:              "employee-chris-morgan",
		Kind:            "employee",
		Person:          "Chris Morgan",
		Email:           "chris.morgan@wusd.org",
		EmployeeID:      "103118",
		SiteID:          "clover-hs",
		Site:            "Clover HS",
		TerminationDate: "2025-05-03",
		Source:          "Escape",
	},
	{
		ID:         "employee-taylor-singh",
		Kind:       "employee",
		Person:     "Taylor Singh",
		Email:      "taylor.singh@wusd.org",
		EmployeeID: "103442",
		SiteID:     "district-office",
		Site:       "District Office",
		Source:     "Escape",
	},
	{
		ID:              "contractor-sam-ortega",
		Kind:            "contractor",
		Person:          "Sam Ortega",
		Email:           "sam.ortega@wusd.org",
		EmployeeID:      "6600142",
		SiteID:          "desert-view",
		Site:            "Desert View",
		TerminationDate: "2026-06-30",
		Source:          "Manual Non-Escape",
	},
	{
		ID:         "contractor-nina-patel",
		Kind:       "contractor",
		Person:     "Nina Patel",
		Email:      "nina.patel@wusd.org",
		EmployeeID: "6600184",
		SiteID:     "franklin-ms",
		Site:       "Franklin MS",
		Source:     "Manual Non-Escape",
	},
}
