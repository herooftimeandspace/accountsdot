package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	devDepartingSeniorsStore     = newDevDepartingSeniorsStore()
	devDepartingSeniorsNow       = func() time.Time { return time.Now().UTC() }
	devDepartingSeniorsCutoff    = time.Month(time.August)
	devDepartingSeniorsCutoffDay = 31
)

type departingSeniorsPagePayload struct {
	PageID      string                      `json:"page_id"`
	Persona     devPersona                  `json:"persona"`
	Shell       devShellPayload             `json:"shell"`
	GeneratedAt string                      `json:"generated_at"`
	Page        departingSeniorsPageContent `json:"page"`
}

type departingSeniorsPageContent struct {
	Title             string                             `json:"title"`
	Description       string                             `json:"description"`
	LastRefreshed     string                             `json:"last_refreshed"`
	SchoolYear        string                             `json:"school_year"`
	SchoolYearOptions []departingSeniorsSchoolYearOption `json:"school_year_options"`
	GraduationYear    string                             `json:"graduation_year"`
	CanManage         bool                               `json:"can_manage"`
	Rows              []departingSeniorRowPayload        `json:"rows"`
}

type departingSeniorsSchoolYearOption struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	GraduationYear string `json:"graduation_year"`
	Current        bool   `json:"current"`
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
	SchoolYear         string                         `json:"school_year"`
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
	AssetID  string `json:"asset_id"`
	Serial   string `json:"serial"`
	Type     string `json:"type"`
	Domain   string `json:"domain,omitempty"`
	AssetURL string `json:"asset_url,omitempty"`
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
	mu                    sync.Mutex
	endDates              map[string]string
	clearedLocalOverrides map[string]bool
	deprovisioned         map[string]bool
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
	EndDateSource      string
	Deprovisioned      bool
	OutstandingDevices []departingSeniorDevicePayload
}

// newDevDepartingSeniorsStore creates the in-memory DEV state used by the
// Departing Seniors mock routes. The store holds only local override dates,
// explicit clear markers for seeded local overrides, and deprovision flags;
// live Aeries, Google, Zoom, and IncidentIQ records are not mutated by this
// slice.
func newDevDepartingSeniorsStore() *devDepartingSeniorsStoreState {
	return &devDepartingSeniorsStoreState{
		endDates:              map[string]string{},
		clearedLocalOverrides: map[string]bool{},
		deprovisioned:         map[string]bool{},
	}
}

// ResetDevDepartingSeniorsStateForTest restores the package-level DEV store,
// deterministic clock, and cutoff defaults used by web tests. Production code
// does not call it; tests use it so retained-year fixture expectations do not
// leak between subtests or drift with the wall clock.
func ResetDevDepartingSeniorsStateForTest() {
	devDepartingSeniorsStore = newDevDepartingSeniorsStore()
	devDepartingSeniorsNow = func() time.Time { return time.Now().UTC() }
	devDepartingSeniorsCutoff = time.August
	devDepartingSeniorsCutoffDay = 31
}

// SetDevDepartingSeniorsClockForTest pins the DEV page clock for retained-year
// tests. The returned cleanup function restores the default wall-clock source
// and keeps handler tests from depending on the date when the suite runs.
func SetDevDepartingSeniorsClockForTest(now time.Time) func() {
	previous := devDepartingSeniorsNow
	devDepartingSeniorsNow = func() time.Time { return now.UTC() }
	return func() {
		devDepartingSeniorsNow = previous
	}
}

// handleDevDepartingSeniorsPage serves the DEV JSON payload consumed by
// DepartingSeniorsPage. It requires a logged-in IT Admin or Device Wrangler,
// accepts an optional school_year query value, and returns only the current
// senior year plus four retained previous senior years.
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

	now := devDepartingSeniorsNow()
	schoolYearOptions := departingSeniorsSchoolYearOptions(now)
	selectedSchoolYear := selectedDepartingSeniorsSchoolYear(r.URL.Query().Get("school_year"), schoolYearOptions)
	writeJSON(w, http.StatusOK, departingSeniorsPagePayload{
		PageID:      "departing-seniors",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: now.Format(time.RFC3339),
		Page: departingSeniorsPageContent{
			Title:             "Departing Seniors",
			Description:       "Current senior class account retirement and outstanding IncidentIQ device review.",
			LastRefreshed:     "Last refreshed:\nMay 8, 2026 9:00 AM PT",
			SchoolYear:        selectedSchoolYear.ID,
			SchoolYearOptions: schoolYearOptions,
			GraduationYear:    selectedSchoolYear.GraduationYear,
			CanManage:         canUseDepartingSeniors(config),
			Rows:              devDepartingSeniorsStore.rows(selectedSchoolYear.GraduationYear, now),
		},
	})
}

// handleDevDepartingSeniorRecord applies IT/Admin or Device Wrangler mock
// mutations for one departing-senior row. End-date updates and deprovision
// requests change only devDepartingSeniorsStore so the UI can model future
// operator behavior without touching live providers.
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

// canUseDepartingSeniors centralizes the persona gate shared by the page-load
// and row-mutation routes. Departing Seniors is intentionally limited to IT
// Admin and Device Wrangler in the DEV frontend slice.
func canUseDepartingSeniors(config devPersonaConfig) bool {
	return config.Persona.ID == "it_admin" || config.Persona.ID == "device_wrangler"
}

// rows builds the visible DEV table for one graduation year at the current
// retained-year clock. Deprovisioned or expired clean rows are suppressed, but
// rows with outstanding devices remain visible because the remaining task is
// IncidentIQ device recovery rather than account access.
func (s *devDepartingSeniorsStoreState) rows(graduationYear string, now time.Time) []departingSeniorRowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := make([]departingSeniorRowPayload, 0, len(devDepartingSeniorSeedRecords))
	for _, record := range devDepartingSeniorSeedRecords {
		if record.GraduationYear != graduationYear {
			continue
		}
		if !s.visibleLocked(record, now) {
			continue
		}
		rows = append(rows, s.rowPayloadLocked(record, now))
	}
	return rows
}

// updateEndDate stores or clears a local DEV override for a senior's planned
// account-retirement date. The successful response returns the refreshed row;
// invalid date input returns a field error and leaves the mock store unchanged.
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
		s.clearedLocalOverrides[id] = true
		return s.rowPayloadLocked(record, devDepartingSeniorsNow()), http.StatusOK, nil
	}
	if _, err := time.ParseInLocation("2006-01-02", normalized, onboardingTimeLocation()); err != nil {
		return departingSeniorRowPayload{}, http.StatusBadRequest, map[string]string{
			"end_date": "Use YYYY-MM-DD.",
		}
	}
	s.endDates[id] = normalized
	delete(s.clearedLocalOverrides, id)
	return s.rowPayloadLocked(record, devDepartingSeniorsNow()), http.StatusOK, nil
}

// deprovision marks a DEV senior account as deprovisioned. Rows with no
// outstanding devices are removed from the visible payload, while rows with
// devices stay visible so operators can keep working the return queue.
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
	return s.rowPayloadLocked(record, devDepartingSeniorsNow()), false, http.StatusOK
}

// rowPayloadLocked converts one immutable seed record plus local DEV mutation
// state into the JSON row consumed by the React table and drawer. Callers must
// hold s.mu so date overrides and deprovision flags are read consistently.
func (s *devDepartingSeniorsStoreState) rowPayloadLocked(record departingSeniorSeedRecord, now time.Time) departingSeniorRowPayload {
	endDate := record.EndDate
	endDateSource := "Aeries senior class default"
	if record.EndDateSource != "" {
		endDateSource = record.EndDateSource
	}
	if override, ok := s.endDates[record.ID]; ok {
		endDate = override
		endDateSource = "Local override"
	} else if s.clearedLocalOverrides[record.ID] && record.EndDateSource == "Local override" {
		endDateSource = "Aeries senior class default"
	}
	deprovisioned := s.isDeprovisionedLocked(record, now)
	status := "Ready"
	notes := []string{}
	switch {
	case deprovisioned && len(record.OutstandingDevices) > 0:
		status = "Device return required"
		notes = append(notes, "The account is deprovisioned, but the row remains until IncidentIQ shows no outstanding assigned devices.")
	case deprovisioned:
		status = "Account deprovisioned"
	case len(record.OutstandingDevices) > 0:
		status = "Device return required"
		notes = append(notes, "Outstanding IncidentIQ devices must be returned before this student leaves the list after deprovisioning.")
	case s.hasLocalOverrideLocked(record) && s.hasValidLocalOverrideLocked(record, now):
		status = "Access retained by local override"
		notes = append(notes, "A local override keeps the account active until the listed end date.")
	case !pastSeniorCutoffForGraduationYear(record.GraduationYear, now):
		status = "Suppressed by senior exception"
		notes = append(notes, "Student identity access is intentionally retained through the configured senior cutoff day.")
	}
	devices := make([]departingSeniorDevicePayload, 0, len(record.OutstandingDevices))
	for _, device := range record.OutstandingDevices {
		if device.Domain == "" && device.AssetID != "" {
			device.Domain = "wusd.incidentiq.com"
		}
		device.AssetURL = incidentIQAssetURL(device.Domain, device.AssetID)
		devices = append(devices, device)
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
		SchoolYear:         schoolYearLabelForGraduationYearString(record.GraduationYear),
		GraduationYear:     record.GraduationYear,
		EndDate:            endDate,
		EndDateSource:      endDateSource,
		Status:             status,
		OutstandingDevices: devices,
		CanOverrideEndDate: true,
		CanDeprovision:     !deprovisioned,
		Deprovisioned:      deprovisioned,
		Notes:              notes,
	}
}

// visibleLocked applies retained-year visibility rules after the requested
// school-year window has already been validated. Clean deprovisioned rows and
// expired previous-year local overrides disappear unless assigned devices still
// need return; current-year rows stay visible until cutoff or deprovisioning.
func (s *devDepartingSeniorsStoreState) visibleLocked(record departingSeniorSeedRecord, now time.Time) bool {
	if len(record.OutstandingDevices) > 0 {
		return true
	}
	if s.isDeprovisionedLocked(record, now) {
		return false
	}
	if s.hasLocalOverrideLocked(record) && s.hasValidLocalOverrideLocked(record, now) {
		return true
	}
	return !pastSeniorCutoffForGraduationYear(record.GraduationYear, now)
}

// isDeprovisionedLocked treats explicit DEV clicks, fixture state, and the
// post-cutoff current senior rule as the same account-retirement outcome for
// payload generation. Device rows remain visible separately through
// visibleLocked so operators can finish IncidentIQ recovery.
func (s *devDepartingSeniorsStoreState) isDeprovisionedLocked(record departingSeniorSeedRecord, now time.Time) bool {
	if s.deprovisioned[record.ID] || record.Deprovisioned {
		return true
	}
	return pastSeniorCutoffForGraduationYear(record.GraduationYear, now) && !(s.hasLocalOverrideLocked(record) && s.hasValidLocalOverrideLocked(record, now))
}

// hasLocalOverrideLocked distinguishes explicit local override dates from
// Aeries default end dates. Only local overrides can intentionally retain
// previous-year access after the normal senior cutoff.
func (s *devDepartingSeniorsStoreState) hasLocalOverrideLocked(record departingSeniorSeedRecord) bool {
	if _, ok := s.endDates[record.ID]; ok {
		return true
	}
	if s.clearedLocalOverrides[record.ID] {
		return false
	}
	return record.EndDateSource == "Local override"
}

// hasValidLocalOverrideLocked reports whether a local end-date override still
// protects a retained senior account today. Invalid fixture dates are treated
// as expired so tests fail closed rather than preserving stale access.
func (s *devDepartingSeniorsStoreState) hasValidLocalOverrideLocked(record departingSeniorSeedRecord, now time.Time) bool {
	value := record.EndDate
	if override, ok := s.endDates[record.ID]; ok {
		value = override
	}
	if value == "" {
		return false
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, onboardingTimeLocation())
	if err != nil {
		return false
	}
	current := now.In(onboardingTimeLocation())
	today := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, onboardingTimeLocation())
	return !parsed.Before(today)
}

// isCurrentSeniorGraduationYear compares fixture rows to the Aeries-derived
// senior cohort for the provided clock.
func isCurrentSeniorGraduationYear(graduationYear string, now time.Time) bool {
	return graduationYear == currentSeniorGraduationYear(now)
}

// pastSeniorCutoffForGraduationYear returns true on the calendar day after the
// configured cutoff for the row's graduation year. Access is retained through
// the end of the cutoff day itself, and previous cohorts are therefore treated
// as deprovisioned unless an override or device-return row keeps them visible.
func pastSeniorCutoffForGraduationYear(graduationYear string, now time.Time) bool {
	year, err := strconv.Atoi(graduationYear)
	if err != nil {
		return true
	}
	current := now.In(onboardingTimeLocation())
	cutoff := time.Date(year, devDepartingSeniorsCutoff, devDepartingSeniorsCutoffDay, 23, 59, 59, int(time.Second-time.Nanosecond), onboardingTimeLocation())
	return current.After(cutoff)
}

// currentSeniorGraduationYear derives the current senior graduation year from
// the active Aeries-style school-year boundary. August starts the next school
// year, so seniors after that boundary graduate in the following calendar year.
func currentSeniorGraduationYear(now time.Time) string {
	current := now.In(onboardingTimeLocation())
	year := current.Year()
	if current.Month() >= time.August {
		year++
	}
	return strconv.Itoa(year)
}

// currentSchoolYearLabel returns the Aeries-style school-year label that should
// be selected by default on the Departing Seniors page.
func currentSchoolYearLabel(now time.Time) string {
	current := now.In(onboardingTimeLocation())
	year := current.Year()
	if current.Month() >= time.August {
		return strconv.Itoa(year) + "-" + strconv.Itoa(year+1)
	}
	return strconv.Itoa(year-1) + "-" + strconv.Itoa(year)
}

// departingSeniorsSchoolYearOptions exposes the current senior year and four
// retained previous senior years for the page dropdown. The generated list is
// intentionally bounded so old senior cohorts cannot leak back into DEV API
// responses once they fall outside the documented retention window.
func departingSeniorsSchoolYearOptions(now time.Time) []departingSeniorsSchoolYearOption {
	currentGraduationYear, _ := strconv.Atoi(currentSeniorGraduationYear(now))
	options := make([]departingSeniorsSchoolYearOption, 0, 5)
	for graduationYear := currentGraduationYear; graduationYear >= currentGraduationYear-4; graduationYear-- {
		options = append(options, departingSeniorsSchoolYearOption{
			ID:             schoolYearLabelForGraduationYear(graduationYear),
			Label:          schoolYearLabelForGraduationYear(graduationYear),
			GraduationYear: strconv.Itoa(graduationYear),
			Current:        graduationYear == currentGraduationYear,
		})
	}
	return options
}

// selectedDepartingSeniorsSchoolYear validates a requested school-year id
// against the retained option list. Unknown or expired values fall back to the
// current Aeries-derived senior year rather than returning stale cohorts.
func selectedDepartingSeniorsSchoolYear(requested string, options []departingSeniorsSchoolYearOption) departingSeniorsSchoolYearOption {
	for _, option := range options {
		if option.ID == requested {
			return option
		}
	}
	for _, option := range options {
		if option.Current {
			return option
		}
	}
	return options[0]
}

// schoolYearLabelForGraduationYear formats the district school year attached
// to a graduating senior cohort. A 2026 senior belongs to school year
// 2025-2026.
func schoolYearLabelForGraduationYear(graduationYear int) string {
	return strconv.Itoa(graduationYear-1) + "-" + strconv.Itoa(graduationYear)
}

// schoolYearLabelForGraduationYearString is the defensive string wrapper used
// when seed rows are converted to payload rows. Bad fixture data returns an
// empty label so tests can catch missing school-year context.
func schoolYearLabelForGraduationYearString(graduationYear string) string {
	parsed, err := strconv.Atoi(graduationYear)
	if err != nil {
		return ""
	}
	return schoolYearLabelForGraduationYear(parsed)
}

// incidentIQAssetURL builds the DEV asset deep link shown in the table and
// drawer. Empty asset ids or domains return an empty URL so the frontend can
// render plain text instead of an invalid external link.
func incidentIQAssetURL(domain string, assetID string) string {
	if strings.TrimSpace(domain) == "" || strings.TrimSpace(assetID) == "" {
		return ""
	}
	return "https://" + strings.TrimSpace(domain) + "/agent/assets/" + strings.TrimSpace(assetID)
}

// findDepartingSeniorSeedRecord looks up a seed row for row-level DEV
// mutations. It searches all retained and fixture-only seed records so mutation
// routes can return 404 only when the row id is truly unknown.
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
		ID:             "senior-sam-rivera",
		FirstName:      "Sam",
		LastName:       "Rivera",
		Email:          "sam.rivera@stu.wusd.org",
		StudentID:      "S-2026-10133",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2026",
		EndDate:        "2026-06-05",
		OutstandingDevices: []departingSeniorDevicePayload{
			{Serial: "CB-CLA-24-plain", Type: "Chromebook"},
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
	{
		ID:             "senior-ava-rodriguez-2025",
		FirstName:      "Ava",
		LastName:       "Rodriguez",
		Email:          "ava.rodriguez@stu.wusd.org",
		StudentID:      "S-2025-09017",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2025",
		EndDate:        "2025-06-06",
		Deprovisioned:  true,
		OutstandingDevices: []departingSeniorDevicePayload{
			{AssetID: "IIQ-100512", Serial: "CB-CLA-22-10255", Type: "Chromebook"},
		},
	},
	{
		ID:             "senior-emma-nguyen-2025-override",
		FirstName:      "Emma",
		LastName:       "Nguyen",
		Email:          "emma.nguyen@stu.wusd.org",
		StudentID:      "S-2025-09031",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2025",
		EndDate:        "2026-12-31",
		EndDateSource:  "Local override",
	},
	{
		ID:             "senior-ben-owens-2025-expired-override",
		FirstName:      "Ben",
		LastName:       "Owens",
		Email:          "ben.owens@stu.wusd.org",
		StudentID:      "S-2025-09044",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2025",
		EndDate:        "2026-01-15",
		EndDateSource:  "Local override",
	},
	{
		ID:             "senior-noah-kim-2024",
		FirstName:      "Noah",
		LastName:       "Kim",
		Email:          "noah.kim@stu.wusd.org",
		StudentID:      "S-2024-08444",
		SiteID:         "desert-view",
		Site:           "Desert View",
		GraduationYear: "2024",
		EndDate:        "2024-06-07",
		Deprovisioned:  true,
	},
	{
		ID:             "senior-zoe-patel-2021",
		FirstName:      "Zoe",
		LastName:       "Patel",
		Email:          "zoe.patel@stu.wusd.org",
		StudentID:      "S-2021-07111",
		SiteID:         "clover-hs",
		Site:           "Clover HS",
		GraduationYear: "2021",
		EndDate:        "2021-06-04",
	},
}
