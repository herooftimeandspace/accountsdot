package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const devSessionCookieName = "wizard_dev_session"
const devFeatureFlagUpdateMaxBodyBytes int64 = 16 * 1024

const (
	phoneDirectoryTypePerson        = "person"
	phoneDirectoryTypeCommonArea    = "common_area"
	phoneDirectoryTypeClassroomSLG  = "classroom_slg"
	phoneDirectoryTypeDepartmentSLG = "department_slg"
	phoneDirectoryTypeCallQueue     = "call_queue"
	phoneDirectoryTypeAutoAttendant = "auto_attendant"
)

var (
	devGlobalSearchRoute     = "/search"
	devDepartingSeniorsRoute = "/departing-seniors"
	devPhoneDirectoryRoutes  = []string{
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
		"/room-moves/bulk-draft",
	}
	devITOnlyRoutes = []string{
		"/dashboard/it-admin",
		"/data-quality",
		"/reports", "/reports/security-issues",
		"/reports/sync-transparency",
		"/admin",
		"/admin/feature-flags",
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
	Environment         string                   `json:"environment"`
	Authenticated       bool                     `json:"authenticated"`
	Authorized          bool                     `json:"authorized"`
	AuthenticationMode  string                   `json:"authentication_mode,omitempty"`
	BreakglassAccountID string                   `json:"breakglass_account_id,omitempty"`
	CurrentPersona      *devPersona              `json:"current_persona,omitempty"`
	Personas            []devPersona             `json:"personas"`
	LandingPath         string                   `json:"landing_path,omitempty"`
	AllowedRoutes       []string                 `json:"allowed_routes,omitempty"`
	FeatureFlags        []devFeatureAvailability `json:"feature_flags,omitempty"`
	Shell               devShellPayload          `json:"shell,omitempty"`
	DefaultSiteID       string                   `json:"default_site_id,omitempty"`
	DefaultSiteName     string                   `json:"default_site_name,omitempty"`
	CurrentSiteID       string                   `json:"current_site_id,omitempty"`
	CurrentSiteName     string                   `json:"current_site_name,omitempty"`
	VisibleSites        []devSiteContext         `json:"visible_sites,omitempty"`
}

type devLoginRequest struct {
	PersonaID string `json:"persona_id"`
}

type devFeatureAvailability struct {
	Key        string                    `json:"key"`
	Label      string                    `json:"label"`
	Enabled    bool                      `json:"enabled"`
	Indicators []devFeatureFlagIndicator `json:"indicators"`
}

type devFeatureFlagIndicator struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	TargetLabel string `json:"target_label"`
	Enabled     bool   `json:"enabled"`
	ReadOnly    bool   `json:"read_only"`
}

type devFeatureFlagsPayload struct {
	PageID      string                  `json:"page_id"`
	Persona     devPersona              `json:"persona"`
	Shell       devShellPayload         `json:"shell"`
	GeneratedAt string                  `json:"generated_at"`
	Flags       []devFeatureFlagPayload `json:"flags"`
	Personas    []devFeatureTarget      `json:"personas"`
	Sites       []devFeatureTarget      `json:"sites"`
}

type devFeatureFlagPayload struct {
	Key              string                    `json:"key"`
	Label            string                    `json:"label"`
	Description      string                    `json:"description"`
	FeatureRoute     string                    `json:"feature_route"`
	Routes           []string                  `json:"routes"`
	DefaultEnabled   bool                      `json:"default_enabled"`
	EffectiveForIT   bool                      `json:"effective_for_it_admin"`
	PersonaTargets   []devFeatureTargetState   `json:"persona_targets"`
	SiteTargets      []devFeatureTargetState   `json:"site_targets"`
	ActiveIndicators []devFeatureFlagIndicator `json:"active_indicators"`
}

type devFeatureTarget struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type devFeatureTargetState struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	ReadOnly bool   `json:"read_only"`
}

type devFeatureFlagUpdateRequest struct {
	Targets []devFeatureFlagTargetUpdate `json:"targets"`
}

type devFeatureFlagTargetUpdate struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Enabled    bool   `json:"enabled"`
}

type devFeatureFlagAuditDelta struct {
	FlagKey       string    `json:"flag_key"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	BeforeEnabled bool      `json:"before_enabled"`
	AfterEnabled  bool      `json:"after_enabled"`
	ActorID       string    `json:"actor_id"`
	ChangedAt     time.Time `json:"changed_at"`
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
	Mode                  string                       `json:"mode"`
	Title                 string                       `json:"title"`
	Description           string                       `json:"description"`
	LastRefreshed         string                       `json:"last_refreshed"`
	Query                 string                       `json:"query"`
	CurrentSiteID         string                       `json:"current_site_id"`
	CurrentSiteName       string                       `json:"current_site_name"`
	DirectoryScopeID      string                       `json:"directory_scope_id"`
	DirectoryScopeLabel   string                       `json:"directory_scope_label"`
	DirectoryScopeOptions []directoryScopeOption       `json:"directory_scope_options"`
	Results               []phoneDirectorySearchResult `json:"results"`
	SelectedResult        *phoneDirectorySearchResult  `json:"selected_result,omitempty"`
}

type directoryScopeOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type dataQualityContentPayload struct {
	Title         string                  `json:"title"`
	LastRefreshed string                  `json:"last_refreshed"`
	RefreshLabel  string                  `json:"refresh_label"`
	SummaryCards  []summaryCardPayload    `json:"summary_cards"`
	Queue         dataQualityQueuePayload `json:"queue"`
}

type summaryCardPayload struct {
	Title string `json:"title"`
	Count string `json:"count"`
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

var devSiteOrder = []string{
	"district-office",
	"business-office",
	"clover-hs",
	"desert-view",
	"franklin-ms",
	"highland-es",
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
				devGlobalSearchRoute,
				devDepartingSeniorsRoute,
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
				devGlobalSearchRoute,
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
				devGlobalSearchRoute,
				"/student-data-cleanup",
				"/frequent-fliers",
				"/onboarding",
				"/offboarding",
				"/room-moves",
				"/room-moves/bulk-draft",
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
		DefaultSite:  siteByID("clover-hs"),
		CurrentSite:  siteByID("clover-hs"),
		VisibleSites: sitesByID("clover-hs"),
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
				devGlobalSearchRoute,
				"/onboarding",
				"/student-data-cleanup",
				"/room-moves",
				"/room-moves/bulk-draft",
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
				devGlobalSearchRoute,
				"/frequent-fliers",
				devDepartingSeniorsRoute,
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
				devGlobalSearchRoute,
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

type devFeatureFlagDefinition struct {
	Key            string
	Label          string
	Description    string
	FeatureRoute   string
	Routes         []string
	DefaultEnabled bool
}

type devFeatureFlagTargetKey struct {
	TargetType string
	TargetID   string
}

var devFeatureFlagRegistry = []devFeatureFlagDefinition{
	{
		Key:            "dashboard.site_admin",
		Label:          "Site Admin Dashboard",
		Description:    "Controls the site-scoped administrative dashboard route for non-IT users.",
		FeatureRoute:   "/dashboard/site-admin",
		Routes:         []string{"/dashboard/site-admin"},
		DefaultEnabled: true,
	},
	{
		Key:            "onboarding",
		Label:          "Onboarding",
		Description:    "Controls staff onboarding visibility and DEV onboarding API access.",
		FeatureRoute:   "/onboarding",
		Routes:         []string{"/onboarding"},
		DefaultEnabled: true,
	},
	{
		Key:            "offboarding",
		Label:          "Offboarding",
		Description:    "Controls offboarding visibility and DEV offboarding API access.",
		FeatureRoute:   "/offboarding",
		Routes:         []string{"/offboarding"},
		DefaultEnabled: true,
	},
	{
		Key:            "departing_seniors",
		Label:          "Departing Seniors",
		Description:    "Controls departing-senior account lifecycle visibility and DEV API access.",
		FeatureRoute:   "/departing-seniors",
		Routes:         []string{"/departing-seniors"},
		DefaultEnabled: true,
	},
	{
		Key:            "room_moves",
		Label:          "Room Moves",
		Description:    "Controls room-move review, draft, and reversal DEV routes for non-IT users.",
		FeatureRoute:   "/room-moves",
		Routes:         []string{"/room-moves", "/room-moves/bulk-draft"},
		DefaultEnabled: true,
	},
	{
		Key:            "phone_directory",
		Label:          "Phone Directory",
		Description:    "Controls phone directory routes and DEV directory API access.",
		FeatureRoute:   "/phone-directory/by-person",
		Routes:         slices.Clone(devPhoneDirectoryRoutes),
		DefaultEnabled: true,
	},
	{
		Key:            "student_data_cleanup",
		Label:          "Student Data Cleanup",
		Description:    "Controls the student source-data cleanup queue for scoped site users.",
		FeatureRoute:   "/student-data-cleanup",
		Routes:         []string{"/student-data-cleanup"},
		DefaultEnabled: true,
	},
	{
		Key:            "frequent_fliers",
		Label:          "Frequent Fliers",
		Description:    "Controls repeated device-support pattern visibility for site-scoped users.",
		FeatureRoute:   "/frequent-fliers",
		Routes:         []string{"/frequent-fliers"},
		DefaultEnabled: true,
	},
}

var (
	devFeatureFlagStateMu            sync.Mutex
	devFeatureFlagState              = initialDevFeatureFlagState()
	devFeatureFlagStateLoaded        bool
	devFeatureFlagStateLoadAttempted bool
	devFeatureFlagStoreMu            sync.Mutex
	devFeatureFlagStore              devFeatureFlagStorage
	devFeatureFlagStoreError         error
)

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
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	identity, ok := resolveAuthenticatedDevSession(r)
	if !ok {
		writeJSON(w, http.StatusOK, devSessionPayload{
			Environment:   currentAppEnvironment(),
			Authenticated: false,
			Authorized:    false,
			Personas:      orderedDevPersonas(),
		})
		return
	}

	config := identity.Config
	if identity.Breakglass {
		writeJSON(w, http.StatusOK, buildBreakglassSessionPayload(r.Context(), config, identity.BreakglassAccountID))
		return
	}
	writeJSON(w, http.StatusOK, buildDevSessionPayload(r.Context(), config))
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

	writeDevSessionCookie(w, r, config.Persona.ID)
	writeJSON(w, http.StatusOK, buildDevSessionPayload(r.Context(), config))
}

func handleDevLogout(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	identity, ok := resolveAuthenticatedDevSession(r)
	if ok && identity.Breakglass {
		if err := recordBreakglassAudit(r.Context(), breakglassAuditEvent{
			AccountID:     identity.BreakglassAccountID,
			Action:        "sign_out",
			Outcome:       "allowed",
			SourceIP:      sourceIPForBreakglass(r).String(),
			PersonaID:     identity.Config.Persona.ID,
			RecordedAt:    time.Now().UTC(),
			RequestID:     strings.TrimSpace(r.Header.Get("X-Request-ID")),
			TargetSession: "cookie:" + devSessionCookieName,
		}); err != nil {
			writeBreakglassAuditUnavailable(w, err)
			return
		}
	}
	clearDevSessionCookie(w, r)
	writeJSON(w, http.StatusOK, devSessionPayload{
		Environment:   currentAppEnvironment(),
		Authenticated: false,
		Authorized:    false,
		Personas:      orderedDevPersonas(),
	})
}

// handleDevFeatureFlags returns the IT Admin-only feature flag management payload.
func handleDevFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can manage feature flags.",
		})
		return
	}
	if config.Persona.ID != "it_admin" {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Feature flags are available to IT Admin only.",
			"persona": config.Persona,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, buildDevFeatureFlagsPayload(r.Context(), config))
	default:
		http.NotFound(w, r)
	}
}

// handleDevFeatureFlag applies IT Admin feature flag target updates for one flag key.
func handleDevFeatureFlag(w http.ResponseWriter, r *http.Request) {
	if !devSessionConsumerEnabled(r) || r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}

	config, ok := resolveAuthenticatedDevPersona(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "not_authorized",
			"message": "You need to sign in before you can manage feature flags.",
		})
		return
	}
	if config.Persona.ID != "it_admin" {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Feature flags are available to IT Admin only.",
			"persona": config.Persona,
		})
		return
	}

	flagKey := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/dev/feature-flags/"), "/")
	definition, ok := devFeatureFlagDefinitionByKey(flagKey)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "not_found",
			"message": "Unknown feature flag.",
		})
		return
	}

	request, err := decodeDevFeatureFlagUpdateRequest(w, r)
	if err != nil {
		status := http.StatusBadRequest
		message := "Request body must include feature flag targets."
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			status = http.StatusRequestEntityTooLarge
			message = "Feature flag update payload is too large."
		}
		writeJSON(w, status, map[string]any{
			"code":    "invalid_request",
			"message": message,
		})
		return
	}
	if len(request.Targets) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "invalid_request",
			"message": "At least one feature flag target is required.",
		})
		return
	}
	seenTargets := make(map[devFeatureFlagTargetKey]bool, len(request.Targets))
	for _, target := range request.Targets {
		if target.TargetType == "persona" && target.TargetID == "it_admin" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_target",
				"message": "IT Admin is always enabled and cannot be stored as an editable target.",
			})
			return
		}
		if !validDevFeatureFlagTarget(target) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_target",
				"message": "Feature flag targets must reference a known non-IT persona or site.",
			})
			return
		}
		targetKey := devFeatureFlagTargetKey{TargetType: target.TargetType, TargetID: target.TargetID}
		if seenTargets[targetKey] {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "invalid_target",
				"message": "Feature flag target updates must not contain duplicate target_type and target_id entries.",
			})
			return
		}
		seenTargets[targetKey] = true
	}

	if err := updateDevFeatureFlagTargets(r.Context(), definition.Key, request.Targets, config.Persona.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "feature_flag_store_failed",
			"message": "Feature flag state could not be persisted.",
		})
		return
	}
	writeJSON(w, http.StatusOK, buildDevFeatureFlagPayload(r.Context(), definition))
}

// decodeDevFeatureFlagUpdateRequest reads the feature-flag target update body
// for handleDevFeatureFlagUpdate. It enforces the small DEV API request limit,
// rejects unknown fields and trailing JSON, and returns a typed request so the
// handler can validate targets before mutating storage.
func decodeDevFeatureFlagUpdateRequest(w http.ResponseWriter, r *http.Request) (devFeatureFlagUpdateRequest, error) {
	var request devFeatureFlagUpdateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, devFeatureFlagUpdateMaxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return devFeatureFlagUpdateRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return devFeatureFlagUpdateRequest{}, err
		}
		return devFeatureFlagUpdateRequest{}, errors.New("request body must contain one JSON object")
	}
	return request, nil
}

// handleDevDataQualityPage serves the DEV Data Quality page payload. The route
// requires development mode, a signed-in IT Admin persona, and feature-flag
// route access; successful responses contain the awareness queue consumed by
// the React page. The row actions intentionally point toward owner surfaces
// rather than local correction buttons because this page escalates district-wide
// issues but does not edit HR, site, student, or provider source records.
func handleDevDataQualityPage(w http.ResponseWriter, r *http.Request) {
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
	if !routeAllowed(r.Context(), config, "/data-quality") {
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
			LastRefreshed: "Last refreshed:\nApr 30, 2026\n12:00 PM PT",
			RefreshLabel:  "Refresh",
			SummaryCards: []summaryCardPayload{
				{Title: "Title Mapping", Count: "18"},
				{Title: "Room Mapping", Count: "23"},
				{Title: "Source Conflicts", Count: "41"},
				{Title: "Resolved Today", Count: "29"},
			},
			Queue: dataQualityQueuePayload{
				Rows: []dataQualityQueueRow{
					{Issue: "Unmapped job title", Source: "Escape / SFTP", Owner: "HR + IT", Impact: "Blocks access bundle", NextAction: "Review in HR lifecycle"},
					{Issue: "Room mismatch", Source: "Aeries", Owner: "Site Secretary", Impact: "Blocks sync", NextAction: "Route to site owner"},
					{Issue: "Google-active / Aeries-inactive", Source: "Google + Aeries", Owner: "IT", Impact: "Security review", NextAction: "Review in Admin"},
					{Issue: "Missing mandatory field", Source: "HR intake", Owner: "HR", Impact: "Blocks onboarding", NextAction: "Complete in Onboarding"},
					{Issue: "Site mismatch", Source: "Escape / Aeries", Owner: "HR", Impact: "Blocks baseline site selection", NextAction: "Review in HR lifecycle"},
				},
			},
		},
		Hotspots: map[string]hotspotPayload{
			"refresh": {
				NodeID: "f104",
				Label:  "Refresh Data Quality",
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
	routePath := "/phone-directory/by-" + mode
	if !routeAllowed(r.Context(), config, routePath) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "forbidden",
			"message": "Phone Directory is not available for this role.",
			"persona": config.Persona,
		})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	directoryScopeID, directoryScopeLabel := resolvePhoneDirectoryScope(config, strings.TrimSpace(r.URL.Query().Get("site_id")))
	results := searchPhoneDirectory(query, mode, directoryScopeID)

	writeJSON(w, http.StatusOK, phoneDirectoryPagePayload{
		PageID:      "phone-directory-by-" + mode,
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: "2026-05-03T12:00:00Z",
		Page: phoneDirectoryContentPayload{
			Mode:                  mode,
			Title:                 "Phone Directory",
			Description:           phoneDirectoryDescription(mode),
			LastRefreshed:         "Last refreshed:\nMay 3, 2026\n9:00 AM PT",
			Query:                 query,
			CurrentSiteID:         config.CurrentSite.ID,
			CurrentSiteName:       config.CurrentSite.Name,
			DirectoryScopeID:      directoryScopeID,
			DirectoryScopeLabel:   directoryScopeLabel,
			DirectoryScopeOptions: phoneDirectoryScopeOptions(),
			Results:               results,
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

// devFeatureFlagDefinitionByKey returns the configured feature flag with the requested key.
func devFeatureFlagDefinitionByKey(key string) (devFeatureFlagDefinition, bool) {
	for _, definition := range devFeatureFlagRegistry {
		if definition.Key == key {
			return definition, true
		}
	}
	return devFeatureFlagDefinition{}, false
}

// devFeatureFlagDefinitionForRoute returns the feature flag that gates a frontend route.
func devFeatureFlagDefinitionForRoute(path string) (devFeatureFlagDefinition, bool) {
	for _, definition := range devFeatureFlagRegistry {
		if slices.Contains(definition.Routes, path) {
			return definition, true
		}
	}
	return devFeatureFlagDefinition{}, false
}

// devFeatureFlagStorage abstracts feature flag state so DEV can fall back to memory without a database.
type devFeatureFlagStorage interface {
	Snapshot(context.Context) (map[string]map[devFeatureFlagTargetKey]bool, error)
	UpdateTargets(context.Context, string, []devFeatureFlagTargetUpdate, string) error
}

type memoryDevFeatureFlagStore struct{}

// Snapshot returns the current process-local feature-flag matrix for DEV runs
// without DATABASE_URL. It clones state before returning so callers cannot
// mutate shared routing decisions outside the store lock.
func (memoryDevFeatureFlagStore) Snapshot(context.Context) (map[string]map[devFeatureFlagTargetKey]bool, error) {
	devFeatureFlagStateMu.Lock()
	defer devFeatureFlagStateMu.Unlock()
	return cloneDevFeatureFlagState(devFeatureFlagState), nil
}

// UpdateTargets applies feature-flag target changes to the process-local DEV
// store. This mock-only write path is used when no database is configured and
// has no retry or audit persistence beyond the in-memory state mutation.
func (memoryDevFeatureFlagStore) UpdateTargets(_ context.Context, flagKey string, updates []devFeatureFlagTargetUpdate, _ string) error {
	devFeatureFlagStateMu.Lock()
	defer devFeatureFlagStateMu.Unlock()
	if _, ok := devFeatureFlagState[flagKey]; !ok {
		devFeatureFlagState[flagKey] = make(map[devFeatureFlagTargetKey]bool)
	}
	for _, update := range updates {
		devFeatureFlagState[flagKey][devFeatureFlagTargetKey{TargetType: update.TargetType, TargetID: update.TargetID}] = update.Enabled
	}
	return nil
}

type postgresDevFeatureFlagStore struct {
	pool *pgxpool.Pool
}

// Snapshot loads the database-backed feature-flag target matrix for DEV
// routing. It first reconciles the checked-in registry into feature_flags and
// feature_flag_targets inside db.WithRetry, then returns a full target map for
// session and route-authorization checks.
func (store postgresDevFeatureFlagStore) Snapshot(ctx context.Context) (map[string]map[devFeatureFlagTargetKey]bool, error) {
	state := initialDevFeatureFlagState()
	err := db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureDevFeatureFlagRegistry(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			select flag_key, target_type, target_id, enabled
			from feature_flag_targets
			order by flag_key, target_type, target_id
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var flagKey string
			var target devFeatureFlagTargetKey
			var enabled bool
			if err := rows.Scan(&flagKey, &target.TargetType, &target.TargetID, &enabled); err != nil {
				return err
			}
			if _, ok := state[flagKey]; !ok {
				state[flagKey] = make(map[devFeatureFlagTargetKey]bool)
			}
			state[flagKey][target] = enabled
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

// UpdateTargets persists DEV feature-flag target changes and writes matching
// audit_log deltas in one retried database transaction. The caller supplies the
// authenticated DEV persona id as actorID; unchanged targets are skipped so
// repeat requests stay idempotent and do not create duplicate audit entries.
func (store postgresDevFeatureFlagStore) UpdateTargets(ctx context.Context, flagKey string, updates []devFeatureFlagTargetUpdate, actorID string) error {
	return db.WithRetry(ctx, store.pool, func(tx pgx.Tx) error {
		if err := ensureDevFeatureFlagRegistry(ctx, tx); err != nil {
			return err
		}
		changedAt := time.Now().UTC()
		for _, update := range updates {
			var beforeEnabled bool
			if err := tx.QueryRow(ctx, `
				select enabled
				from feature_flag_targets
				where flag_key = $1 and target_type = $2 and target_id = $3
			`, flagKey, update.TargetType, update.TargetID).Scan(&beforeEnabled); err != nil {
				return err
			}
			if beforeEnabled == update.Enabled {
				continue
			}
			if _, err := tx.Exec(ctx, `
				update feature_flag_targets
				set enabled = $4, actor_id = $5, updated_at = $6
				where flag_key = $1 and target_type = $2 and target_id = $3
			`, flagKey, update.TargetType, update.TargetID, update.Enabled, actorID, changedAt); err != nil {
				return err
			}
			delta := devFeatureFlagAuditDelta{
				FlagKey:       flagKey,
				TargetType:    update.TargetType,
				TargetID:      update.TargetID,
				BeforeEnabled: beforeEnabled,
				AfterEnabled:  update.Enabled,
				ActorID:       actorID,
				ChangedAt:     changedAt,
			}
			diff, err := json.Marshal(delta)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				insert into audit_log (actor_id, actor_type, target_entity, target_id, reason, diff, created_at)
				values ($1, 'dev_persona', 'feature_flag_target', $2, 'dev_feature_flag_update', $3::jsonb, $4)
			`, actorID, flagKey+":"+update.TargetType+":"+update.TargetID, string(diff), changedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

// ensureDevFeatureFlagRegistry reconciles the static DEV feature-flag registry
// into database tables before snapshot or update work. It upserts flag metadata
// and creates missing persona/site targets with default values so later writes
// can update known rows only.
func ensureDevFeatureFlagRegistry(ctx context.Context, tx pgx.Tx) error {
	for _, definition := range devFeatureFlagRegistry {
		if _, err := tx.Exec(ctx, `
			insert into feature_flags (flag_key, label, description, feature_route, default_enabled, actor_id, updated_at)
			values ($1, $2, $3, $4, $5, 'registry', now())
			on conflict (flag_key) do update
			set label = excluded.label,
				description = excluded.description,
				feature_route = excluded.feature_route,
				default_enabled = excluded.default_enabled,
				updated_at = now()
			where feature_flags.label is distinct from excluded.label
				or feature_flags.description is distinct from excluded.description
				or feature_flags.feature_route is distinct from excluded.feature_route
				or feature_flags.default_enabled is distinct from excluded.default_enabled
		`, definition.Key, definition.Label, definition.Description, definition.FeatureRoute, definition.DefaultEnabled); err != nil {
			return err
		}
		for _, persona := range orderedDevFeatureFlagPersonas() {
			if err := ensureDevFeatureFlagTarget(ctx, tx, definition, "persona", persona.ID); err != nil {
				return err
			}
		}
		for _, site := range orderedDevFeatureFlagSites() {
			if err := ensureDevFeatureFlagTarget(ctx, tx, definition, "site", site.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureDevFeatureFlagTarget creates a missing database target row for one
// registry flag and persona/site target. Existing target rows are left untouched
// so operator changes survive registry reconciliation.
func ensureDevFeatureFlagTarget(ctx context.Context, tx pgx.Tx, definition devFeatureFlagDefinition, targetType string, targetID string) error {
	_, err := tx.Exec(ctx, `
		insert into feature_flag_targets (flag_key, target_type, target_id, enabled, actor_id)
		values ($1, $2, $3, $4, 'registry')
		on conflict (flag_key, target_type, target_id) do nothing
	`, definition.Key, targetType, targetID, definition.DefaultEnabled)
	return err
}

// currentDevFeatureFlagStore returns the configured DEV feature-flag storage
// backend. It memoizes either an in-memory store or a database-backed store, and
// caches connection setup errors so request handlers can fail closed with a
// stable debugging signal.
func currentDevFeatureFlagStore(ctx context.Context) (devFeatureFlagStorage, error) {
	devFeatureFlagStoreMu.Lock()
	defer devFeatureFlagStoreMu.Unlock()
	if devFeatureFlagStore != nil || devFeatureFlagStoreError != nil {
		return devFeatureFlagStore, devFeatureFlagStoreError
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		devFeatureFlagStore = memoryDevFeatureFlagStore{}
		return devFeatureFlagStore, nil
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		devFeatureFlagStoreError = err
		return nil, err
	}
	devFeatureFlagStore = postgresDevFeatureFlagStore{pool: pool}
	return devFeatureFlagStore, nil
}

// refreshDevFeatureFlagState reloads persisted feature-flag targets into the
// process cache used by routeAllowed and session payload builders. Callers pass
// their request context so database delays or cancellations fail the current
// access decision closed instead of using stale default state.
func refreshDevFeatureFlagState(ctx context.Context) error {
	store, err := currentDevFeatureFlagStore(ctx)
	if err != nil {
		return err
	}
	state, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}
	devFeatureFlagStateMu.Lock()
	defer devFeatureFlagStateMu.Unlock()
	devFeatureFlagState = state
	devFeatureFlagStateLoaded = true
	devFeatureFlagStateLoadAttempted = true
	return nil
}

// cloneDevFeatureFlagState deep-copies the nested feature-flag target map used
// by DEV routing. Store implementations use it at read boundaries so callers
// can inspect state without sharing mutable map references.
func cloneDevFeatureFlagState(state map[string]map[devFeatureFlagTargetKey]bool) map[string]map[devFeatureFlagTargetKey]bool {
	cloned := make(map[string]map[devFeatureFlagTargetKey]bool, len(state))
	for flagKey, targets := range state {
		clonedTargets := make(map[devFeatureFlagTargetKey]bool, len(targets))
		for target, enabled := range targets {
			clonedTargets[target] = enabled
		}
		cloned[flagKey] = clonedTargets
	}
	return cloned
}

// initialDevFeatureFlagState builds the in-memory default target matrix for every flag.
func initialDevFeatureFlagState() map[string]map[devFeatureFlagTargetKey]bool {
	state := make(map[string]map[devFeatureFlagTargetKey]bool, len(devFeatureFlagRegistry))
	for _, definition := range devFeatureFlagRegistry {
		targets := make(map[devFeatureFlagTargetKey]bool)
		for _, persona := range orderedDevFeatureFlagPersonas() {
			targets[devFeatureFlagTargetKey{TargetType: "persona", TargetID: persona.ID}] = definition.DefaultEnabled
		}
		for _, site := range orderedDevFeatureFlagSites() {
			targets[devFeatureFlagTargetKey{TargetType: "site", TargetID: site.ID}] = definition.DefaultEnabled
		}
		state[definition.Key] = targets
	}
	return state
}

// orderedDevFeatureFlagPersonas returns editable non-IT persona targets in UI order.
func orderedDevFeatureFlagPersonas() []devFeatureTarget {
	targets := make([]devFeatureTarget, 0, len(devPersonaOrder)-1)
	for _, id := range devPersonaOrder {
		if id == "it_admin" {
			continue
		}
		config, ok := devPersonaConfigs[id]
		if !ok {
			continue
		}
		targets = append(targets, devFeatureTarget{ID: config.Persona.ID, Label: config.Persona.Label})
	}
	return targets
}

// orderedDevFeatureFlagSites returns editable site targets in shell/site order.
func orderedDevFeatureFlagSites() []devFeatureTarget {
	targets := make([]devFeatureTarget, 0, len(devSiteOrder))
	for _, id := range devSiteOrder {
		site := siteByID(id)
		targets = append(targets, devFeatureTarget{ID: site.ID, Label: site.Name})
	}
	return targets
}

// ensureDevFeatureFlagStateLoaded performs one lazy store snapshot attempt for route checks.
func ensureDevFeatureFlagStateLoaded(ctx context.Context) {
	devFeatureFlagStateMu.Lock()
	loaded := devFeatureFlagStateLoaded
	attempted := devFeatureFlagStateLoadAttempted
	if !loaded && !attempted {
		devFeatureFlagStateLoadAttempted = true
	}
	devFeatureFlagStateMu.Unlock()
	if loaded || attempted {
		return
	}
	_ = refreshDevFeatureFlagState(ctx)
}

// devFeatureFlagTargetEnabled resolves a stored target override with a flag default fallback.
func devFeatureFlagTargetEnabled(ctx context.Context, flagKey string, targetType string, targetID string, fallback bool) bool {
	ensureDevFeatureFlagStateLoaded(ctx)
	devFeatureFlagStateMu.Lock()
	defer devFeatureFlagStateMu.Unlock()
	if targets, ok := devFeatureFlagState[flagKey]; ok {
		if enabled, ok := targets[devFeatureFlagTargetKey{TargetType: targetType, TargetID: targetID}]; ok {
			if !devFeatureFlagStateLoaded {
				return false
			}
			return enabled
		}
	}
	if !devFeatureFlagStateLoaded {
		return false
	}
	return fallback
}

// devFeatureFlagEffective computes whether a route-gated feature is enabled for a persona/site session.
func devFeatureFlagEffective(ctx context.Context, definition devFeatureFlagDefinition, config devPersonaConfig) bool {
	if config.Persona.ID == "it_admin" {
		return true
	}
	personaEnabled := devFeatureFlagTargetEnabled(ctx, definition.Key, "persona", config.Persona.ID, definition.DefaultEnabled)
	siteEnabled := devFeatureFlagTargetEnabled(ctx, definition.Key, "site", config.CurrentSite.ID, definition.DefaultEnabled)
	return definition.DefaultEnabled && personaEnabled && siteEnabled
}

// devFeatureFlagIndicators returns read-only session indicators for feature-flag state.
func devFeatureFlagIndicators(ctx context.Context, definition devFeatureFlagDefinition, config devPersonaConfig) []devFeatureFlagIndicator {
	if config.Persona.ID == "it_admin" {
		return []devFeatureFlagIndicator{
			{
				Key:         definition.Key,
				Label:       definition.Label,
				TargetType:  "persona",
				TargetID:    "it_admin",
				TargetLabel: "IT Admin",
				Enabled:     true,
				ReadOnly:    true,
			},
		}
	}
	return []devFeatureFlagIndicator{
		{
			Key:         definition.Key,
			Label:       definition.Label,
			TargetType:  "persona",
			TargetID:    config.Persona.ID,
			TargetLabel: config.Persona.Label,
			Enabled:     devFeatureFlagTargetEnabled(ctx, definition.Key, "persona", config.Persona.ID, definition.DefaultEnabled),
			ReadOnly:    true,
		},
		{
			Key:         definition.Key,
			Label:       definition.Label,
			TargetType:  "site",
			TargetID:    config.CurrentSite.ID,
			TargetLabel: config.CurrentSite.Name,
			Enabled:     devFeatureFlagTargetEnabled(ctx, definition.Key, "site", config.CurrentSite.ID, definition.DefaultEnabled),
			ReadOnly:    true,
		},
	}
}

// devFeatureAvailabilities returns feature flag state summaries included in the DEV session payload.
func devFeatureAvailabilities(ctx context.Context, config devPersonaConfig) []devFeatureAvailability {
	availability := make([]devFeatureAvailability, 0, len(devFeatureFlagRegistry))
	for _, definition := range devFeatureFlagRegistry {
		availability = append(availability, devFeatureAvailability{
			Key:        definition.Key,
			Label:      definition.Label,
			Enabled:    devFeatureFlagEffective(ctx, definition, config),
			Indicators: devFeatureFlagIndicators(ctx, definition, config),
		})
	}
	return availability
}

// routeFeatureEnabled reports whether a route is currently enabled for a DEV persona config.
func routeFeatureEnabled(ctx context.Context, config devPersonaConfig, path string) bool {
	definition, ok := devFeatureFlagDefinitionForRoute(path)
	if !ok {
		return true
	}
	return devFeatureFlagEffective(ctx, definition, config)
}

// featureFilteredRoutes removes disabled feature routes from the session's allowed route list.
func featureFilteredRoutes(ctx context.Context, config devPersonaConfig) []string {
	routes := make([]string, 0, len(config.Allowed))
	for _, route := range config.Allowed {
		if routeFeatureEnabled(ctx, config, route) {
			routes = append(routes, route)
		}
	}
	return routes
}

// featureFilteredLandingPath chooses a landing path that remains accessible after feature filtering.
func featureFilteredLandingPath(ctx context.Context, config devPersonaConfig) string {
	if routeAllowed(ctx, config, config.LandingPath) {
		return config.LandingPath
	}
	filtered := featureFilteredRoutes(ctx, config)
	if len(filtered) > 0 {
		return filtered[0]
	}
	return "/dashboard"
}

// buildDevFeatureFlagsPayload builds the IT Admin feature flag management API response.
func buildDevFeatureFlagsPayload(ctx context.Context, config devPersonaConfig) devFeatureFlagsPayload {
	flags := make([]devFeatureFlagPayload, 0, len(devFeatureFlagRegistry))
	for _, definition := range devFeatureFlagRegistry {
		flags = append(flags, buildDevFeatureFlagPayload(ctx, definition))
	}
	return devFeatureFlagsPayload{
		PageID:      "feature-flags",
		Persona:     config.Persona,
		Shell:       config.Shell,
		GeneratedAt: "2026-05-12T12:00:00Z",
		Flags:       flags,
		Personas:    orderedDevFeatureFlagPersonas(),
		Sites:       orderedDevFeatureFlagSites(),
	}
}

// buildDevFeatureFlagPayload builds one flag row with editable target state and IT Admin override state.
func buildDevFeatureFlagPayload(ctx context.Context, definition devFeatureFlagDefinition) devFeatureFlagPayload {
	personaTargets := make([]devFeatureTargetState, 0, len(devPersonaOrder)-1)
	for _, persona := range orderedDevFeatureFlagPersonas() {
		personaTargets = append(personaTargets, devFeatureTargetState{
			ID:       persona.ID,
			Label:    persona.Label,
			Enabled:  devFeatureFlagTargetEnabled(ctx, definition.Key, "persona", persona.ID, definition.DefaultEnabled),
			ReadOnly: false,
		})
	}

	siteTargets := make([]devFeatureTargetState, 0, len(devSiteOrder))
	for _, site := range orderedDevFeatureFlagSites() {
		siteTargets = append(siteTargets, devFeatureTargetState{
			ID:       site.ID,
			Label:    site.Label,
			Enabled:  devFeatureFlagTargetEnabled(ctx, definition.Key, "site", site.ID, definition.DefaultEnabled),
			ReadOnly: false,
		})
	}

	return devFeatureFlagPayload{
		Key:            definition.Key,
		Label:          definition.Label,
		Description:    definition.Description,
		FeatureRoute:   definition.FeatureRoute,
		Routes:         slices.Clone(definition.Routes),
		DefaultEnabled: definition.DefaultEnabled,
		EffectiveForIT: true,
		PersonaTargets: append([]devFeatureTargetState{
			{ID: "it_admin", Label: "IT Admin", Enabled: true, ReadOnly: true},
		}, personaTargets...),
		SiteTargets: siteTargets,
		ActiveIndicators: []devFeatureFlagIndicator{
			{
				Key:         definition.Key,
				Label:       definition.Label,
				TargetType:  "persona",
				TargetID:    "it_admin",
				TargetLabel: "IT Admin",
				Enabled:     true,
				ReadOnly:    true,
			},
		},
	}
}

// validDevFeatureFlagTarget validates editable persona/site targets for a feature flag mutation.
func validDevFeatureFlagTarget(target devFeatureFlagTargetUpdate) bool {
	switch target.TargetType {
	case "persona":
		if target.TargetID == "it_admin" {
			return false
		}
		_, ok := devPersonaConfigs[target.TargetID]
		return ok
	case "site":
		_, ok := devSiteCatalog[target.TargetID]
		return ok
	default:
		return false
	}
}

// updateDevFeatureFlagTargets persists target state changes and refreshes the DEV flag cache when possible.
func updateDevFeatureFlagTargets(ctx context.Context, flagKey string, updates []devFeatureFlagTargetUpdate, actorID string) error {
	store, err := currentDevFeatureFlagStore(ctx)
	if err != nil {
		return err
	}
	if err := store.UpdateTargets(ctx, flagKey, updates, actorID); err != nil {
		return err
	}
	_ = refreshDevFeatureFlagState(ctx)
	return nil
}

// ResetDevFeatureFlagStateForTest restores DEV feature flags to their documented defaults.
// Tests call this helper because DEV feature flag updates intentionally mutate package-global
// state to mimic a running single-process local dashboard.
func ResetDevFeatureFlagStateForTest() {
	devFeatureFlagStateMu.Lock()
	defer devFeatureFlagStateMu.Unlock()
	devFeatureFlagState = initialDevFeatureFlagState()
	devFeatureFlagStateLoaded = false
	devFeatureFlagStateLoadAttempted = false
	devFeatureFlagStoreMu.Lock()
	defer devFeatureFlagStoreMu.Unlock()
	devFeatureFlagStore = nil
	devFeatureFlagStoreError = nil
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

func buildDevSessionPayload(ctx context.Context, config devPersonaConfig) devSessionPayload {
	persona := config.Persona
	return devSessionPayload{
		Environment:     "development",
		Authenticated:   true,
		Authorized:      true,
		CurrentPersona:  &persona,
		Personas:        orderedDevPersonas(),
		LandingPath:     featureFilteredLandingPath(ctx, config),
		AllowedRoutes:   featureFilteredRoutes(ctx, config),
		FeatureFlags:    devFeatureAvailabilities(ctx, config),
		Shell:           config.Shell,
		DefaultSiteID:   config.DefaultSite.ID,
		DefaultSiteName: config.DefaultSite.Name,
		CurrentSiteID:   config.CurrentSite.ID,
		CurrentSiteName: config.CurrentSite.Name,
		VisibleSites:    config.VisibleSites,
	}
}

type devSessionIdentity struct {
	Config              devPersonaConfig
	Breakglass          bool
	BreakglassAccountID string
}

func resolveAuthenticatedDevSession(r *http.Request) (devSessionIdentity, bool) {
	cookie, err := r.Cookie(devSessionCookieName)
	if err != nil {
		return devSessionIdentity{}, false
	}
	value := strings.TrimSpace(cookie.Value)
	if strings.HasPrefix(value, breakglassSessionCookiePrefix) {
		accountID := strings.TrimPrefix(value, breakglassSessionCookiePrefix)
		accounts, err := configuredBreakglassAccounts()
		if err != nil {
			return devSessionIdentity{}, false
		}
		account, ok := accounts[accountID]
		if !ok {
			return devSessionIdentity{}, false
		}
		config, ok := devPersonaConfigs[account.PersonaID]
		if !ok {
			return devSessionIdentity{}, false
		}
		return devSessionIdentity{Config: config, Breakglass: true, BreakglassAccountID: account.AccountID}, true
	}

	config, ok := devPersonaConfigs[value]
	if !ok {
		return devSessionIdentity{}, false
	}
	return devSessionIdentity{Config: config}, true
}

func resolveAuthenticatedDevPersona(r *http.Request) (devPersonaConfig, bool) {
	identity, ok := resolveAuthenticatedDevSession(r)
	if !ok {
		return devPersonaConfig{}, false
	}
	return identity.Config, true
}

func routeAllowed(ctx context.Context, config devPersonaConfig, path string) bool {
	return slices.Contains(config.Allowed, path) && routeFeatureEnabled(ctx, config, path)
}

func writeDevSessionCookie(w http.ResponseWriter, r *http.Request, personaID string) {
	writeDevSessionCookieValue(w, r, personaID)
}

func writeDevSessionCookieValue(w http.ResponseWriter, r *http.Request, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     devSessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureDevSessionCookie(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(12 * time.Hour),
	})
}

func clearDevSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     devSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureDevSessionCookie(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func devModeEnabled() bool {
	return currentAppEnvironment() == "development"
}

func currentAppEnvironment() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
}

func devSessionConsumerEnabled(r *http.Request) bool {
	if devModeEnabled() {
		return true
	}
	if !breakglassModeEnabled() {
		return false
	}
	identity, ok := resolveAuthenticatedDevSession(r)
	return ok && identity.Breakglass
}

func secureDevSessionCookie(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	return currentAppEnvironment() == "staging"
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
		return "Search common area phones and classroom shared lines across the district. Directory scope changes result order only."
	case "department":
		return "Search department shared lines and call queues across the district. Directory scope changes result order only."
	default:
		return "Search people and common area phones across the district. Directory scope changes result order only."
	}
}

const devDirectoryScopeDistrictWide = "district-wide"

func defaultPhoneDirectoryScopeID(config devPersonaConfig) string {
	switch config.Persona.ID {
	case "it_admin", "human_resources":
		return devDirectoryScopeDistrictWide
	default:
		if config.CurrentSite.ID != "" {
			return config.CurrentSite.ID
		}
		return config.DefaultSite.ID
	}
}

func resolvePhoneDirectoryScope(config devPersonaConfig, requestedScopeID string) (string, string) {
	scopeID := strings.TrimSpace(requestedScopeID)
	if scopeID == "" {
		scopeID = defaultPhoneDirectoryScopeID(config)
	}
	if scopeID == devDirectoryScopeDistrictWide {
		return devDirectoryScopeDistrictWide, "District-wide"
	}
	if site, ok := devSiteCatalog[scopeID]; ok {
		return site.ID, site.Name
	}

	defaultScopeID := defaultPhoneDirectoryScopeID(config)
	if defaultScopeID == devDirectoryScopeDistrictWide {
		return devDirectoryScopeDistrictWide, "District-wide"
	}
	site := devSiteCatalog[defaultScopeID]
	return site.ID, site.Name
}

func phoneDirectoryScopeOptions() []directoryScopeOption {
	options := []directoryScopeOption{{ID: devDirectoryScopeDistrictWide, Label: "District-wide"}}
	for _, siteID := range devSiteOrder {
		site, ok := devSiteCatalog[siteID]
		if !ok {
			continue
		}
		options = append(options, directoryScopeOption{ID: site.ID, Label: site.Name})
	}
	return options
}

func phoneDirectorySiteOrder() map[string]int {
	siteOrder := make(map[string]int, len(devSiteOrder))
	for index, siteID := range devSiteOrder {
		siteOrder[siteID] = index
	}
	return siteOrder
}

func searchPhoneDirectory(query string, mode string, directoryScopeID string) []phoneDirectorySearchResult {
	siteOrderByID := phoneDirectorySiteOrder()
	normalizedQuery := normalizeSearchValue(query)
	ranked := make([]rankedPhoneDirectoryResult, 0, len(devPhoneDirectoryEntries))
	for _, entry := range devPhoneDirectoryEntries {
		siteOrder, ok := siteOrderByID[entry.SiteID]
		if !ok {
			siteOrder = len(siteOrderByID) + 1
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
		if directoryScopeID != devDirectoryScopeDistrictWide && entry.SiteID == directoryScopeID {
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
		if left.SiteOrder != right.SiteOrder {
			return left.SiteOrder < right.SiteOrder
		}
		if left.TypeRank != right.TypeRank {
			return left.TypeRank < right.TypeRank
		}
		if left.MatchRank != right.MatchRank {
			return left.MatchRank < right.MatchRank
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
