package web

import (
	"net/http"
	"strconv"
	"time"
)

type merakiLastSeenPagePayload struct {
	PageID      string                    `json:"page_id"`
	Persona     devPersona                `json:"persona"`
	Shell       devShellPayload           `json:"shell"`
	GeneratedAt string                    `json:"generated_at"`
	Page        merakiLastSeenPageContent `json:"page"`
}

type merakiLastSeenPageContent struct {
	Title         string                       `json:"title"`
	Description   string                       `json:"description"`
	HelpText      string                       `json:"help_text"`
	LastRefreshed string                       `json:"last_refreshed"`
	SummaryCards  []summaryCardPayload         `json:"summary_cards"`
	Rows          []merakiLastSeenRowPayload   `json:"rows"`
	Filters       []merakiLastSeenFilterOption `json:"filters"`
}

type merakiLastSeenFilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type merakiLastSeenRowPayload struct {
	ID                  string   `json:"id"`
	Student             string   `json:"student,omitempty"`
	StudentID           string   `json:"student_id,omitempty"`
	Device              string   `json:"device"`
	SerialNumber        string   `json:"serial_number"`
	AssetTag            string   `json:"asset_tag"`
	MACAddress          string   `json:"mac_address"`
	Hostname            string   `json:"hostname"`
	SiteID              string   `json:"site_id"`
	Site                string   `json:"site"`
	LastSeen            string   `json:"last_seen"`
	LastSeenAt          string   `json:"last_seen_at"`
	AssignmentType      string   `json:"assignment_type"`
	AssignmentTypeLabel string   `json:"assignment_type_label"`
	MatchState          string   `json:"match_state"`
	MatchConfidence     string   `json:"match_confidence"`
	SourceSystems       []string `json:"source_systems"`
	MatchExplanation    string   `json:"match_explanation"`
	ReviewReason        string   `json:"review_reason,omitempty"`
}

type merakiLastSeenSeedRecord struct {
	ID                  string
	Student             string
	StudentID           string
	Device              string
	SerialNumber        string
	AssetTag            string
	MACAddress          string
	Hostname            string
	SiteID              string
	Site                string
	LastSeen            string
	LastSeenAt          string
	AssignmentType      string
	AssignmentTypeLabel string
	MatchState          string
	MatchConfidence     string
	SourceSystems       []string
	MatchExplanation    string
	ReviewReason        string
}

// handleDevMerakiLastSeenPage serves the read-only DEV Meraki last-seen
// dashboard. The handler enforces the same route, persona, feature-flag, and
// site-scope boundary that production handlers must preserve before exposing
// provider-backed Meraki, IncidentIQ, or Google device payloads.
func handleDevMerakiLastSeenPage(w http.ResponseWriter, r *http.Request) {
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
	if !routeAllowed(r.Context(), config, devMerakiLastSeenRoute) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Meraki Last Seen is available only to IT Admin, Site Admin, and Device Wrangler roles.",
			"persona": config.Persona,
		})
		return
	}

	rows := merakiLastSeenRowsForPersona(config)
	writeJSON(w, http.StatusOK, merakiLastSeenPagePayload{
		PageID:      "meraki-last-seen",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Page: merakiLastSeenPageContent{
			Title:         "Meraki Last Seen",
			Description:   "Student-assigned and classroom spare devices matched across Meraki, IncidentIQ, and Google.",
			HelpText:      "Ambiguous devices stay reviewable until IncidentIQ or Google assignment metadata clearly identifies student assignment or spare-pool status.",
			LastRefreshed: "Last refreshed:\nMay 3, 2026 9:00 AM PT",
			SummaryCards: []summaryCardPayload{
				{Title: "Visible Devices", Count: countString(len(rows))},
				{Title: "Classroom Spares", Count: countString(countMerakiLastSeenAssignmentType(rows, "classroom_spare"))},
				{Title: "Needs Review", Count: countString(countMerakiLastSeenMatchState(rows, "ambiguous"))},
			},
			Filters: []merakiLastSeenFilterOption{
				{Value: "all", Label: "All devices"},
				{Value: "assigned_student", Label: "Assigned student devices"},
				{Value: "classroom_spare", Label: "Classroom spares"},
			},
			Rows: rows,
		},
	})
}

func merakiLastSeenRowsForPersona(config devPersonaConfig) []merakiLastSeenRowPayload {
	visibleSiteIDs := make(map[string]bool, len(config.VisibleSites))
	for _, site := range config.VisibleSites {
		visibleSiteIDs[site.ID] = true
	}
	rows := []merakiLastSeenRowPayload{}
	for _, seed := range merakiLastSeenSeedRows() {
		if config.Persona.ID != "it_admin" && !visibleSiteIDs[seed.SiteID] {
			continue
		}
		rows = append(rows, merakiLastSeenRowPayload(seed))
	}
	return rows
}

func countMerakiLastSeenAssignmentType(rows []merakiLastSeenRowPayload, assignmentType string) int {
	count := 0
	for _, row := range rows {
		if row.AssignmentType == assignmentType {
			count++
		}
	}
	return count
}

func countMerakiLastSeenMatchState(rows []merakiLastSeenRowPayload, matchState string) int {
	count := 0
	for _, row := range rows {
		if row.MatchState == matchState {
			count++
		}
	}
	return count
}

func countString(count int) string {
	return strconv.Itoa(count)
}

func merakiLastSeenSeedRows() []merakiLastSeenSeedRecord {
	return []merakiLastSeenSeedRecord{
		{
			ID:                  "meraki-cla-maria-nguyen",
			Student:             "Maria Nguyen",
			StudentID:           "3501187",
			Device:              "Chromebook CLA-24-19912",
			SerialNumber:        "CLA-24-19912",
			AssetTag:            "IIQ-CLA-19912",
			MACAddress:          "68:3A:1E:44:19:12",
			Hostname:            "stu-mnguyen-19912",
			SiteID:              "clover-hs",
			Site:                "Clover High School",
			LastSeen:            "May 3, 2026 8:42 AM PT",
			LastSeenAt:          "2026-05-03T15:42:00Z",
			AssignmentType:      "assigned_student",
			AssignmentTypeLabel: "Assigned student device",
			MatchState:          "matched",
			MatchConfidence:     "High",
			SourceSystems:       []string{"Meraki", "IncidentIQ", "Google"},
			MatchExplanation:    "Serial, MAC address, and Google assigned-user metadata all point to the same active student assignment.",
		},
		{
			ID:                  "meraki-cla-room-b204-spare",
			Device:              "Chromebook CLA-SPARE-B204",
			SerialNumber:        "CLA-24-88001",
			AssetTag:            "IIQ-CLA-SPARE-B204",
			MACAddress:          "68:3A:1E:88:00:01",
			Hostname:            "cla-b204-spare-01",
			SiteID:              "clover-hs",
			Site:                "Clover High School",
			LastSeen:            "May 3, 2026 7:18 AM PT",
			LastSeenAt:          "2026-05-03T14:18:00Z",
			AssignmentType:      "classroom_spare",
			AssignmentTypeLabel: "Classroom spare / spare pool",
			MatchState:          "matched",
			MatchConfidence:     "High",
			SourceSystems:       []string{"Meraki", "IncidentIQ"},
			MatchExplanation:    "IncidentIQ location and asset owner metadata classify this device as a classroom spare; no student owner is required.",
		},
		{
			ID:                  "meraki-fms-omar-castillo",
			Student:             "Omar Castillo",
			StudentID:           "3508449",
			Device:              "Chromebook FMS-24-32108",
			SerialNumber:        "FMS-24-32108",
			AssetTag:            "IIQ-FMS-32108",
			MACAddress:          "68:3A:1E:32:10:08",
			Hostname:            "stu-ocastillo-32108",
			SiteID:              "franklin-ms",
			Site:                "Franklin Middle School",
			LastSeen:            "May 3, 2026 8:08 AM PT",
			LastSeenAt:          "2026-05-03T15:08:00Z",
			AssignmentType:      "assigned_student",
			AssignmentTypeLabel: "Assigned student device",
			MatchState:          "matched",
			MatchConfidence:     "High",
			SourceSystems:       []string{"Meraki", "IncidentIQ", "Google"},
			MatchExplanation:    "IncidentIQ active checkout and Google device assignment agree on the student and serial number.",
		},
		{
			ID:                  "meraki-fms-library-spare",
			Device:              "Chromebook FMS-LIB-SPARE-07",
			SerialNumber:        "FMS-24-44007",
			AssetTag:            "IIQ-FMS-SPARE-07",
			MACAddress:          "68:3A:1E:44:00:07",
			Hostname:            "fms-library-spare-07",
			SiteID:              "franklin-ms",
			Site:                "Franklin Middle School",
			LastSeen:            "May 3, 2026 8:55 AM PT",
			LastSeenAt:          "2026-05-03T15:55:00Z",
			AssignmentType:      "classroom_spare",
			AssignmentTypeLabel: "Classroom spare / spare pool",
			MatchState:          "matched",
			MatchConfidence:     "Medium",
			SourceSystems:       []string{"Meraki", "IncidentIQ"},
			MatchExplanation:    "Meraki last-seen data is matched to an IncidentIQ library spare asset with no single student owner.",
		},
		{
			ID:                  "meraki-fms-ambiguous-checkout",
			Device:              "Chromebook FMS-24-55131",
			SerialNumber:        "FMS-24-55131",
			AssetTag:            "IIQ-FMS-55131",
			MACAddress:          "68:3A:1E:55:13:10",
			Hostname:            "fms-cart-ambiguous-55131",
			SiteID:              "franklin-ms",
			Site:                "Franklin Middle School",
			LastSeen:            "May 3, 2026 8:33 AM PT",
			LastSeenAt:          "2026-05-03T15:33:00Z",
			AssignmentType:      "ambiguous",
			AssignmentTypeLabel: "Ambiguous assignment",
			MatchState:          "ambiguous",
			MatchConfidence:     "Review",
			SourceSystems:       []string{"Meraki", "IncidentIQ", "Google"},
			MatchExplanation:    "Google reports a recent student association while IncidentIQ still marks the asset as spare-pool inventory.",
			ReviewReason:        "Confirm whether the device is actively assigned or should remain in classroom spare inventory.",
		},
	}
}
