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
)

const (
	onboardingManualDraftStatusIncomplete = "Incomplete Data"
	onboardingManualDraftStatusReady      = "Ready to Provision"
	onboardingManualDraftTTL              = 30 * 24 * time.Hour
)

var (
	onboardingPersonalEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	onboardingLast4Pattern         = regexp.MustCompile(`^\d{4}$`)
	devOnboardingStore             = newDevOnboardingStore()
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
	ID              string                   `json:"id"`
	Kind            string                   `json:"kind"`
	DateAdded       string                   `json:"date_added"`
	DateAddedReason string                   `json:"date_added_reason"`
	StartDate       string                   `json:"start_date"`
	Person          string                   `json:"person"`
	Site            string                   `json:"site"`
	CurrentStep     string                   `json:"current_step"`
	IssueAction     string                   `json:"issue_action"`
	WorkflowStatus  string                   `json:"workflow_status"`
	AssignedEmail   string                   `json:"assigned_email,omitempty"`
	EmployeeNumber  string                   `json:"employee_number,omitempty"`
	ManualDraftID   string                   `json:"manual_draft_id,omitempty"`
	IncidentIQ      string                   `json:"incident_iq,omitempty"`
	AeriesTicket    string                   `json:"aeries_ticket,omitempty"`
	VerkadaTicket   string                   `json:"verkada_ticket,omitempty"`
	WorkflowSteps   []onboardingWorkflowStep `json:"workflow_steps,omitempty"`
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

type onboardingFormOptions struct {
	EmployeeTypes         []string                   `json:"employee_types"`
	Classifications       []string                   `json:"classifications"`
	JobTitles             []string                   `json:"job_titles"`
	Sites                 []devSiteContext           `json:"sites"`
	PreferredDevices      []string                   `json:"preferred_devices"`
	RequestedAeriesAccess []string                   `json:"requested_aeries_access"`
	ReplacingEmployees    []onboardingEmployeeOption `json:"replacing_employees"`
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
	StartDate             string `json:"start_date"`
	SSNLast4              string `json:"ssn_last4"`
	EmployeeType          string `json:"employee_type"`
	Classification        string `json:"classification"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	JobTitle              string `json:"job_title"`
	SiteID                string `json:"site_id"`
	PersonalEmail         string `json:"personal_email"`
	PreferredDevice       string `json:"preferred_device"`
	RequestedAeriesAccess string `json:"requested_aeries_access"`
	ReplacingEmployeeID   string `json:"replacing_employee_id"`
	RoomID                string `json:"room_id"`
	Notes                 string `json:"notes"`
}

type onboardingManualDraftPayload struct {
	ID                     string   `json:"id"`
	Status                 string   `json:"status"`
	StartDate              string   `json:"start_date"`
	SSNLast4               string   `json:"ssn_last4,omitempty"`
	EmployeeType           string   `json:"employee_type"`
	Classification         string   `json:"classification"`
	FirstName              string   `json:"first_name"`
	LastName               string   `json:"last_name"`
	JobTitle               string   `json:"job_title"`
	SiteID                 string   `json:"site_id"`
	SiteName               string   `json:"site_name"`
	PersonalEmail          string   `json:"personal_email"`
	PreferredDevice        string   `json:"preferred_device"`
	RequestedAeriesAccess  string   `json:"requested_aeries_access"`
	ReplacingEmployeeID    string   `json:"replacing_employee_id,omitempty"`
	ReplacingEmployeeName  string   `json:"replacing_employee_name,omitempty"`
	ReplacingEmployeeEmail string   `json:"replacing_employee_email,omitempty"`
	RoomID                 string   `json:"room_id,omitempty"`
	RoomName               string   `json:"room_name,omitempty"`
	Notes                  string   `json:"notes,omitempty"`
	GeneratedEmail         string   `json:"generated_email,omitempty"`
	GeneratedEmployeeID    string   `json:"generated_employee_id,omitempty"`
	MissingFields          []string `json:"missing_fields"`
	CreatedAt              string   `json:"created_at"`
	UpdatedAt              string   `json:"updated_at"`
	FinalizedAt            string   `json:"finalized_at,omitempty"`
}

type onboardingManualDraftResponse struct {
	Draft onboardingManualDraftPayload `json:"draft"`
	Rows  []onboardingRowPayload       `json:"rows,omitempty"`
}

type devOnboardingStoreState struct {
	mu           sync.Mutex
	nextDraft    int
	nextEmployee int
	drafts       map[string]*onboardingManualDraft
}

type onboardingManualDraft struct {
	ID                    string
	Status                string
	StartDate             string
	SSNLast4              string
	EmployeeType          string
	Classification        string
	FirstName             string
	LastName              string
	JobTitle              string
	SiteID                string
	PersonalEmail         string
	PreferredDevice       string
	RequestedAeriesAccess string
	ReplacingEmployeeID   string
	RoomID                string
	Notes                 string
	GeneratedEmail        string
	GeneratedEmployeeID   string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	FinalizedAt           *time.Time
}

func newDevOnboardingStore() *devOnboardingStoreState {
	return &devOnboardingStoreState{
		nextDraft:    1,
		nextEmployee: 1,
		drafts:       map[string]*onboardingManualDraft{},
	}
}

func handleDevOnboardingPage(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodGet {
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
	if !routeAllowed(config, "/onboarding") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Onboarding is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

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
			Rows:                  devOnboardingStore.rows(now),
			Drafts:                devOnboardingStore.draftPayloads(now),
			ManualDraftRetention:  "30 days",
			ManualAutosaveSeconds: 60,
		},
		Form:     devOnboardingFormOptions(config),
		Hotspots: map[string]hotspotPayload{"add_manual": {NodeID: "f109", Label: "Add Non-Escape Record"}},
	})
}

func handleDevOnboardingManualDrafts(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	config, ok := requireManualOnboardingManager(w, r)
	if !ok {
		return
	}
	now := time.Now().UTC()
	draft := devOnboardingStore.create(now, config)
	writeJSON(w, http.StatusCreated, onboardingManualDraftResponse{Draft: draft.toPayload()})
}

func handleDevOnboardingManualDraft(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() {
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
		draft, found, validationErrors := devOnboardingStore.update(draftID, request, time.Now().UTC(), config)
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
				"draft":   draft.toPayload(),
			})
			return
		}
		writeJSON(w, http.StatusOK, onboardingManualDraftResponse{Draft: draft.toPayload()})
	case r.Method == http.MethodPost && finalize:
		draft, found := devOnboardingStore.finalize(draftID, time.Now().UTC())
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "not_found",
				"message": "Manual onboarding draft was not found.",
			})
			return
		}
		writeJSON(w, http.StatusOK, onboardingManualDraftResponse{
			Draft: draft.toPayload(),
			Rows:  devOnboardingStore.rows(time.Now().UTC()),
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
		Rooms: []onboardingRoomOption{
			{ID: "iiq-room-cla-101", Name: "CLA Room 101", SiteID: "clover-hs", SiteName: "Clover High School"},
			{ID: "iiq-room-do-hr", Name: "District Office HR", SiteID: "district-office", SiteName: "District Office"},
			{ID: "iiq-room-fms-a101", Name: "FMS Room A101", SiteID: "franklin-ms", SiteName: "Franklin Middle School"},
		},
	}
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
		ID:        id,
		Status:    onboardingManualDraftStatusIncomplete,
		SiteID:    siteID,
		CreatedAt: now,
		UpdatedAt: now,
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
	draft.PreferredDevice = clean.PreferredDevice
	draft.RequestedAeriesAccess = clean.RequestedAeriesAccess
	draft.ReplacingEmployeeID = clean.ReplacingEmployeeID
	draft.RoomID = clean.RoomID
	draft.Notes = clean.Notes
	draft.UpdatedAt = now
	if len(draft.missingFields()) > 0 {
		draft.Status = onboardingManualDraftStatusIncomplete
	} else {
		draft.Status = onboardingManualDraftStatusIncomplete
		draft.GeneratedEmail = s.generatedEmailLocked(draft)
	}
	return cloneOnboardingDraft(draft), true, nil
}

func (s *devOnboardingStoreState) finalize(id string, now time.Time) (*onboardingManualDraft, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	draft, ok := s.drafts[id]
	if !ok {
		return nil, false
	}
	if len(draft.missingFields()) > 0 {
		draft.Status = onboardingManualDraftStatusIncomplete
		draft.UpdatedAt = now
		return cloneOnboardingDraft(draft), true
	}
	if draft.GeneratedEmployeeID == "" {
		draft.GeneratedEmployeeID = "66" + leftPadInt(s.nextEmployee, 5)
		s.nextEmployee++
	}
	if draft.GeneratedEmail == "" {
		draft.GeneratedEmail = s.generatedEmailLocked(draft)
	}
	finalizedAt := now
	draft.FinalizedAt = &finalizedAt
	draft.Status = onboardingManualDraftStatusReady
	draft.UpdatedAt = now
	return cloneOnboardingDraft(draft), true
}

func (s *devOnboardingStoreState) rows(now time.Time) []onboardingRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	rows := devSeedOnboardingRows()
	drafts := make([]*onboardingManualDraft, 0, len(s.drafts))
	for _, draft := range s.drafts {
		drafts = append(drafts, draft)
	}
	sort.Slice(drafts, func(i, j int) bool {
		return drafts[i].CreatedAt.Before(drafts[j].CreatedAt)
	})
	for _, draft := range drafts {
		rows = append(rows, draft.toRowPayload())
	}
	return rows
}

func (s *devOnboardingStoreState) draftPayloads(now time.Time) []onboardingManualDraftPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked(now)
	drafts := make([]*onboardingManualDraft, 0, len(s.drafts))
	for _, draft := range s.drafts {
		drafts = append(drafts, draft)
	}
	sort.Slice(drafts, func(i, j int) bool {
		return drafts[i].CreatedAt.Before(drafts[j].CreatedAt)
	})
	payload := make([]onboardingManualDraftPayload, 0, len(drafts))
	for _, draft := range drafts {
		payload = append(payload, draft.toPayload())
	}
	return payload
}

func (s *devOnboardingStoreState) purgeExpiredLocked(now time.Time) {
	for id, draft := range s.drafts {
		if draft.FinalizedAt != nil {
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

func sanitizeManualDraftRequest(request onboardingManualDraftRequest, config devPersonaConfig) (onboardingManualDraftRequest, map[string]string) {
	clean := onboardingManualDraftRequest{
		StartDate:             strings.TrimSpace(request.StartDate),
		SSNLast4:              strings.TrimSpace(request.SSNLast4),
		EmployeeType:          normalizeSpaces(request.EmployeeType),
		Classification:        normalizeSpaces(request.Classification),
		FirstName:             normalizeSpaces(request.FirstName),
		LastName:              normalizeSpaces(request.LastName),
		JobTitle:              normalizeSpaces(request.JobTitle),
		SiteID:                strings.TrimSpace(request.SiteID),
		PersonalEmail:         strings.ToLower(strings.TrimSpace(request.PersonalEmail)),
		PreferredDevice:       normalizeSpaces(request.PreferredDevice),
		RequestedAeriesAccess: normalizeSpaces(request.RequestedAeriesAccess),
		ReplacingEmployeeID:   strings.TrimSpace(request.ReplacingEmployeeID),
		RoomID:                strings.TrimSpace(request.RoomID),
		Notes:                 normalizeSpaces(request.Notes),
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
	options := devOnboardingFormOptions(config)
	validateOption(errors, "employee_type", clean.EmployeeType, options.EmployeeTypes)
	validateOption(errors, "classification", clean.Classification, options.Classifications)
	validateOption(errors, "job_title", clean.JobTitle, options.JobTitles)
	validateSiteOption(errors, clean.SiteID, options.Sites)
	validateOption(errors, "preferred_device", clean.PreferredDevice, options.PreferredDevices)
	validateOption(errors, "requested_aeries_access", clean.RequestedAeriesAccess, options.RequestedAeriesAccess)
	validateReplacingEmployee(errors, clean.ReplacingEmployeeID, options.ReplacingEmployees)
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

func (draft *onboardingManualDraft) toPayload() onboardingManualDraftPayload {
	site := siteByID(draft.SiteID)
	replacing := replacingEmployeeByID(draft.ReplacingEmployeeID)
	room := roomByID(draft.RoomID)
	payload := onboardingManualDraftPayload{
		ID:                    draft.ID,
		Status:                draft.Status,
		StartDate:             draft.StartDate,
		SSNLast4:              draft.SSNLast4,
		EmployeeType:          draft.EmployeeType,
		Classification:        draft.Classification,
		FirstName:             draft.FirstName,
		LastName:              draft.LastName,
		JobTitle:              draft.JobTitle,
		SiteID:                draft.SiteID,
		SiteName:              site.Name,
		PersonalEmail:         draft.PersonalEmail,
		PreferredDevice:       draft.PreferredDevice,
		RequestedAeriesAccess: draft.RequestedAeriesAccess,
		ReplacingEmployeeID:   draft.ReplacingEmployeeID,
		RoomID:                draft.RoomID,
		Notes:                 draft.Notes,
		GeneratedEmail:        draft.GeneratedEmail,
		GeneratedEmployeeID:   draft.GeneratedEmployeeID,
		MissingFields:         draft.missingFields(),
		CreatedAt:             draft.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             draft.UpdatedAt.Format(time.RFC3339),
	}
	if replacing.ID != "" {
		payload.ReplacingEmployeeName = replacing.Name
		payload.ReplacingEmployeeEmail = replacing.Email
	}
	if room.ID != "" {
		payload.RoomName = room.Name
	}
	if draft.FinalizedAt != nil {
		payload.FinalizedAt = draft.FinalizedAt.Format(time.RFC3339)
	}
	return payload
}

func (draft *onboardingManualDraft) toRowPayload() onboardingRowPayload {
	person := strings.TrimSpace(strings.TrimSpace(draft.FirstName) + " " + strings.TrimSpace(draft.LastName))
	if person == "" {
		person = "Manual Non-Escape Draft"
	}
	site := siteByID(draft.SiteID)
	workflowStatus := draft.Status
	currentStep := "Manual intake"
	issueAction := "Missing required data"
	if len(draft.missingFields()) == 0 {
		currentStep = "Ready"
		issueAction = "Manual Non-Escape record"
	}
	if draft.FinalizedAt != nil {
		currentStep = "Workflow queued"
		issueAction = "Mock employee + onboarding workflow"
	}
	return onboardingRowPayload{
		ID:              "manual-row-" + draft.ID,
		Kind:            "manual",
		DateAdded:       formatOnboardingDate(draft.CreatedAt),
		DateAddedReason: "Manual Non-Escape record added",
		StartDate:       draft.StartDate,
		Person:          person,
		Site:            site.Name,
		CurrentStep:     currentStep,
		IssueAction:     issueAction,
		WorkflowStatus:  workflowStatus,
		AssignedEmail:   draft.GeneratedEmail,
		EmployeeNumber:  draft.GeneratedEmployeeID,
		ManualDraftID:   draft.ID,
		WorkflowSteps:   draft.workflowSteps(),
	}
}

func (draft *onboardingManualDraft) workflowSteps() []onboardingWorkflowStep {
	missing := draft.missingFields()
	if len(missing) > 0 {
		return []onboardingWorkflowStep{{
			Name:   "Manual intake",
			Status: onboardingManualDraftStatusIncomplete,
			Detail: "Required manual onboarding data is missing. Complete the highlighted fields and save again.",
		}}
	}
	if draft.FinalizedAt == nil {
		return []onboardingWorkflowStep{{
			Name:   "Manual intake",
			Status: "Ready",
			Detail: "All required fields are present. Save again to finalize the DEV mock employee and queue onboarding.",
		}}
	}
	return []onboardingWorkflowStep{
		{
			Name:   "Mock employee creation",
			Status: "Queued",
			Detail: "The DEV mock employee is ready for baseline onboarding using the generated employee ID and email.",
		},
		{
			Name:   "Aeries access follow-up",
			Status: "External action",
			Detail: "Requested Aeries access is tracked as workflow data. The app links external IncidentIQ follow-up status when it exists.",
			Actions: []onboardingWorkflowAction{{
				Label:      "Open mock Aeries access request",
				Resolution: "Confirm the requested Aeries role and complete the external user-rights task.",
				System:     "IncidentIQ",
				Href:       mockWorkflowHref("incidentiq", "aeries-"+draft.ID),
			}},
		},
	}
}

func cloneOnboardingDraft(draft *onboardingManualDraft) *onboardingManualDraft {
	if draft == nil {
		return nil
	}
	clone := *draft
	return &clone
}

func devSeedOnboardingRows() []onboardingRowPayload {
	return []onboardingRowPayload{
		{
			ID: "jordan-miles", Kind: "seed", DateAdded: "Apr 29, 2025", DateAddedReason: "First Escape import", StartDate: "May 6, 2025", Person: "Jordan Miles", Site: "Clover HS", CurrentStep: "Google pending", IssueAction: "Waiting Entra convergence", WorkflowStatus: "In Progress", AssignedEmail: "jordan.miles@wusd.org", IncidentIQ: "No local write owned by this app. User lookup retries at most once per hour.", AeriesTicket: "IT-12904 Open", VerkadaTicket: "MOT-4412 Waiting",
			WorkflowSteps: []onboardingWorkflowStep{
				{Name: "Google account", Status: "Complete", Detail: "The account exists and baseline profile planning has completed."},
				{Name: "Entra convergence", Status: "Running", Detail: "AD -> Entra propagation is still inside the expected one-hour window."},
				{Name: "IncidentIQ user sync", Status: "Waiting", Detail: "IncidentIQ is expected to sync from Google and Aeries on its normal cadence."},
			},
		},
		{
			ID: "nia-brooks", Kind: "seed", DateAdded: "May 1, 2025", DateAddedReason: "Escape inactive employee set active", StartDate: "May 8, 2025", Person: "Nia Brooks", Site: "District Office", CurrentStep: "Sync dry-run", IssueAction: "Room mapping required", WorkflowStatus: "Needs Review", AssignedEmail: "nia.brooks@wusd.org", IncidentIQ: "Room assignment mismatch is waiting on district-office review before provisioning resumes.", AeriesTicket: "IT-12941 Needs room mapping", VerkadaTicket: "MOT-4420 Not started",
			WorkflowSteps: []onboardingWorkflowStep{{
				Name:   "Room mapping",
				Status: "Manual action",
				Detail: "The target room does not match the IncidentIQ room inventory. Confirm or override the room before provisioning resumes.",
				Actions: []onboardingWorkflowAction{{
					Label:      "Resolve room in IncidentIQ",
					Resolution: "Select the correct room inventory item or document a temporary manual override.",
					System:     "IncidentIQ",
					Href:       mockWorkflowHref("incidentiq", "room-mapping-nia-brooks"),
				}},
			}},
		},
		{
			ID: "evan-ruiz", Kind: "seed", DateAdded: "May 2, 2025", DateAddedReason: "First Escape import", StartDate: "May 12, 2025", Person: "Evan Ruiz", Site: "Franklin MS", CurrentStep: "HR intake", IssueAction: "Missing mandatory field", WorkflowStatus: "Blocked", AssignedEmail: "evan.ruiz@wusd.org", IncidentIQ: "HR intake is missing a required employment field; downstream account work is blocked.", AeriesTicket: "IT-12988 Waiting on HR", VerkadaTicket: "MOT-4434 Waiting",
			WorkflowSteps: []onboardingWorkflowStep{{
				Name:   "HR intake",
				Status: "Blocked",
				Detail: "A required HR source field is missing. Update the source record, then rerun the next DEV mock sync.",
				Actions: []onboardingWorkflowAction{{
					Label:      "Open mock HR source record",
					Resolution: "Enter the missing employment field and confirm the source record is active.",
					System:     "Escape",
					Href:       mockWorkflowHref("escape", "hr-intake-evan-ruiz"),
				}},
			}},
		},
		{
			ID: "mika-ito", Kind: "seed", DateAdded: "May 3, 2025", DateAddedReason: "First Escape import", StartDate: "May 13, 2025", Person: "Mika Ito", Site: "Desert View", CurrentStep: "Ready", IssueAction: "No blockers", WorkflowStatus: "Ready", AssignedEmail: "mika.ito@wusd.org", IncidentIQ: "Ready for baseline provisioning. No external follow-up is currently required.", AeriesTicket: "IT-13002 Ready", VerkadaTicket: "MOT-4441 Ready",
			WorkflowSteps: []onboardingWorkflowStep{{Name: "Baseline readiness", Status: "Ready", Detail: "All required context is present. No user action is required."}},
		},
	}
}

func formatOnboardingDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("Jan 2, 2006")
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
