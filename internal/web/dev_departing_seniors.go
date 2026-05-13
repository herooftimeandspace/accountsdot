package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var devDepartingSeniorsStore = newDevDepartingSeniorsStore()

type departingSeniorsPagePayload struct {
	PageID      string                      `json:"page_id"`
	Persona     devPersona                  `json:"persona"`
	Shell       devShellPayload             `json:"shell"`
	GeneratedAt string                      `json:"generated_at"`
	Page        departingSeniorsPageContent `json:"page"`
}

type departingSeniorsPageContent struct {
	Title          string                      `json:"title"`
	Description    string                      `json:"description"`
	LastRefreshed  string                      `json:"last_refreshed"`
	SchoolYear     string                      `json:"school_year"`
	GraduationYear string                      `json:"graduation_year"`
	CanManage      bool                        `json:"can_manage"`
	Rows           []departingSeniorRowPayload `json:"rows"`
}

type departingSeniorRowPayload struct {
	ID                 string                         `json:"id"`
	FirstName          string                         `json:"first_name"`
	LastName           string                         `json:"last_name"`
	DisplayName        string                         `json:"display_name"`
	Email              string                         `json:"email"`
	StudentID          string                         `json:"student_id"`
	SiteID             string                         `json:"site_id"`
	Site               string                         `json:"site"`
	GraduationYear     string                         `json:"graduation_year"`
	EndDate            string                         `json:"end_date"`
	EndDateSource      string                         `json:"end_date_source"`
	Status             string                         `json:"status"`
	OutstandingDevices []departingSeniorDevicePayload `json:"outstanding_devices"`
	CanOverrideEndDate bool                           `json:"can_override_end_date"`
	CanDeprovision     bool                           `json:"can_deprovision"`
	Deprovisioned      bool                           `json:"deprovisioned"`
	Notes              []string                       `json:"notes,omitempty"`
}

type departingSeniorDevicePayload struct {
	AssetID string `json:"asset_id"`
	Serial  string `json:"serial"`
	Type    string `json:"type"`
}

type departingSeniorsEndDateRequest struct {
	EndDate string `json:"end_date"`
}

type departingSeniorsEndDateResponse struct {
	Row departingSeniorRowPayload `json:"row"`
}

type departingSeniorsDeprovisionResponse struct {
	Removed bool                       `json:"removed"`
	Row     *departingSeniorRowPayload `json:"row,omitempty"`
}

type devDepartingSeniorsStoreState struct {
	mu            sync.Mutex
	endDates      map[string]string
	deprovisioned map[string]bool
}

type departingSeniorSeedRecord struct {
	ID                 string
	FirstName          string
	LastName           string
	Email              string
	StudentID          string
	SiteID             string
	Site               string
	GraduationYear     string
	EndDate            string
	OutstandingDevices []departingSeniorDevicePayload
}

// newDevDepartingSeniorsStore builds the value used by internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func newDevDepartingSeniorsStore() *devDepartingSeniorsStoreState {
	return &devDepartingSeniorsStoreState{
		endDates:      map[string]string{},
		deprovisioned: map[string]bool{},
	}
}

// handleDevDepartingSeniorsPage handles the request path for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleDevDepartingSeniorsPage(w http.ResponseWriter, r *http.Request) {
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
	if !canUseDepartingSeniors(config) || !routeAllowed(r.Context(), config, devDepartingSeniorsRoute) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Departing Seniors is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	now := time.Now().UTC()
	graduationYear := currentSeniorGraduationYear(now)
	writeJSON(w, http.StatusOK, departingSeniorsPagePayload{
		PageID:      "departing-seniors",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: departingSeniorsPageContent{
			Title:          "Departing Seniors",
			Description:    "Current senior class account retirement and outstanding IncidentIQ device review.",
			LastRefreshed:  "Last refreshed:\nMay 8, 2026 9:00 AM PT",
			SchoolYear:     currentSchoolYearLabel(now),
			GraduationYear: graduationYear,
			CanManage:      canUseDepartingSeniors(config),
			Rows:           devDepartingSeniorsStore.rows(graduationYear),
		},
	})
}

// handleDevDepartingSeniorRecord handles the request path for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func handleDevDepartingSeniorRecord(w http.ResponseWriter, r *http.Request) {
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
	if !canUseDepartingSeniors(config) || !routeAllowed(r.Context(), config, devDepartingSeniorsRoute) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "This persona cannot update departing senior records.",
		})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dev/departing-seniors/records/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	switch parts[1] {
	case "end-date":
		if r.Method != http.MethodPut {
			http.NotFound(w, r)
			return
		}
		var request departingSeniorsEndDateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_json",
				"message": "Request body must be valid JSON.",
			})
			return
		}
		row, status, errors := devDepartingSeniorsStore.updateEndDate(parts[0], request.EndDate)
		if status != http.StatusOK {
			body := map[string]any{
				"code":    "departing_senior_end_date_rejected",
				"message": "The departing senior end date could not be updated.",
			}
			if len(errors) > 0 {
				body["errors"] = errors
			}
			writeJSON(w, status, body)
			return
		}
		writeJSON(w, http.StatusOK, departingSeniorsEndDateResponse{Row: row})
	case "deprovision":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		row, removed, status := devDepartingSeniorsStore.deprovision(parts[0])
		if status != http.StatusOK {
			writeJSON(w, status, map[string]any{
				"code":    "departing_senior_deprovision_rejected",
				"message": "The departing senior could not be deprovisioned.",
			})
			return
		}
		response := departingSeniorsDeprovisionResponse{Removed: removed}
		if !removed {
			response.Row = &row
		}
		writeJSON(w, http.StatusOK, response)
	default:
		http.NotFound(w, r)
	}
}

// canUseDepartingSeniors resolves decision data for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func canUseDepartingSeniors(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "device_wrangler"
}

// rows documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (s *devDepartingSeniorsStoreState) rows(graduationYear string) []departingSeniorRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := make([]departingSeniorRowPayload, 0, len(devDepartingSeniorSeedRecords))
	for _, record := range devDepartingSeniorSeedRecords {
		if record.GraduationYear != graduationYear {
			continue
		}
		if s.deprovisioned[record.ID] && len(record.OutstandingDevices) == 0 {
			continue
		}
		rows = append(rows, s.rowPayloadLocked(record))
	}
	return rows
}

// updateEndDate documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (s *devDepartingSeniorsStoreState) updateEndDate(id string, value string) (departingSeniorRowPayload, int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := findDepartingSeniorSeedRecord(id)
	if !ok {
		return departingSeniorRowPayload{}, http.StatusNotFound, nil
	}
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		delete(s.endDates, id)
		return s.rowPayloadLocked(record), http.StatusOK, nil
	}
	if _, err := time.ParseInLocation("2006-01-02", normalized, onboardingTimeLocation()); err != nil {
		return departingSeniorRowPayload{}, http.StatusBadRequest, map[string]string{
			"end_date": "Use YYYY-MM-DD.",
		}
	}
	s.endDates[id] = normalized
	return s.rowPayloadLocked(record), http.StatusOK, nil
}

// deprovision documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (s *devDepartingSeniorsStoreState) deprovision(id string) (departingSeniorRowPayload, bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := findDepartingSeniorSeedRecord(id)
	if !ok {
		return departingSeniorRowPayload{}, false, http.StatusNotFound
	}
	s.deprovisioned[id] = true
	removed := len(record.OutstandingDevices) == 0
	if removed {
		return departingSeniorRowPayload{}, true, http.StatusOK
	}
	return s.rowPayloadLocked(record), false, http.StatusOK
}

// rowPayloadLocked documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (s *devDepartingSeniorsStoreState) rowPayloadLocked(record departingSeniorSeedRecord) departingSeniorRowPayload {
	endDate := record.EndDate
	endDateSource := "Aeries senior class default"
	if override, ok := s.endDates[record.ID]; ok {
		endDate = override
		endDateSource = "Local override"
	}
	deprovisioned := s.deprovisioned[record.ID]
	status := "Ready"
	notes := []string{}
	switch {
	case deprovisioned && len(record.OutstandingDevices) > 0:
		status = "Device return required"
		notes = append(notes, "The account is deprovisioned, but the row remains until IncidentIQ shows no outstanding assigned devices.")
	case len(record.OutstandingDevices) > 0:
		status = "Device return required"
		notes = append(notes, "Outstanding IncidentIQ devices must be returned before this student leaves the list after deprovisioning.")
	}

	return departingSeniorRowPayload{
		ID:                 record.ID,
		FirstName:          record.FirstName,
		LastName:           record.LastName,
		DisplayName:        strings.TrimSpace(record.FirstName + " " + record.LastName),
		Email:              record.Email,
		StudentID:          record.StudentID,
		SiteID:             record.SiteID,
		Site:               record.Site,
		GraduationYear:     record.GraduationYear,
		EndDate:            endDate,
		EndDateSource:      endDateSource,
		Status:             status,
		OutstandingDevices: append([]departingSeniorDevicePayload(nil), record.OutstandingDevices...),
		CanOverrideEndDate: true,
		CanDeprovision:     !deprovisioned,
		Deprovisioned:      deprovisioned,
		Notes:              notes,
	}
}

// currentSeniorGraduationYear documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func currentSeniorGraduationYear(now time.Time) string {
	year := now.Year()
	if now.Month() >= time.August {
		year++
	}
	return strconv.Itoa(year)
}

// currentSchoolYearLabel documents the data flow for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func currentSchoolYearLabel(now time.Time) string {
	year := now.Year()
	if now.Month() >= time.August {
		return strconv.Itoa(year) + "-" + strconv.Itoa(year+1)
	}
	return strconv.Itoa(year-1) + "-" + strconv.Itoa(year)
}

// findDepartingSeniorSeedRecord resolves decision data for internal/web/dev_departing_seniors.go. HTTP routes, DEV frontend APIs, or web tests reach this function; debug it by following the registered route, request method, persona checks, and JSON response. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func findDepartingSeniorSeedRecord(id string) (departingSeniorSeedRecord, bool) {
	for _, record := range devDepartingSeniorSeedRecords {
		if record.ID == id {
			return record, true
		}
	}
	return departingSeniorSeedRecord{}, false
}

var devDepartingSeniorSeedRecords = []departingSeniorSeedRecord{
	{
		ID:             "senior-maya-chen",
		FirstName:      "Maya",
		LastName:       "Chen",
		Email:          "maya.chen@stu.wusd.org",
		StudentID:      "S-2026-10041",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2026",
		EndDate:        "2026-06-05",
	},
	{
		ID:             "senior-luis-alvarez",
		FirstName:      "Luis",
		LastName:       "Alvarez",
		Email:          "luis.alvarez@stu.wusd.org",
		StudentID:      "S-2026-10088",
		SiteID:         "franklin-ms",
		Site:           "Franklin MS",
		GraduationYear: "2026",
		EndDate:        "2026-06-05",
		OutstandingDevices: []departingSeniorDevicePayload{
			{AssetID: "IIQ-109284", Serial: "CB-CLA-24-18011", Type: "Chromebook"},
		},
	},
	{
		ID:             "senior-priya-shah",
		FirstName:      "Priya",
		LastName:       "Shah",
		Email:          "priya.shah@stu.wusd.org",
		StudentID:      "S-2026-10113",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2026",
		EndDate:        "2026-06-05",
		OutstandingDevices: []departingSeniorDevicePayload{
			{AssetID: "IIQ-110044", Serial: "CB-CLA-23-22109", Type: "Chromebook"},
			{AssetID: "IIQ-110045", Serial: "HOTSPOT-8830", Type: "Hotspot"},
		},
	},
	{
		ID:             "senior-jordan-miles",
		FirstName:      "Jordan",
		LastName:       "Miles",
		Email:          "jordan.miles@stu.wusd.org",
		StudentID:      "S-2026-10152",
		SiteID:         "desert-view",
		Site:           "Desert View",
		GraduationYear: "2026",
		EndDate:        "2026-06-05",
	},
}
