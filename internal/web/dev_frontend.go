package web

import (
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"
)

const devSessionCookieName = "wizard_dev_session"

const (
	phoneDirectoryTypePerson        = "person"
	phoneDirectoryTypeCommonArea    = "common_area"
	phoneDirectoryTypeClassroomSLG  = "classroom_slg"
	phoneDirectoryTypeDepartmentSLG = "department_slg"
	phoneDirectoryTypeCallQueue     = "call_queue"
	phoneDirectoryTypeAutoAttendant = "auto_attendant"
)

var (
	devPhoneDirectoryRoutes = []string{
		"/phone-directory/by-person",
		"/phone-directory/by-room",
		"/phone-directory/by-department",
	}
	devSiteScopedRoutes = []string{
		"/student-data-cleanup",
		"/frequent-fliers",
		"/onboarding",
		"/offboarding",
		"/room-moves",
	}
	devITOnlyRoutes = []string{
		"/dashboard/it-admin",
		"/data-quality",
		"/reports",
		"/reports/sync-transparency",
		"/reports/ticketing-human-work",
		"/admin",
	}
)

type devPersona struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	DisplayName     string `json:"display_name"`
	Initials        string `json:"initials"`
	ProfilePhotoURL string `json:"profile_photo_url,omitempty"`
}

type devSiteContext struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type devPersonaConfig struct {
	Persona      devPersona
	LandingPath  string
	Allowed      []string
	Shell        devShellPayload
	DefaultSite  devSiteContext
	CurrentSite  devSiteContext
	VisibleSites []devSiteContext
}

type devSessionPayload struct {
	Environment     string          `json:"environment"`
	Authenticated   bool            `json:"authenticated"`
	Authorized      bool            `json:"authorized"`
	CurrentPersona  *devPersona     `json:"current_persona,omitempty"`
	Personas        []devPersona    `json:"personas"`
	LandingPath     string          `json:"landing_path,omitempty"`
	AllowedRoutes   []string        `json:"allowed_routes,omitempty"`
	Shell           devShellPayload `json:"shell,omitempty"`
	DefaultSiteID   string          `json:"default_site_id,omitempty"`
	DefaultSiteName string          `json:"default_site_name,omitempty"`
	CurrentSiteID   string          `json:"current_site_id,omitempty"`
	CurrentSiteName string          `json:"current_site_name,omitempty"`
}

type devLoginRequest struct {
	PersonaID string `json:"persona_id"`
}

type dataQualityPagePayload struct {
	PageID      string                    `json:"page_id"`
	Persona     devPersona                `json:"persona"`
	Shell       devShellPayload           `json:"shell"`
	Page        dataQualityContentPayload `json:"page"`
	Hotspots    map[string]hotspotPayload `json:"hotspots"`
	GeneratedAt string                    `json:"generated_at"`
}

type phoneDirectoryPagePayload struct {
	PageID      string                       `json:"page_id"`
	Persona     devPersona                   `json:"persona"`
	Shell       devShellPayload              `json:"shell"`
	Page        phoneDirectoryContentPayload `json:"page"`
	GeneratedAt string                       `json:"generated_at"`
}

type devShellPayload struct {
	ScopeTitle        string `json:"scope_title"`
	ScopeSubtitle     string `json:"scope_subtitle"`
	SearchPlaceholder string `json:"search_placeholder"`
	NotificationCount string `json:"notification_count"`
	PlatformStatus    string `json:"platform_status"`
}

type phoneDirectoryContentPayload struct {
	Mode            string                       `json:"mode"`
	Title           string                       `json:"title"`
	Description     string                       `json:"description"`
	LastRefreshed   string                       `json:"last_refreshed"`
	Query           string                       `json:"query"`
	CurrentSiteID   string                       `json:"current_site_id"`
	CurrentSiteName string                       `json:"current_site_name"`
	Results         []phoneDirectorySearchResult `json:"results"`
	SelectedResult  *phoneDirectorySearchResult  `json:"selected_result,omitempty"`
}

type dataQualityContentPayload struct {
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	LastRefreshed string                  `json:"last_refreshed"`
	RefreshLabel  string                  `json:"refresh_label"`
	SummaryCards  []summaryCardPayload    `json:"summary_cards"`
	RoutingCard   routingCardPayload      `json:"routing_card"`
	Queue         dataQualityQueuePayload `json:"queue"`
	RoutingRules  routingRulesPayload     `json:"routing_rules"`
}

type summaryCardPayload struct {
	Title string `json:"title"`
	Count string `json:"count"`
}

type routingCardPayload struct {
	Title    string `json:"title"`
	Headline string `json:"headline"`
	Body     string `json:"body"`
}

type dataQualityQueuePayload struct {
	Rows []dataQualityQueueRow `json:"rows"`
}

type dataQualityQueueRow struct {
	Issue      string `json:"issue"`
	Source     string `json:"source"`
	Owner      string `json:"owner"`
	Impact     string `json:"impact"`
	NextAction string `json:"next_action"`
}

type routingRulesPayload struct {
	Title              string               `json:"title"`
	Rules              []routingRulePayload `json:"rules"`
	PrimaryActionLabel string               `json:"primary_action_label"`
}

type routingRulePayload struct {
	Queue       string `json:"queue"`
	Description string `json:"description"`
}

type hotspotPayload struct {
	NodeID string `json:"node_id"`
	Label  string `json:"label"`
}

type phoneDirectorySearchResult struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	TypeLabel       string `json:"type_label"`
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	SiteID          string `json:"site_id"`
	SiteName        string `json:"site_name"`
	Role            string `json:"role,omitempty"`
	Department      string `json:"department,omitempty"`
	Location        string `json:"location,omitempty"`
	Email           string `json:"email,omitempty"`
	Phone           string `json:"phone,omitempty"`
	Extension       string `json:"extension,omitempty"`
	ExtensionLength int    `json:"extension_length"`
	ExtensionValid  bool   `json:"extension_valid"`
	Identifier      string `json:"identifier,omitempty"`
}

type devPhoneDirectoryEntry struct {
	ID              string
	Type            string
	TypeLabel       string
	Title           string
	Subtitle        string
	SiteID          string
	SiteName        string
	Role            string
	Department      string
	Location        string
	Email           string
	Phone           string
	Extension       string
	ExtensionLength int
	ExtensionValid  bool
	Identifier      string
	Searchable      []string
}

type phoneDirectorySearchMatch struct {
	Rank int
}

type rankedPhoneDirectoryResult struct {
	Result        phoneDirectorySearchResult
	SiteRank      int
	SiteOrder     int
	TypeRank      int
	MatchRank     int
	NormalizedKey string
}

var devSiteCatalog = map[string]devSiteContext{
	"district-office": {ID: "district-office", Name: "District Office"},
	"clover-hs":       {ID: "clover-hs", Name: "Clover High School"},
	"desert-view":     {ID: "desert-view", Name: "Desert View Elementary"},
	"highland-es":     {ID: "highland-es", Name: "Highland Elementary"},
	"franklin-ms":     {ID: "franklin-ms", Name: "Franklin Middle School"},
	"business-office": {ID: "business-office", Name: "Business Office"},
}

var devPersonaConfigs = map[string]devPersonaConfig{
	"it_admin": {
		Persona: devPersona{
			ID:          "it_admin",
			Label:       "IT Admin",
			DisplayName: "Alex Ramirez",
			Initials:    "AR",
		},
		LandingPath: "/dashboard/it-admin",
		Allowed: concatRoutes(
			[]string{
				"/dashboard/it-admin",
				"/dashboard/hr-lifecycle",
				"/dashboard/site-admin",
				"/my-profile",
			},
			devPhoneDirectoryRoutes,
			devSiteScopedRoutes,
			devITOnlyRoutes,
		),
		Shell: devShellPayload{
			ScopeTitle:        "District-wide",
			ScopeSubtitle:     "All Sites",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "3",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite: siteByID("clover-hs"),
		CurrentSite: siteByID("clover-hs"),
		VisibleSites: sitesByID(
			"clover-hs",
			"desert-view",
			"highland-es",
			"franklin-ms",
			"business-office",
			"district-office",
		),
	},
	"human_resources": {
		Persona: devPersona{
			ID:          "human_resources",
			Label:       "Human Resources",
			DisplayName: "Maria Torres",
			Initials:    "MT",
		},
		LandingPath: "/dashboard/hr-lifecycle",
		Allowed: append(
			[]string{
				"/dashboard/hr-lifecycle",
				"/my-profile",
				"/onboarding",
				"/offboarding",
			},
			devPhoneDirectoryRoutes...,
		),
		Shell: devShellPayload{
			ScopeTitle:        "District-wide",
			ScopeSubtitle:     "All Sites",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "2",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite: siteByID("district-office"),
		CurrentSite: siteByID("district-office"),
		VisibleSites: sitesByID(
			"district-office",
			"clover-hs",
			"desert-view",
			"highland-es",
			"franklin-ms",
			"business-office",
		),
	},
	"site_admin": {
		Persona: devPersona{
			ID:          "site_admin",
			Label:       "Site Admin",
			DisplayName: "Janelle Brooks",
			Initials:    "JB",
		},
		LandingPath: "/dashboard/site-admin",
		Allowed: append(
			[]string{
				"/dashboard/site-admin",
				"/my-profile",
				"/student-data-cleanup",
				"/frequent-fliers",
				"/onboarding",
				"/offboarding",
				"/room-moves",
			},
			devPhoneDirectoryRoutes...,
		),
		Shell: devShellPayload{
			ScopeTitle:        "Assigned site(s)",
			ScopeSubtitle:     "Scoped Access",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "2",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite: siteByID("clover-hs"),
		CurrentSite: siteByID("clover-hs"),
		VisibleSites: sitesByID(
			"clover-hs",
			"desert-view",
		),
	},
	"site_secretary": {
		Persona: devPersona{
			ID:          "site_secretary",
			Label:       "Site Secretary",
			DisplayName: "Lena Alvarez",
			Initials:    "LA",
		},
		LandingPath: "/phone-directory/by-room",
		Allowed: append(
			[]string{
				"/my-profile",
				"/student-data-cleanup",
				"/room-moves",
			},
			devPhoneDirectoryRoutes...,
		),
		Shell: devShellPayload{
			ScopeTitle:        "Assigned site(s)",
			ScopeSubtitle:     "Scoped Access",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "1",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite: siteByID("clover-hs"),
		CurrentSite: siteByID("clover-hs"),
		VisibleSites: sitesByID(
			"clover-hs",
			"desert-view",
		),
	},
	"device_wrangler": {
		Persona: devPersona{
			ID:          "device_wrangler",
			Label:       "Device Wrangler",
			DisplayName: "Darius Cole",
			Initials:    "DC",
		},
		LandingPath: "/frequent-fliers",
		Allowed: append(
			[]string{
				"/my-profile",
				"/frequent-fliers",
			},
			devPhoneDirectoryRoutes...,
		),
		Shell: devShellPayload{
			ScopeTitle:        "Assigned site(s)",
			ScopeSubtitle:     "Scoped Access",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "1",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite: siteByID("franklin-ms"),
		CurrentSite: siteByID("franklin-ms"),
		VisibleSites: sitesByID(
			"franklin-ms",
			"highland-es",
		),
	},
	"faculty_staff": {
		Persona: devPersona{
			ID:          "faculty_staff",
			Label:       "Faculty and Staff",
			DisplayName: "Avery Shah",
			Initials:    "AS",
		},
		LandingPath: "/phone-directory/by-person",
		Allowed: append(
			[]string{
				"/my-profile",
			},
			devPhoneDirectoryRoutes...,
		),
		Shell: devShellPayload{
			ScopeTitle:        "Home site",
			ScopeSubtitle:     "Scoped Access",
			SearchPlaceholder: "Search by name, email, phone, extension, or ID...",
			NotificationCount: "0",
			PlatformStatus:    "All Systems Operational",
		},
		DefaultSite:  siteByID("clover-hs"),
		CurrentSite:  siteByID("clover-hs"),
		VisibleSites: sitesByID("clover-hs"),
	},
}

var devPhoneDirectoryEntries = []devPhoneDirectoryEntry{
	// Derived from the read-only directory reference HTML. Extension values and type
	// families are preserved from the source exports; names, emails, phone numbers,
	// and identifiers are deterministic synthetic DEV-only placeholders.
	personDirectoryEntry(
		"person-clover-morgan-slate",
		siteByID("clover-hs"),
		"Morgan Slate",
		"Mathematics Teacher",
		"Mathematics",
		"morgan.slate",
		"360017",
		"EMP-MOCK-1001",
	),
	personDirectoryEntry(
		"person-clover-riley-vale",
		siteByID("clover-hs"),
		"Riley Vale",
		"School Counselor",
		"Student Services",
		"riley.vale",
		"34017",
		"EMP-MOCK-1002",
	),
	personDirectoryEntry(
		"person-desert-taylor-quinn",
		siteByID("desert-view"),
		"Taylor Quinn",
		"Library Media Specialist",
		"Library",
		"taylor.quinn",
		"610053",
		"EMP-MOCK-2001",
	),
	personDirectoryEntry(
		"person-district-jules-rowan",
		siteByID("district-office"),
		"Jules Rowan",
		"HR Specialist",
		"Human Resources",
		"jules.rowan",
		"110013",
		"EMP-MOCK-3001",
	),
	personDirectoryEntry(
		"person-franklin-sage-reed",
		siteByID("franklin-ms"),
		"Sage Reed",
		"Student Support Coach",
		"Student Services",
		"sage.reed",
		"410009",
		"EMP-MOCK-4001",
	),
	commonAreaDirectoryEntry(
		"common-clover-fusion-dialcast",
		siteByID("clover-hs"),
		"Fusion DialCast",
		"Campus Broadcast",
		"40099",
		"CA-MOCK-40099",
	),
	commonAreaDirectoryEntry(
		"common-clover-fusion-intercom",
		siteByID("clover-hs"),
		"Fusion Intercom",
		"Campus Broadcast",
		"40098",
		"CA-MOCK-40098",
	),
	commonAreaDirectoryEntry(
		"common-desert-front-desk",
		siteByID("desert-view"),
		"Front Desk Common Area",
		"Front Office",
		"70099",
		"CA-MOCK-70099",
	),
	commonAreaDirectoryEntry(
		"common-district-food-service",
		siteByID("district-office"),
		"Food Service Common Area",
		"Nutrition Services",
		"22171",
		"CA-MOCK-22171",
	),
	classroomSLGDirectoryEntry(
		"classroom-clover-rm01",
		siteByID("clover-hs"),
		"CLA-RM01",
		"Room 01",
		"330155",
		"SLG-MOCK-330155",
	),
	classroomSLGDirectoryEntry(
		"classroom-clover-rm04",
		siteByID("clover-hs"),
		"CLA-RM04",
		"Room 04",
		"330171",
		"SLG-MOCK-330171",
	),
	classroomSLGDirectoryEntry(
		"classroom-desert-rm01",
		siteByID("desert-view"),
		"MWE-RM01",
		"Room 01",
		"630025",
		"SLG-MOCK-630025",
	),
	classroomSLGDirectoryEntry(
		"classroom-franklin-a101",
		siteByID("franklin-ms"),
		"WMS-A101",
		"Room A101",
		"430002",
		"SLG-MOCK-430002",
	),
	departmentSLGDirectoryEntry(
		"department-clover-main-office",
		siteByID("clover-hs"),
		"CLA - Main Office",
		"Main Office",
		"Administration",
		"350003",
		"LINE-MOCK-350003",
	),
	departmentSLGDirectoryEntry(
		"department-clover-counseling",
		siteByID("clover-hs"),
		"CLA - Counseling",
		"Department",
		"Student Services",
		"350021",
		"LINE-MOCK-350021",
	),
	departmentSLGDirectoryEntry(
		"department-desert-athletics",
		siteByID("desert-view"),
		"MWE - Athletics",
		"Department",
		"Athletics",
		"650010",
		"LINE-MOCK-650010",
	),
	departmentSLGDirectoryEntry(
		"department-district-business",
		siteByID("district-office"),
		"DO - Business Department",
		"Department",
		"Business Services",
		"150009",
		"LINE-MOCK-150009",
	),
	callQueueDirectoryEntry(
		"queue-clover-2fa",
		siteByID("clover-hs"),
		"CLA - 2FA",
		"Security",
		"350022",
		"QUEUE-MOCK-350022",
	),
	callQueueDirectoryEntry(
		"queue-clover-attendance",
		siteByID("clover-hs"),
		"CLA - Attendance",
		"Attendance",
		"350004",
		"QUEUE-MOCK-350004",
	),
	callQueueDirectoryEntry(
		"queue-district-2fa",
		siteByID("district-office"),
		"DO - 2FA",
		"Security",
		"150022",
		"QUEUE-MOCK-150022",
	),
	callQueueDirectoryEntry(
		"queue-desert-attendance",
		siteByID("desert-view"),
		"MWE - Attendance",
		"Attendance",
		"650004",
		"QUEUE-MOCK-650004",
	),
	autoAttendantDirectoryEntry(
		"auto-desert-main",
		siteByID("desert-view"),
		"Mattie Washburn Main Auto Receptionist",
		"650000",
		"AUTO-MOCK-650000",
	),
	autoAttendantDirectoryEntry(
		"auto-district-main",
		siteByID("district-office"),
		"District Office Main Auto Receptionist",
		"150000",
		"AUTO-MOCK-150000",
	),
}

var devPersonaOrder = []string{
	"it_admin",
	"human_resources",
	"site_admin",
	"site_secretary",
	"device_wrangler",
	"faculty_staff",
}

func handleDevSession(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusOK, devSessionPayload{
			Environment:   "development",
			Authenticated: false,
			Authorized:    false,
			Personas:      orderedDevPersonas(),
		})
		return
	}

	writeJSON(w, http.StatusOK, buildDevSessionPayload(config))
}

func handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var request devLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_request",
			"message": "Request body must include persona_id.",
		})
		return
	}

	config, ok := devPersonaConfigs[strings.TrimSpace(request.PersonaID)]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_persona",
			"message": "Unknown DEV persona.",
		})
		return
	}

	writeDevSessionCookie(w, config.Persona.ID)
	writeJSON(w, http.StatusOK, buildDevSessionPayload(config))
}

func handleDevLogout(w http.ResponseWriter, r *http.Request) {
	if !devModeEnabled() || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	clearDevSessionCookie(w)
	writeJSON(w, http.StatusOK, devSessionPayload{
		Environment:   "development",
		Authenticated: false,
		Authorized:    false,
		Personas:      orderedDevPersonas(),
	})
}

func handleDevDataQualityPage(w http.ResponseWriter, r *http.Request) {
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
	if !routeAllowed(config, "/data-quality") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Data Quality is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	writeJSON(w, http.StatusOK, dataQualityPagePayload{
		PageID:      "data-quality",
		Persona:     config.Persona,
		GeneratedAt: "2026-04-30T12:00:00Z",
		Shell:       config.Shell,
		Page: dataQualityContentPayload{
			Title:         "Data Quality",
			Description:   "Source-system conflict and missing-data queues routed to the teams that can fix upstream records.",
			LastRefreshed: "Last refreshed:\nApr 30, 2026\n12:00 PM PT",
			RefreshLabel:  "Refresh",
			SummaryCards: []summaryCardPayload{
				{Title: "Title Mapping", Count: "18"},
				{Title: "Room Mapping", Count: "23"},
				{Title: "Source Conflicts", Count: "41"},
				{Title: "Resolved Today", Count: "29"},
			},
			RoutingCard: routingCardPayload{
				Title:    "Routing",
				Headline: "HR, Site, and IT queues",
				Body:     "Issues are owned by the team that can correct the upstream source. This dashboard surfaces blockers rather than silently patching data.",
			},
			Queue: dataQualityQueuePayload{
				Rows: []dataQualityQueueRow{
					{Issue: "Unmapped job title", Source: "Escape / SFTP", Owner: "HR + IT", Impact: "Blocks access bundle", NextAction: "Map title"},
					{Issue: "Room mismatch", Source: "Aeries", Owner: "Site Secretary", Impact: "Blocks sync", NextAction: "Confirm room"},
					{Issue: "Google-active / Aeries-inactive", Source: "Google + Aeries", Owner: "IT", Impact: "Security review", NextAction: "Schedule deprovision"},
					{Issue: "Missing mandatory field", Source: "HR intake", Owner: "HR", Impact: "Blocks onboarding", NextAction: "Update record"},
					{Issue: "Site mismatch", Source: "Escape / Aeries", Owner: "HR", Impact: "Blocks baseline site selection", NextAction: "Apply temporary override"},
				},
			},
			RoutingRules: routingRulesPayload{
				Title: "Issue Routing Rules",
				Rules: []routingRulePayload{
					{Queue: "HR queues", Description: "Sensitive lifecycle or title issues"},
					{Queue: "Site queues", Description: "Room and student data corrections"},
					{Queue: "IT queues", Description: "Provider conflicts and security mismatches"},
				},
				PrimaryActionLabel: "Open Mapping Dashboard",
			},
		},
		Hotspots: map[string]hotspotPayload{
			"refresh": {
				NodeID: "f104",
				Label:  "Refresh Data Quality",
			},
			"open_mapping_dashboard": {
				NodeID: "f183",
				Label:  "Open Mapping Dashboard",
			},
		},
	})
}

func handleDevPhoneDirectoryByPersonPage(w http.ResponseWriter, r *http.Request) {
	writeDevPhoneDirectoryPage(w, r, "person")
}

func handleDevPhoneDirectoryByRoomPage(w http.ResponseWriter, r *http.Request) {
	writeDevPhoneDirectoryPage(w, r, "room")
}

func handleDevPhoneDirectoryByDepartmentPage(w http.ResponseWriter, r *http.Request) {
	writeDevPhoneDirectoryPage(w, r, "department")
}

func writeDevPhoneDirectoryPage(w http.ResponseWriter, r *http.Request, mode string) {
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
	routePath := "/phone-directory/by-" + mode
	if !routeAllowed(config, routePath) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Phone Directory is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	results := searchPhoneDirectory(config, query, mode)
	var selectedResult *phoneDirectorySearchResult
	if len(results) > 0 {
		first := results[0]
		selectedResult = &first
	}

	writeJSON(w, http.StatusOK, phoneDirectoryPagePayload{
		PageID:      "phone-directory-by-" + mode,
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: "2026-05-03T12:00:00Z",
		Page: phoneDirectoryContentPayload{
			Mode:            mode,
			Title:           "Phone Directory",
			Description:     phoneDirectoryDescription(mode),
			LastRefreshed:   "Last refreshed:\nMay 3, 2026\n9:00 AM PT",
			Query:           query,
			CurrentSiteID:   config.CurrentSite.ID,
			CurrentSiteName: config.CurrentSite.Name,
			Results:         results,
			SelectedResult:  selectedResult,
		},
	})
}

func orderedDevPersonas() []devPersona {
	personas := make([]devPersona, 0, len(devPersonaOrder))
	for _, id := range devPersonaOrder {
		if config, ok := devPersonaConfigs[id]; ok {
			personas = append(personas, config.Persona)
		}
	}
	return personas
}

func concatRoutes(groups ...[]string) []string {
	total := 0
	for _, group := range groups {
		total += len(group)
	}

	routes := make([]string, 0, total)
	for _, group := range groups {
		routes = append(routes, group...)
	}
	return routes
}

func buildDevSessionPayload(config devPersonaConfig) devSessionPayload {
	persona := config.Persona
	return devSessionPayload{
		Environment:     "development",
		Authenticated:   true,
		Authorized:      true,
		CurrentPersona:  &persona,
		Personas:        orderedDevPersonas(),
		LandingPath:     config.LandingPath,
		AllowedRoutes:   slices.Clone(config.Allowed),
		Shell:           config.Shell,
		DefaultSiteID:   config.DefaultSite.ID,
		DefaultSiteName: config.DefaultSite.Name,
		CurrentSiteID:   config.CurrentSite.ID,
		CurrentSiteName: config.CurrentSite.Name,
	}
}

func resolveAuthenticatedDevPersona(r *http.Request) (devPersonaConfig, bool) {
	cookie, err := r.Cookie(devSessionCookieName)
	if err != nil {
		return devPersonaConfig{}, false
	}

	config, ok := devPersonaConfigs[strings.TrimSpace(cookie.Value)]
	if !ok {
		return devPersonaConfig{}, false
	}
	return config, true
}

func routeAllowed(config devPersonaConfig, path string) bool {
	return slices.Contains(config.Allowed, path)
}

func writeDevSessionCookie(w http.ResponseWriter, personaID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     devSessionCookieName,
		Value:    personaID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(12 * time.Hour),
	})
}

func clearDevSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     devSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func devModeEnabled() bool {
	mode := strings.TrimSpace(os.Getenv("APP_ENV"))
	if mode == "" {
		mode = "development"
	}
	return strings.EqualFold(mode, "development")
}

func siteByID(id string) devSiteContext {
	return devSiteCatalog[id]
}

func sitesByID(ids ...string) []devSiteContext {
	sites := make([]devSiteContext, 0, len(ids))
	for _, id := range ids {
		sites = append(sites, siteByID(id))
	}
	return sites
}

func personDirectoryEntry(id string, site devSiteContext, name string, role string, department string, emailLocalPart string, extension string, identifier string) devPhoneDirectoryEntry {
	email := syntheticEmail(emailLocalPart)
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypePerson,
		TypeLabel:  "Person",
		Title:      name,
		Subtitle:   role + " • " + department,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Role:       role,
		Department: department,
		Email:      email,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(name, role, department, email, phone, extension, identifier, site.Name),
	})
}

func commonAreaDirectoryEntry(id string, site devSiteContext, title string, location string, extension string, identifier string) devPhoneDirectoryEntry {
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypeCommonArea,
		TypeLabel:  "Common Area",
		Title:      title,
		Subtitle:   "Common area phone • " + location,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Location:   location,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(title, location, phone, extension, identifier, site.Name, "common area"),
	})
}

func classroomSLGDirectoryEntry(id string, site devSiteContext, title string, location string, extension string, identifier string) devPhoneDirectoryEntry {
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypeClassroomSLG,
		TypeLabel:  "Classroom Shared Line",
		Title:      title,
		Subtitle:   "Classroom shared line • " + location,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Location:   location,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(title, location, phone, extension, identifier, site.Name, "classroom shared line"),
	})
}

func departmentSLGDirectoryEntry(id string, site devSiteContext, title string, classification string, department string, extension string, identifier string) devPhoneDirectoryEntry {
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypeDepartmentSLG,
		TypeLabel:  classification,
		Title:      title,
		Subtitle:   classification + " • " + department,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Department: department,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(title, classification, department, phone, extension, identifier, site.Name),
	})
}

func callQueueDirectoryEntry(id string, site devSiteContext, title string, department string, extension string, identifier string) devPhoneDirectoryEntry {
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypeCallQueue,
		TypeLabel:  "Call Queue",
		Title:      title,
		Subtitle:   "Call queue • " + department,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Department: department,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(title, department, phone, extension, identifier, site.Name, "call queue"),
	})
}

func autoAttendantDirectoryEntry(id string, site devSiteContext, title string, extension string, identifier string) devPhoneDirectoryEntry {
	phone := syntheticPhoneNumber(extension)
	return newPhoneDirectoryEntry(devPhoneDirectoryEntry{
		ID:         id,
		Type:       phoneDirectoryTypeAutoAttendant,
		TypeLabel:  "Auto Attendant",
		Title:      title,
		Subtitle:   "Auto attendant • " + site.Name,
		SiteID:     site.ID,
		SiteName:   site.Name,
		Phone:      phone,
		Extension:  extension,
		Identifier: identifier,
		Searchable: buildSearchableValues(title, phone, extension, identifier, site.Name, "auto attendant"),
	})
}

func newPhoneDirectoryEntry(entry devPhoneDirectoryEntry) devPhoneDirectoryEntry {
	length, valid := extensionMetadata(entry.Extension)
	entry.ExtensionLength = length
	entry.ExtensionValid = valid
	return entry
}

func buildSearchableValues(values ...string) []string {
	searchable := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		searchable = append(searchable, trimmed)
	}
	return searchable
}

func extensionMetadata(extension string) (int, bool) {
	digits := extensionDigits(extension)
	length := len(digits)
	return length, length >= 4 && length <= 6
}

func extensionDigits(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func syntheticPhoneNumber(extension string) string {
	suffix := extensionDigits(extension)
	if suffix == "" {
		return ""
	}
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	if len(suffix) < 4 {
		suffix = strings.Repeat("0", 4-len(suffix)) + suffix
	}
	return "(707) 555-" + suffix
}

func syntheticEmail(localPart string) string {
	return localPart + "@mock.wusd.invalid"
}

func phoneDirectoryDescription(mode string) string {
	switch mode {
	case "room":
		return "Search common area phones and classroom shared lines. Results from your current site appear first."
	case "department":
		return "Search department shared lines and call queues. Results from your current site appear first."
	default:
		return "Search people and common area phones. Results from your current site appear first."
	}
}

func searchPhoneDirectory(config devPersonaConfig, query string, mode string) []phoneDirectorySearchResult {
	visibleSiteOrder := map[string]int{}
	for index, site := range config.VisibleSites {
		visibleSiteOrder[site.ID] = index
	}

	normalizedQuery := normalizeSearchValue(query)
	ranked := make([]rankedPhoneDirectoryResult, 0, len(devPhoneDirectoryEntries))
	for _, entry := range devPhoneDirectoryEntries {
		siteOrder, visible := visibleSiteOrder[entry.SiteID]
		if !visible {
			continue
		}
		if !phoneDirectoryModeAllows(mode, entry.Type) {
			continue
		}

		match := bestPhoneDirectoryMatch(entry, normalizedQuery)
		if normalizedQuery != "" && match == nil {
			continue
		}

		matchRank := 3
		if match != nil {
			matchRank = match.Rank
		}

		siteRank := 1
		if entry.SiteID == config.CurrentSite.ID {
			siteRank = 0
		}

		ranked = append(ranked, rankedPhoneDirectoryResult{
			Result: phoneDirectorySearchResult{
				ID:              entry.ID,
				Type:            entry.Type,
				TypeLabel:       entry.TypeLabel,
				Title:           entry.Title,
				Subtitle:        entry.Subtitle,
				SiteID:          entry.SiteID,
				SiteName:        entry.SiteName,
				Role:            entry.Role,
				Department:      entry.Department,
				Location:        entry.Location,
				Email:           entry.Email,
				Phone:           entry.Phone,
				Extension:       entry.Extension,
				ExtensionLength: entry.ExtensionLength,
				ExtensionValid:  entry.ExtensionValid,
				Identifier:      entry.Identifier,
			},
			SiteRank:      siteRank,
			SiteOrder:     siteOrder,
			TypeRank:      phoneDirectoryTypeRank(mode, entry.Type),
			MatchRank:     matchRank,
			NormalizedKey: normalizeSearchValue(entry.Title),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]

		if left.SiteRank != right.SiteRank {
			return left.SiteRank < right.SiteRank
		}
		if left.TypeRank != right.TypeRank {
			return left.TypeRank < right.TypeRank
		}
		if left.MatchRank != right.MatchRank {
			return left.MatchRank < right.MatchRank
		}
		if left.SiteOrder != right.SiteOrder {
			return left.SiteOrder < right.SiteOrder
		}
		if left.NormalizedKey != right.NormalizedKey {
			return left.NormalizedKey < right.NormalizedKey
		}
		return left.Result.ID < right.Result.ID
	})

	results := make([]phoneDirectorySearchResult, 0, len(ranked))
	for _, entry := range ranked {
		results = append(results, entry.Result)
	}
	return results
}

func phoneDirectoryModeAllows(mode string, entryType string) bool {
	switch mode {
	case "room":
		return entryType == phoneDirectoryTypeCommonArea || entryType == phoneDirectoryTypeClassroomSLG
	case "department":
		return entryType == phoneDirectoryTypeDepartmentSLG || entryType == phoneDirectoryTypeCallQueue
	default:
		return entryType == phoneDirectoryTypePerson || entryType == phoneDirectoryTypeCommonArea
	}
}

func bestPhoneDirectoryMatch(entry devPhoneDirectoryEntry, normalizedQuery string) *phoneDirectorySearchMatch {
	if normalizedQuery == "" {
		return &phoneDirectorySearchMatch{Rank: 3}
	}

	bestRank := 99
	for _, candidate := range entry.Searchable {
		normalizedCandidate := normalizeSearchValue(candidate)
		if normalizedCandidate == "" {
			continue
		}

		rank := 99
		switch {
		case normalizedCandidate == normalizedQuery:
			rank = 0
		case strings.HasPrefix(normalizedCandidate, normalizedQuery):
			rank = 1
		case strings.Contains(normalizedCandidate, normalizedQuery):
			rank = 2
		}

		if rank < bestRank {
			bestRank = rank
		}
	}

	if bestRank == 99 {
		return nil
	}
	return &phoneDirectorySearchMatch{Rank: bestRank}
}

func phoneDirectoryTypeRank(mode string, resultType string) int {
	switch mode {
	case "room":
		switch resultType {
		case phoneDirectoryTypeCommonArea:
			return 0
		case phoneDirectoryTypeClassroomSLG:
			return 1
		case phoneDirectoryTypePerson:
			return 2
		case phoneDirectoryTypeDepartmentSLG, phoneDirectoryTypeCallQueue:
			return 3
		default:
			return 4
		}
	case "department":
		switch resultType {
		case phoneDirectoryTypeDepartmentSLG:
			return 0
		case phoneDirectoryTypeCallQueue:
			return 1
		case phoneDirectoryTypeCommonArea, phoneDirectoryTypeClassroomSLG:
			return 2
		case phoneDirectoryTypePerson:
			return 3
		default:
			return 4
		}
	default:
		switch resultType {
		case phoneDirectoryTypePerson:
			return 0
		case phoneDirectoryTypeCommonArea:
			return 1
		case phoneDirectoryTypeClassroomSLG, phoneDirectoryTypeDepartmentSLG, phoneDirectoryTypeCallQueue:
			return 2
		default:
			return 3
		}
	}
}

func normalizeSearchValue(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == ' ':
			builder.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}
