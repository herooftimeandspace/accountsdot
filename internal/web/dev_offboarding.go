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

type offboardingPageContent struct {
	Title             string                  `json:"title"`
	Description       string                  `json:"description"`
	LastRefreshed     string                  `json:"last_refreshed"`
	CanManageEndDates bool                    `json:"can_manage_end_dates"`
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

type devOffboardingStoreState struct {
	mu       sync.Mutex
	endDates map[string]string
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

// newDevOffboardingStore builds the value used by internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func newDevOffboardingStore() *devOffboardingStoreState {
	return &devOffboardingStoreState{endDates: map[string]string{}}
}

// handleDevOffboardingPage handles the request path for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleDevOffboardingPage(w http.ResponseWriter, r *http.Request) {
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
			Description:       "Offboarding status by person across accounts, licenses, assets, and security review queues.",
			LastRefreshed:     "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			CanManageEndDates: canManageOffboardingEndDates(config),
			ShowEmployeeIDs:   canSeeOffboardingEmployeeIDs(config),
			SummaryCards: []summaryCardPayload{
				{Title: "Scheduled Leaves", Count: "58"},
				{Title: "Immediate Terms", Count: "9"},
				{Title: "Asset Retrieval", Count: "37"},
				{Title: "Security Risk", Count: "6"},
			},
			Rows: rows,
		},
	})
}

// handleDevOffboardingRecord handles the request path for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleDevOffboardingRecord(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() {
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

// canManageOffboardingEndDates resolves decision data for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func canManageOffboardingEndDates(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

// canSeeOffboardingEmployeeIDs resolves decision data for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func canSeeOffboardingEmployeeIDs(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "human_resources"
}

// rows documents the data flow for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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

// updateEndDate documents the data flow for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
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

// rowPayloadLocked documents the data flow for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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

// offboardingRecordVisible documents the data flow for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func offboardingRecordVisible(record offboardingSeedRecord, config devPersonaConfig) bool {
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

// findOffboardingSeedRecord resolves decision data for internal/web/dev_offboarding.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
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
