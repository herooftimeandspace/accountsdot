package web_test

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

type devSessionResponse struct {
	Environment     string `json:"environment"`
	Authenticated   bool   `json:"authenticated"`
	Authorized      bool   `json:"authorized"`
	DefaultSiteID   string `json:"default_site_id"`
	DefaultSiteName string `json:"default_site_name"`
	CurrentSiteID   string `json:"current_site_id"`
	CurrentSiteName string `json:"current_site_name"`
	VisibleSites    []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"visible_sites"`
	CurrentPersona *struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		DisplayName string `json:"display_name"`
	} `json:"current_persona,omitempty"`
	LandingPath         string   `json:"landing_path"`
	AllowedRoutes       []string `json:"allowed_routes"`
	AuthenticationMode  string   `json:"authentication_mode"`
	BreakglassAccountID string   `json:"breakglass_account_id"`
	FeatureFlags        []struct {
		Key        string `json:"key"`
		Label      string `json:"label"`
		Enabled    bool   `json:"enabled"`
		Indicators []struct {
			TargetType  string `json:"target_type"`
			TargetID    string `json:"target_id"`
			TargetLabel string `json:"target_label"`
			Enabled     bool   `json:"enabled"`
			ReadOnly    bool   `json:"read_only"`
		} `json:"indicators"`
	} `json:"feature_flags"`
	Personas []struct {
		ID string `json:"id"`
	} `json:"personas"`
}

type featureFlagsResponse struct {
	PageID string `json:"page_id"`
	Flags  []struct {
		Key            string   `json:"key"`
		Label          string   `json:"label"`
		FeatureRoute   string   `json:"feature_route"`
		Routes         []string `json:"routes"`
		EffectiveForIT bool     `json:"effective_for_it_admin"`
		PersonaTargets []struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		} `json:"persona_targets"`
		SiteTargets []struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		} `json:"site_targets"`
		ActiveIndicators []struct {
			TargetType string `json:"target_type"`
			TargetID   string `json:"target_id"`
			Enabled    bool   `json:"enabled"`
			ReadOnly   bool   `json:"read_only"`
		} `json:"active_indicators"`
	} `json:"flags"`
}

type featureFlagResponse struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	EffectiveForIT bool   `json:"effective_for_it_admin"`
	PersonaTargets []struct {
		ID       string `json:"id"`
		Label    string `json:"label"`
		Enabled  bool   `json:"enabled"`
		ReadOnly bool   `json:"read_only"`
	} `json:"persona_targets"`
	SiteTargets []struct {
		ID       string `json:"id"`
		Label    string `json:"label"`
		Enabled  bool   `json:"enabled"`
		ReadOnly bool   `json:"read_only"`
	} `json:"site_targets"`
	ActiveIndicators []struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		Enabled    bool   `json:"enabled"`
		ReadOnly   bool   `json:"read_only"`
	} `json:"active_indicators"`
}

type myProfileResponse struct {
	PageID  string `json:"page_id"`
	Profile struct {
		LegalName          string `json:"legal_name"`
		PreferredFirstName string `json:"preferred_first_name"`
		PreferredLastName  string `json:"preferred_last_name"`
		DisplayName        string `json:"display_name"`
		Pronouns           string `json:"pronouns"`
		Editable           bool   `json:"editable"`
	} `json:"profile"`
}

type dataQualityResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		Title string `json:"title"`
		Queue struct {
			Rows []struct {
				Issue      string `json:"issue"`
				Owner      string `json:"owner"`
				NextAction string `json:"next_action"`
			} `json:"rows"`
		} `json:"queue"`
	} `json:"page"`
	Hotspots map[string]struct {
		NodeID string `json:"node_id"`
	} `json:"hotspots"`
}

type errorResponse struct {
	Code string `json:"code"`
}

type phoneDirectoryResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		Mode                  string `json:"mode"`
		Query                 string `json:"query"`
		CurrentSiteID         string `json:"current_site_id"`
		CurrentSiteName       string `json:"current_site_name"`
		DirectoryScopeID      string `json:"directory_scope_id"`
		DirectoryScopeLabel   string `json:"directory_scope_label"`
		DirectoryScopeOptions []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"directory_scope_options"`
		SelectedResult *struct {
			ID string `json:"id"`
		} `json:"selected_result,omitempty"`
		Results []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			TypeLabel       string `json:"type_label"`
			SiteID          string `json:"site_id"`
			SiteName        string `json:"site_name"`
			Title           string `json:"title"`
			Extension       string `json:"extension"`
			ExtensionLength int    `json:"extension_length"`
			ExtensionValid  bool   `json:"extension_valid"`
		} `json:"results"`
	} `json:"page"`
}

type globalSearchResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		Query  string `json:"query"`
		Groups []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Results []struct {
				ID          string `json:"id"`
				Type        string `json:"type"`
				Title       string `json:"title"`
				Subtitle    string `json:"subtitle"`
				Context     string `json:"context"`
				Destination string `json:"destination"`
				Source      string `json:"source"`
			} `json:"results"`
		} `json:"groups"`
	} `json:"page"`
}

type onboardingResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManageManual bool `json:"can_manage_manual"`
		Rows            []struct {
			Kind                 string `json:"kind"`
			DateAdded            string `json:"date_added"`
			DateAddedReason      string `json:"date_added_reason"`
			StartDate            string `json:"start_date"`
			EffectiveDate        string `json:"effective_date"`
			LeadTimeWarning      bool   `json:"lead_time_warning"`
			Person               string `json:"person"`
			SiteID               string `json:"site_id"`
			Site                 string `json:"site"`
			RoomID               string `json:"room_id"`
			RoomName             string `json:"room_name"`
			CanUpdateRoom        bool   `json:"can_update_room"`
			ManualDraftID        string `json:"manual_draft_id"`
			WorkflowStatus       string `json:"workflow_status"`
			ChangeReason         string `json:"change_reason"`
			LateStart            bool   `json:"late_start"`
			ScheduledFor         string `json:"scheduled_for"`
			ValidityState        string `json:"validity_state"`
			InvalidReason        string `json:"invalid_reason"`
			CanDeleteManualEntry bool   `json:"can_delete_manual_entry"`
			AssignedEmail        string `json:"assigned_email"`
			EmployeeNumber       string `json:"employee_number"`
			LinkedEscapeRecord   *struct {
				ID             string `json:"id"`
				Person         string `json:"person"`
				Site           string `json:"site"`
				AssignedEmail  string `json:"assigned_email"`
				EmployeeNumber string `json:"employee_number"`
				StartDate      string `json:"start_date"`
				CurrentStep    string `json:"current_step"`
				WorkflowStatus string `json:"workflow_status"`
			} `json:"linked_escape_record"`
			WorkflowSteps []struct {
				Name    string `json:"name"`
				Status  string `json:"status"`
				Detail  string `json:"detail"`
				Actions []struct {
					Label      string `json:"label"`
					Resolution string `json:"resolution"`
					System     string `json:"system"`
					Href       string `json:"href"`
				} `json:"actions"`
			} `json:"workflow_steps"`
		} `json:"rows"`
	} `json:"page"`
	Form struct {
		PreferredDevices      []string `json:"preferred_devices"`
		RequestedAeriesAccess []string `json:"requested_aeries_access"`
		Rooms                 []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			SiteID string `json:"site_id"`
		} `json:"rooms"`
	} `json:"form"`
}

type onboardingRoomUpdateResponse struct {
	Row struct {
		ID       string `json:"id"`
		SiteID   string `json:"site_id"`
		RoomID   string `json:"room_id"`
		RoomName string `json:"room_name"`
	} `json:"row"`
	Rows []struct {
		ID     string `json:"id"`
		SiteID string `json:"site_id"`
		RoomID string `json:"room_id"`
	} `json:"rows"`
}

type onboardingDraftResponse struct {
	Draft struct {
		ID                   string `json:"id"`
		Status               string `json:"status"`
		StartDate            string `json:"start_date"`
		EffectiveDate        string `json:"effective_date"`
		FirstName            string `json:"first_name"`
		LastName             string `json:"last_name"`
		PersonalEmail        string `json:"personal_email"`
		PersonalPhone        string `json:"personal_phone"`
		GeneratedEmail       string `json:"generated_email"`
		GeneratedEmployeeID  string `json:"generated_employee_id"`
		ChangeReason         string `json:"change_reason"`
		LateStart            bool   `json:"late_start"`
		ScheduledFor         string `json:"scheduled_for"`
		ValidityState        string `json:"validity_state"`
		InvalidReason        string `json:"invalid_reason"`
		CanDeleteManualEntry bool   `json:"can_delete_manual_entry"`
		LinkedEscapeRecord   *struct {
			ID             string `json:"id"`
			Person         string `json:"person"`
			Site           string `json:"site"`
			AssignedEmail  string `json:"assigned_email"`
			EmployeeNumber string `json:"employee_number"`
			StartDate      string `json:"start_date"`
			CurrentStep    string `json:"current_step"`
			WorkflowStatus string `json:"workflow_status"`
		} `json:"linked_escape_record"`
		MissingFields []string `json:"missing_fields"`
	} `json:"draft"`
	Rows []struct {
		Kind            string `json:"kind"`
		DateAdded       string `json:"date_added"`
		DateAddedReason string `json:"date_added_reason"`
		WorkflowStatus  string `json:"workflow_status"`
		AssignedEmail   string `json:"assigned_email"`
		EmployeeNumber  string `json:"employee_number"`
		WorkflowSteps   []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Actions []struct {
				Href string `json:"href"`
			} `json:"actions"`
		} `json:"workflow_steps"`
	} `json:"rows"`
}

type offboardingResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManageEndDates bool `json:"can_manage_end_dates"`
		CanManageManual   bool `json:"can_manage_manual"`
		ShowEmployeeIDs   bool `json:"show_employee_ids"`
		Rows              []struct {
			ID              string `json:"id"`
			Kind            string `json:"kind"`
			Person          string `json:"person"`
			Email           string `json:"email"`
			EmployeeID      string `json:"employee_id"`
			SiteID          string `json:"site_id"`
			EndDate         string `json:"end_date"`
			EndDateSource   string `json:"end_date_source"`
			EndDateEditable bool   `json:"end_date_editable"`
			Status          string `json:"status"`
			Warning         string `json:"warning"`
			Actions         []struct {
				Name       string `json:"name"`
				Owner      string `json:"owner"`
				Status     string `json:"status"`
				Detail     string `json:"detail"`
				Resolution string `json:"resolution"`
				Links      []struct {
					Href string `json:"href"`
				} `json:"links"`
			} `json:"actions"`
		} `json:"rows"`
	} `json:"page"`
}

type offboardingCandidatesResponse struct {
	Candidates []struct {
		ID              string `json:"id"`
		Kind            string `json:"kind"`
		Person          string `json:"person"`
		Email           string `json:"email"`
		EmployeeID      string `json:"employee_id"`
		TerminationDate string `json:"termination_date"`
	} `json:"candidates"`
}

type offboardingScheduleResponse struct {
	Action struct {
		Kind         string `json:"kind"`
		PersonID     string `json:"person_id"`
		Person       string `json:"person"`
		ScheduledFor string `json:"scheduled_for"`
		ActorID      string `json:"actor_id"`
		Mode         string `json:"mode"`
		Status       string `json:"status"`
	} `json:"action"`
}

type departingSeniorsResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManage         bool   `json:"can_manage"`
		SchoolYear        string `json:"school_year"`
		GraduationYear    string `json:"graduation_year"`
		SchoolYearOptions []struct {
			ID             string `json:"id"`
			Label          string `json:"label"`
			GraduationYear string `json:"graduation_year"`
			Current        bool   `json:"current"`
		} `json:"school_year_options"`
		Rows []struct {
			ID                 string `json:"id"`
			FirstName          string `json:"first_name"`
			LastName           string `json:"last_name"`
			Email              string `json:"email"`
			StudentID          string `json:"student_id"`
			SchoolYear         string `json:"school_year"`
			GraduationYear     string `json:"graduation_year"`
			EndDate            string `json:"end_date"`
			EndDateSource      string `json:"end_date_source"`
			Status             string `json:"status"`
			CanOverrideEndDate bool   `json:"can_override_end_date"`
			CanDeprovision     bool   `json:"can_deprovision"`
			Deprovisioned      bool   `json:"deprovisioned"`
			OutstandingDevices []struct {
				AssetID  string `json:"asset_id"`
				Serial   string `json:"serial"`
				Type     string `json:"type"`
				Domain   string `json:"domain"`
				AssetURL string `json:"asset_url"`
			} `json:"outstanding_devices"`
		} `json:"rows"`
	} `json:"page"`
}

type roomMovesResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManageDistrict bool `json:"can_manage_district"`
		ScopeSite         struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"scope_site"`
		Rooms []struct {
			ID     string `json:"id"`
			Label  string `json:"label"`
			SiteID string `json:"site_id"`
		} `json:"rooms"`
		People []struct {
			ID            string `json:"id"`
			SiteID        string `json:"site_id"`
			CurrentRoomID string `json:"current_room_id"`
			SourceRole    string `json:"source_role"`
		} `json:"people"`
		Rows []struct {
			ID                string   `json:"id"`
			DraftID           string   `json:"draft_id"`
			MoveType          string   `json:"move_type"`
			Person            string   `json:"person"`
			CurrentSiteID     string   `json:"current_site_id"`
			DestinationSiteID string   `json:"destination_site_id"`
			DestinationRoomID string   `json:"destination_room_id"`
			DestinationRoom   string   `json:"destination_room"`
			Phone             string   `json:"phone"`
			Author            string   `json:"author"`
			AuthorID          string   `json:"author_id"`
			State             string   `json:"state"`
			ScheduledFor      string   `json:"scheduled_for"`
			CanEdit           bool     `json:"can_edit"`
			CanCancel         bool     `json:"can_cancel"`
			Warning           string   `json:"warning"`
			AttentionReason   string   `json:"attention_reason"`
			AutomationOutcome string   `json:"automation_outcome"`
			ResolutionSteps   []string `json:"resolution_steps"`
			ExternalSystems   []string `json:"external_systems"`
		} `json:"rows"`
	} `json:"page"`
}

type roomMovesBulkDraftResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManageDistrict bool `json:"can_manage_district"`
		Rooms             []struct {
			ID     string `json:"id"`
			Label  string `json:"label"`
			SiteID string `json:"site_id"`
		} `json:"rooms"`
		Draft roomMoveDraftTestPayload `json:"draft"`
	} `json:"page"`
}

type roomMoveDraftTestResponse struct {
	Draft roomMoveDraftTestPayload `json:"draft"`
}

type roomMoveDraftTestPayload struct {
	ID                string   `json:"id"`
	Mode              string   `json:"mode"`
	Status            string   `json:"status"`
	ScopeSiteID       string   `json:"scope_site_id"`
	Author            string   `json:"author"`
	AuthorID          string   `json:"author_id"`
	CanEdit           bool     `json:"can_edit"`
	CanDelete         bool     `json:"can_delete"`
	CanManageDistrict bool     `json:"can_manage_district"`
	Warnings          []string `json:"warnings"`
	Rows              []struct {
		PersonID           string   `json:"person_id"`
		CurrentSiteID      string   `json:"current_site_id"`
		CurrentRoomID      string   `json:"current_room_id"`
		CurrentRoom        string   `json:"current_room"`
		DestinationSiteID  string   `json:"destination_site_id"`
		DestinationRoomID  string   `json:"destination_room_id"`
		DestinationRoom    string   `json:"destination_room"`
		DestinationRole    string   `json:"destination_role"`
		Action             string   `json:"action"`
		Warning            string   `json:"warning"`
		Phone              string   `json:"phone"`
		AttentionReason    string   `json:"attention_reason"`
		AutomationOutcome  string   `json:"automation_outcome"`
		ResolutionSteps    []string `json:"resolution_steps"`
		ExternalSystems    []string `json:"external_systems"`
		FallbackTicket     string   `json:"fallback_ticket"`
		FallbackTicketHref string   `json:"fallback_ticket_href"`
	} `json:"rows"`
}

type roomMoveCompletedJobsTestResponse struct {
	Jobs []struct {
		ID            string `json:"id"`
		SourceDraftID string `json:"source_draft_id"`
		ScopeSiteID   string `json:"scope_site_id"`
		RowCount      int    `json:"row_count"`
		CanRevert     bool   `json:"can_revert"`
		RevertDraftID string `json:"revert_draft_id"`
		RevertStatus  string `json:"revert_status"`
	} `json:"jobs"`
}

// decodeJSON is a shared test helper for DEV frontend handler tests that
// converts a recorder body into the typed response payload asserted by the
// caller. It fails the current test immediately when a handler returns invalid
// JSON so route regressions point at the response contract under test.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var payload T
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return payload
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func loginAsPersona(t *testing.T, handler http.Handler, personaID string) *http.Cookie {
	t.Helper()
	body, err := json.Marshal(map[string]string{"persona_id": personaID})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s returned %d", personaID, rec.Code)
	}
	cookie := findCookie(rec.Result().Cookies(), "wizard_dev_session")
	if cookie == nil {
		t.Fatalf("login %s did not set session cookie", personaID)
	}
	return cookie
}

func activateSharedMockPersona(t *testing.T, handler http.Handler, personaID string) (*httptest.ResponseRecorder, *http.Cookie) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"persona_id": personaID, "activate_mock_session": true})
	if err != nil {
		t.Fatalf("marshal shared login request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, findCookie(rec.Result().Cookies(), "wizard_dev_session")
}

func configureBreakglassForTest(t *testing.T, accountID string, token string) {
	t.Helper()
	hash := sha256.Sum256([]byte(token))
	envID := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", "@", "_").Replace(accountID))
	t.Setenv("BREAKGLASS_ACCOUNTS", accountID)
	t.Setenv("BREAKGLASS_TOKEN_SHA256_"+envID, hex.EncodeToString(hash[:]))
	t.Setenv("BREAKGLASS_ALLOWED_CIDRS", "10.23.0.0/16,10.19.100.0/24")
	web.ResetBreakglassAuditForTest()
	t.Cleanup(web.ResetBreakglassAuditForTest)
}

func breakglassLogin(t *testing.T, handler http.Handler, accountID string, token string, remoteAddr string) (*httptest.ResponseRecorder, *http.Cookie) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"account_id": accountID, "token": token})
	if err != nil {
		t.Fatalf("marshal breakglass login request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/breakglass/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, findCookie(rec.Result().Cookies(), "wizard_dev_session")
}

// isolateDevFeatureFlagState guards tests that mutate DEV feature flags by
// verifying defaults before the test, resetting state for the test body, and
// restoring defaults afterward. It prevents one route-permission test from
// leaking mock store state into later handler assertions.
func isolateDevFeatureFlagState(t *testing.T, handler http.Handler) {
	t.Helper()
	assertDevFeatureFlagsAtDefaults(t, handler)
	web.ResetDevFeatureFlagStateForTest()
	assertDevFeatureFlagsAtDefaults(t, handler)
	t.Cleanup(func() {
		web.ResetDevFeatureFlagStateForTest()
		assertDevFeatureFlagsAtDefaults(t, handler)
	})
}

// assertDevFeatureFlagsAtDefaults checks the authenticated DEV feature-flag API
// for the all-enabled baseline expected by most frontend handler tests. It
// decodes the response payload and fails immediately if any persona or site
// target was left disabled by a previous test.
func assertDevFeatureFlagsAtDefaults(t *testing.T, handler http.Handler) {
	t.Helper()
	itCookie := loginAsPersona(t, handler, "it_admin")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/feature-flags", nil)
	req.AddCookie(itCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("feature flag default guard returned %d, want 200: %s", rec.Code, rec.Body.String())
	}

	payload := decodeJSON[featureFlagsResponse](t, rec)
	if len(payload.Flags) == 0 {
		t.Fatal("feature flag default guard received no flags")
	}
	for _, flag := range payload.Flags {
		for _, target := range flag.PersonaTargets {
			if !target.Enabled {
				t.Fatalf("feature flag %q persona target %q leaked disabled state", flag.Key, target.ID)
			}
		}
		for _, target := range flag.SiteTargets {
			if !target.Enabled {
				t.Fatalf("feature flag %q site target %q leaked disabled state", flag.Key, target.ID)
			}
		}
	}
}

// updateDevFeatureFlagTargetForTest drives the same PUT route that the DEV
// frontend uses to change one feature-flag target. Tests use the returned typed
// payload to assert route access changes while keeping request construction and
// JSON decoding consistent.
func updateDevFeatureFlagTargetForTest(t *testing.T, handler http.Handler, cookie *http.Cookie, flagKey string, targetType string, targetID string, enabled bool) featureFlagResponse {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"targets": []map[string]any{
			{"target_type": targetType, "target_id": targetID, "enabled": enabled},
		},
	})
	if err != nil {
		t.Fatalf("marshal feature flag update: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/feature-flags/"+flagKey, bytes.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("feature flag update returned %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return decodeJSON[featureFlagResponse](t, rec)
}

func globalSearchHasResult(payload globalSearchResponse, groupID string, titleMatch string) bool {
	for _, group := range payload.Page.Groups {
		if group.ID != groupID {
			continue
		}
		for _, result := range group.Results {
			if strings.Contains(result.Title, titleMatch) {
				return true
			}
		}
	}
	return false
}

func phoneDirectoryHasTitle(payload phoneDirectoryResponse, title string) bool {
	for _, result := range payload.Page.Results {
		if result.Title == title {
			return true
		}
	}
	return false
}

// repoDoc reads a checked-in repository document for tests that assert durable
// documentation content. It resolves paths from the internal/web package so
// failures identify the specific missing or unreadable doc file.
func repoDoc(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(body)
}

func createAndFinalizeManualOnboarding(t *testing.T, handler http.Handler, cookie *http.Cookie, firstName string, lastName string) string {
	t.Helper()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
	createReq.AddCookie(cookie)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("draft create returned %d, want 201", createRec.Code)
	}
	created := decodeJSON[onboardingDraftResponse](t, createRec)
	body, err := json.Marshal(map[string]string{
		"start_date":              "2026-05-11",
		"ssn_last4":               "5678",
		"employee_type":           "Contractor",
		"classification":          "Certificated",
		"first_name":              firstName,
		"last_name":               lastName,
		"job_title":               "Counselor",
		"site_id":                 "district-office",
		"personal_email":          strings.ToLower(firstName + "." + lastName + "@example.com"),
		"personal_phone":          "7075550177",
		"preferred_device":        "Windows",
		"requested_aeries_access": "Counselor",
	})
	if err != nil {
		t.Fatalf("marshal draft: %v", err)
	}
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(body))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.AddCookie(cookie)
	updateRec := httptest.NewRecorder()
	handler.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("draft update returned %d, want 200", updateRec.Code)
	}
	finalizeReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID+"/finalize", nil)
	finalizeReq.AddCookie(cookie)
	finalizeRec := httptest.NewRecorder()
	handler.ServeHTTP(finalizeRec, finalizeReq)
	if finalizeRec.Code != http.StatusOK {
		t.Fatalf("finalize returned %d, want 200", finalizeRec.Code)
	}
	finalized := decodeJSON[onboardingDraftResponse](t, finalizeRec)
	return finalized.Draft.GeneratedEmail
}

func TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	web.ResetDevSharedMockSessionForTest()
	t.Cleanup(web.ResetDevSharedMockSessionForTest)
	web.ResetDevFeatureFlagStateForTest()
	t.Cleanup(web.ResetDevFeatureFlagStateForTest)
	web.ResetDevDepartingSeniorsStateForTest()
	t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})

	t.Run("session is anonymous before login", func(t *testing.T) {
		web.ResetDevSharedMockSessionForTest()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("session returned %d", rec.Code)
		}

		payload := decodeJSON[devSessionResponse](t, rec)
		if payload.Authenticated || payload.Authorized {
			t.Fatalf("expected anonymous session, got authenticated=%v authorized=%v", payload.Authenticated, payload.Authorized)
		}
		if len(payload.Personas) == 0 {
			t.Fatal("expected ordered personas in anonymous session response")
		}
	})

	t.Run("it admin login sets session and can load data quality", func(t *testing.T) {
		web.ResetDevSharedMockSessionForTest()
		cookie := loginAsPersona(t, handler, "it_admin")

		sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		sessionReq.AddCookie(cookie)
		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)
		if sessionRec.Code != http.StatusOK {
			t.Fatalf("session returned %d", sessionRec.Code)
		}

		sessionPayload := decodeJSON[devSessionResponse](t, sessionRec)
		if !sessionPayload.Authenticated || !sessionPayload.Authorized {
			t.Fatalf("expected authenticated authorized session, got authenticated=%v authorized=%v", sessionPayload.Authenticated, sessionPayload.Authorized)
		}
		if sessionPayload.CurrentPersona == nil || sessionPayload.CurrentPersona.ID != "it_admin" {
			t.Fatalf("expected it_admin persona, got %#v", sessionPayload.CurrentPersona)
		}
		if sessionPayload.LandingPath != "/dashboard/it-admin" {
			t.Fatalf("landing path = %q, want /dashboard/it-admin", sessionPayload.LandingPath)
		}
		if sessionPayload.DefaultSiteID != "clover-hs" || sessionPayload.CurrentSiteID != "clover-hs" {
			t.Fatalf("expected clover-hs site context, got default=%q current=%q", sessionPayload.DefaultSiteID, sessionPayload.CurrentSiteID)
		}
		if len(sessionPayload.VisibleSites) < 6 {
			t.Fatalf("expected district-wide visible sites in session payload, got %#v", sessionPayload.VisibleSites)
		}
		if !slices.Contains(sessionPayload.AllowedRoutes, "/data-quality") {
			t.Fatalf("expected /data-quality in allowed routes: %#v", sessionPayload.AllowedRoutes)
		}
		if !slices.Contains(sessionPayload.AllowedRoutes, "/reports/security-issues") {
			t.Fatalf("expected /reports/security-issues in allowed routes: %#v", sessionPayload.AllowedRoutes)
		}
		if slices.Contains(sessionPayload.AllowedRoutes, "/reports/ticketing-human-work") {
			t.Fatalf("retired ticketing-human-work report should not be in allowed routes: %#v", sessionPayload.AllowedRoutes)
		}

		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/data-quality", nil)
		pageReq.AddCookie(cookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusOK {
			t.Fatalf("data quality returned %d", pageRec.Code)
		}

		pagePayload := decodeJSON[dataQualityResponse](t, pageRec)
		if pagePayload.PageID != "data-quality" {
			t.Fatalf("page id = %q, want data-quality", pagePayload.PageID)
		}
		if pagePayload.Page.Title != "Data Quality" {
			t.Fatalf("page title = %q, want Data Quality", pagePayload.Page.Title)
		}
		wantActions := map[string]string{
			"Unmapped job title":              "Review in HR lifecycle",
			"Room mismatch":                   "Route to site owner",
			"Google-active / Aeries-inactive": "Review in Admin",
			"Missing mandatory field":         "Complete in Onboarding",
			"Site mismatch":                   "Review in HR lifecycle",
		}
		if len(pagePayload.Page.Queue.Rows) != len(wantActions) {
			t.Fatalf("data quality queue row count = %d, want %d", len(pagePayload.Page.Queue.Rows), len(wantActions))
		}
		for _, row := range pagePayload.Page.Queue.Rows {
			if wantActions[row.Issue] != row.NextAction {
				t.Fatalf("next action for %q = %q, want %q", row.Issue, row.NextAction, wantActions[row.Issue])
			}
			if strings.Contains(row.NextAction, "Mapping Dashboard") || strings.Contains(row.NextAction, "Map title") {
				t.Fatalf("data quality row %q exposes unsupported local action %q", row.Issue, row.NextAction)
			}
		}
		if pagePayload.Hotspots["refresh"].NodeID != "f104" {
			t.Fatalf("refresh hotspot node = %q, want f104", pagePayload.Hotspots["refresh"].NodeID)
		}
	})

	t.Run("dev persona session payloads enforce site cardinality", func(t *testing.T) {
		tests := []struct {
			personaID      string
			defaultSiteID  string
			visibleSiteIDs []string
		}{
			{personaID: "site_admin", defaultSiteID: "clover-hs", visibleSiteIDs: []string{"clover-hs"}},
			{personaID: "site_secretary", defaultSiteID: "clover-hs", visibleSiteIDs: []string{"clover-hs"}},
			{personaID: "device_wrangler", defaultSiteID: "franklin-ms", visibleSiteIDs: []string{"franklin-ms"}},
			{personaID: "faculty_staff", defaultSiteID: "clover-hs", visibleSiteIDs: []string{"clover-hs", "desert-view"}},
		}

		for _, tt := range tests {
			t.Run(tt.personaID, func(t *testing.T) {
				cookie := loginAsPersona(t, handler, tt.personaID)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
				req.AddCookie(cookie)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("session returned %d: %s", rec.Code, rec.Body.String())
				}

				payload := decodeJSON[devSessionResponse](t, rec)
				if payload.DefaultSiteID != tt.defaultSiteID || payload.CurrentSiteID != tt.defaultSiteID {
					t.Fatalf("site context = default:%q current:%q, want %q", payload.DefaultSiteID, payload.CurrentSiteID, tt.defaultSiteID)
				}
				gotSites := make([]string, 0, len(payload.VisibleSites))
				for _, site := range payload.VisibleSites {
					gotSites = append(gotSites, site.ID)
				}
				if !slices.Equal(gotSites, tt.visibleSiteIDs) {
					t.Fatalf("visible sites = %#v, want %#v", gotSites, tt.visibleSiteIDs)
				}
				if tt.personaID == "faculty_staff" {
					for _, route := range []string{"/student-data-cleanup", "/frequent-fliers", "/onboarding", "/offboarding", "/room-moves"} {
						if slices.Contains(payload.AllowedRoutes, route) {
							t.Fatalf("faculty/staff multi-site association exposed operational route %q in %#v", route, payload.AllowedRoutes)
						}
					}
				}
			})
		}
	})

	t.Run("breakglass login uses named local account and can load it admin route", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		rec, cookie := breakglassLogin(t, handler, "emergency-alex", "local-test-token", "10.23.4.5:62000")
		if rec.Code != http.StatusOK {
			t.Fatalf("breakglass login returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		if cookie == nil || !strings.HasPrefix(cookie.Value, "breakglass:") {
			t.Fatalf("breakglass login cookie = %#v, want breakglass-scoped session cookie", cookie)
		}
		sessionPayload := decodeJSON[devSessionResponse](t, rec)
		if !sessionPayload.Authenticated || !sessionPayload.Authorized {
			t.Fatalf("breakglass session authenticated=%v authorized=%v, want true/true", sessionPayload.Authenticated, sessionPayload.Authorized)
		}
		if sessionPayload.CurrentPersona == nil || sessionPayload.CurrentPersona.ID != "it_admin" {
			t.Fatalf("breakglass current persona = %#v, want local IT Admin persona", sessionPayload.CurrentPersona)
		}
		if !slices.Contains(sessionPayload.AllowedRoutes, "/data-quality") {
			t.Fatalf("breakglass allowed routes missing /data-quality: %#v", sessionPayload.AllowedRoutes)
		}

		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/data-quality", nil)
		pageReq.AddCookie(cookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusOK {
			t.Fatalf("breakglass data quality returned %d, want 200: %s", pageRec.Code, pageRec.Body.String())
		}

		audits := web.BreakglassAuditEventsForTest()
		if len(audits) != 2 {
			t.Fatalf("breakglass audit count = %d, want login attempt and access granted: %#v", len(audits), audits)
		}
		if audits[0].AccountID != "emergency-alex" || audits[0].Action != "login_attempt" || audits[0].Outcome != "allowed" || audits[0].SourceIP != "10.23.4.5" {
			t.Fatalf("login audit = %#v, want allowed emergency-alex from 10.23.4.5", audits[0])
		}
		if audits[1].Action != "access_granted" || audits[1].Outcome != "allowed" {
			t.Fatalf("access audit = %#v, want access_granted allowed", audits[1])
		}
	})

	t.Run("staging consumes only authenticated breakglass sessions", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		t.Setenv("APP_ENV", "staging")

		devLoginBody, err := json.Marshal(map[string]string{"persona_id": "it_admin"})
		if err != nil {
			t.Fatalf("marshal dev login request: %v", err)
		}
		devLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", bytes.NewReader(devLoginBody))
		devLoginRec := httptest.NewRecorder()
		handler.ServeHTTP(devLoginRec, devLoginReq)
		if devLoginRec.Code != http.StatusNotFound {
			t.Fatalf("staging dev persona login returned %d, want 404", devLoginRec.Code)
		}

		rec, cookie := breakglassLogin(t, handler, "emergency-alex", "local-test-token", "10.23.4.5:62000")
		if rec.Code != http.StatusOK {
			t.Fatalf("staging breakglass login returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		if cookie == nil || !cookie.Secure {
			t.Fatalf("staging breakglass cookie = %#v, want Secure breakglass session cookie", cookie)
		}
		loginPayload := decodeJSON[devSessionResponse](t, rec)
		if loginPayload.Environment != "staging" || loginPayload.AuthenticationMode != "breakglass" || loginPayload.BreakglassAccountID != "emergency-alex" {
			t.Fatalf("staging breakglass payload = %#v, want staging breakglass emergency-alex", loginPayload)
		}

		sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		sessionReq.AddCookie(cookie)
		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)
		if sessionRec.Code != http.StatusOK {
			t.Fatalf("staging breakglass session returned %d, want 200: %s", sessionRec.Code, sessionRec.Body.String())
		}
		sessionPayload := decodeJSON[devSessionResponse](t, sessionRec)
		if !sessionPayload.Authenticated || sessionPayload.Environment != "staging" || sessionPayload.AuthenticationMode != "breakglass" {
			t.Fatalf("staging session payload = %#v, want authenticated breakglass session", sessionPayload)
		}

		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/data-quality", nil)
		pageReq.AddCookie(cookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusOK {
			t.Fatalf("staging breakglass data quality returned %d, want 200: %s", pageRec.Code, pageRec.Body.String())
		}
	})

	t.Run("breakglass cookie is secure for HTTPS requests", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		body, err := json.Marshal(map[string]string{"account_id": "emergency-alex", "token": "local-test-token"})
		if err != nil {
			t.Fatalf("marshal breakglass login request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "https://wizard.example.test/api/v1/breakglass/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.23.4.5:62000"
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("HTTPS breakglass login returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		cookie := findCookie(rec.Result().Cookies(), "wizard_dev_session")
		if cookie == nil || !cookie.Secure {
			t.Fatalf("HTTPS breakglass cookie = %#v, want Secure session cookie", cookie)
		}
	})

	t.Run("breakglass denies source address before issuing a session", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		rec, cookie := breakglassLogin(t, handler, "emergency-alex", "local-test-token", "192.0.2.10:62000")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("breakglass denied-source login returned %d, want 403: %s", rec.Code, rec.Body.String())
		}
		if cookie != nil {
			t.Fatalf("denied source received session cookie: %#v", cookie)
		}
		payload := decodeJSON[errorResponse](t, rec)
		if payload.Code != "breakglass_source_denied" {
			t.Fatalf("denied source code = %q, want breakglass_source_denied", payload.Code)
		}
		audits := web.BreakglassAuditEventsForTest()
		if len(audits) != 1 || audits[0].FailureCode != "source_address_denied" || audits[0].Outcome != "denied" {
			t.Fatalf("denied source audit = %#v, want source_address_denied", audits)
		}
	})

	t.Run("breakglass denies unknown account without falling back to persona switcher", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		rec, cookie := breakglassLogin(t, handler, "it_admin", "local-test-token", "10.23.4.5:62000")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unknown breakglass login returned %d, want 401: %s", rec.Code, rec.Body.String())
		}
		if cookie != nil {
			t.Fatalf("unknown account received session cookie: %#v", cookie)
		}
		audits := web.BreakglassAuditEventsForTest()
		if len(audits) != 1 || audits[0].AccountID != "it_admin" || audits[0].FailureCode != "unknown_account" {
			t.Fatalf("unknown account audit = %#v, want unknown_account for it_admin", audits)
		}
	})

	t.Run("breakglass logout audits sign out", func(t *testing.T) {
		configureBreakglassForTest(t, "emergency-alex", "local-test-token")
		rec, cookie := breakglassLogin(t, handler, "emergency-alex", "local-test-token", "10.19.100.25:62000")
		if rec.Code != http.StatusOK || cookie == nil {
			t.Fatalf("breakglass login returned %d cookie %#v", rec.Code, cookie)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/logout", nil)
		req.RemoteAddr = "10.19.100.25:62000"
		req.AddCookie(cookie)
		logoutRec := httptest.NewRecorder()
		handler.ServeHTTP(logoutRec, req)
		if logoutRec.Code != http.StatusOK {
			t.Fatalf("breakglass logout returned %d, want 200", logoutRec.Code)
		}
		audits := web.BreakglassAuditEventsForTest()
		if len(audits) != 3 {
			t.Fatalf("breakglass audit count = %d, want login/access/sign_out: %#v", len(audits), audits)
		}
		if audits[2].Action != "sign_out" || audits[2].Outcome != "allowed" || audits[2].AccountID != "emergency-alex" {
			t.Fatalf("sign-out audit = %#v, want allowed emergency-alex sign_out", audits[2])
		}
	})

	t.Run("feature flags are it admin only and include read only it override", func(t *testing.T) {
		isolateDevFeatureFlagState(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/feature-flags", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous feature flags returned %d, want 401", rec.Code)
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/feature-flags", nil)
		req.AddCookie(siteCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site admin feature flags returned %d, want 403", rec.Code)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/feature-flags/onboarding", strings.NewReader(`{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false}]}`))
		updateReq.AddCookie(siteCookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusForbidden {
			t.Fatalf("site admin feature flag update returned %d, want 403", updateRec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/feature-flags", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it admin feature flags returned %d, want 200", rec.Code)
		}
		payload := decodeJSON[featureFlagsResponse](t, rec)
		if payload.PageID != "feature-flags" || len(payload.Flags) == 0 {
			t.Fatalf("unexpected feature flags payload: %#v", payload)
		}
		flagIndex := slices.IndexFunc(payload.Flags, func(flag struct {
			Key            string   `json:"key"`
			Label          string   `json:"label"`
			FeatureRoute   string   `json:"feature_route"`
			Routes         []string `json:"routes"`
			EffectiveForIT bool     `json:"effective_for_it_admin"`
			PersonaTargets []struct {
				ID       string `json:"id"`
				Label    string `json:"label"`
				Enabled  bool   `json:"enabled"`
				ReadOnly bool   `json:"read_only"`
			} `json:"persona_targets"`
			SiteTargets []struct {
				ID       string `json:"id"`
				Label    string `json:"label"`
				Enabled  bool   `json:"enabled"`
				ReadOnly bool   `json:"read_only"`
			} `json:"site_targets"`
			ActiveIndicators []struct {
				TargetType string `json:"target_type"`
				TargetID   string `json:"target_id"`
				Enabled    bool   `json:"enabled"`
				ReadOnly   bool   `json:"read_only"`
			} `json:"active_indicators"`
		}) bool {
			return flag.Key == "frequent_fliers"
		})
		if flagIndex < 0 {
			t.Fatalf("feature flags missing frequent_fliers: %#v", payload.Flags)
		}
		frequentFliers := payload.Flags[flagIndex]
		if !frequentFliers.EffectiveForIT || len(frequentFliers.PersonaTargets) == 0 || frequentFliers.PersonaTargets[0].ID != "it_admin" || !frequentFliers.PersonaTargets[0].ReadOnly {
			t.Fatalf("expected read-only IT Admin override first, got %#v", frequentFliers.PersonaTargets)
		}
		if len(frequentFliers.ActiveIndicators) == 0 || !frequentFliers.ActiveIndicators[0].ReadOnly || !frequentFliers.ActiveIndicators[0].Enabled {
			t.Fatalf("expected active read-only indicators, got %#v", frequentFliers.ActiveIndicators)
		}
	})

	t.Run("feature flag registry routes declare backend coverage or documented exceptions", func(t *testing.T) {
		type routeCoverage struct {
			APIPaths  []string
			Exception string
		}
		coverage := map[string]routeCoverage{
			"/dashboard/site-admin": {
				Exception: "Flagged route backend coverage exception: /dashboard/site-admin",
			},
			"/onboarding": {
				APIPaths: []string{"/api/v1/dev/pages/onboarding"},
			},
			"/offboarding": {
				APIPaths: []string{"/api/v1/dev/pages/offboarding"},
			},
			"/departing-seniors": {
				APIPaths: []string{"/api/v1/dev/pages/departing-seniors"},
			},
			"/room-moves": {
				APIPaths: []string{"/api/v1/dev/pages/room-moves"},
			},
			"/room-moves/bulk-draft": {
				APIPaths: []string{"/api/v1/dev/pages/room-moves/bulk-draft"},
			},
			"/phone-directory/by-person": {
				APIPaths: []string{"/api/v1/dev/pages/phone-directory/by-person"},
			},
			"/phone-directory/by-room": {
				APIPaths: []string{"/api/v1/dev/pages/phone-directory/by-room"},
			},
			"/phone-directory/by-department": {
				APIPaths: []string{"/api/v1/dev/pages/phone-directory/by-department"},
			},
			"/student-data-cleanup": {
				Exception: "Flagged route backend coverage exception: /student-data-cleanup",
			},
			"/frequent-fliers": {
				Exception: "Flagged route backend coverage exception: /frequent-fliers",
			},
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/feature-flags", nil)
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it admin feature flags returned %d, want 200", rec.Code)
		}
		payload := decodeJSON[featureFlagsResponse](t, rec)
		productRequirements := repoDoc(t, "docs/product/product-requirements.md")
		implementationPlan := repoDoc(t, "docs/planning/implementation-plan.md")

		seen := map[string]bool{}
		for _, flag := range payload.Flags {
			if flag.FeatureRoute == "" {
				t.Fatalf("feature flag %q has no feature route", flag.Key)
			}
			if len(flag.Routes) == 0 {
				t.Fatalf("feature flag %q has no controlled routes", flag.Key)
			}
			for _, route := range flag.Routes {
				routeCoverage, ok := coverage[route]
				if !ok {
					t.Fatalf("feature flag %q route %q has no backend coverage mapping or documented exception", flag.Key, route)
				}
				seen[route] = true
				if routeCoverage.Exception != "" {
					if !strings.Contains(productRequirements, routeCoverage.Exception) {
						t.Fatalf("%s missing from docs/product/product-requirements.md", routeCoverage.Exception)
					}
					if !strings.Contains(implementationPlan, routeCoverage.Exception) {
						t.Fatalf("%s missing from docs/planning/implementation-plan.md", routeCoverage.Exception)
					}
					continue
				}
				if len(routeCoverage.APIPaths) == 0 {
					t.Fatalf("feature flag %q route %q has neither API paths nor exception", flag.Key, route)
				}
				for _, apiPath := range routeCoverage.APIPaths {
					apiReq := httptest.NewRequest(http.MethodGet, apiPath, nil)
					apiReq.AddCookie(itCookie)
					apiRec := httptest.NewRecorder()
					handler.ServeHTTP(apiRec, apiReq)
					if apiRec.Code == http.StatusNotFound {
						t.Fatalf("feature flag %q route %q maps to missing backend API %q", flag.Key, route, apiPath)
					}
				}
			}
		}
		for route := range coverage {
			if !seen[route] {
				t.Fatalf("coverage mapping for %q no longer matches any feature flag registry route", route)
			}
		}
	})

	t.Run("feature flags gate non it allowed routes while preserving it override", func(t *testing.T) {
		isolateDevFeatureFlagState(t, handler)

		itCookie := loginAsPersona(t, handler, "it_admin")
		body, err := json.Marshal(map[string]any{
			"targets": []map[string]any{
				{"target_type": "persona", "target_id": "site_admin", "enabled": false},
			},
		})
		if err != nil {
			t.Fatalf("marshal feature flag update: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/feature-flags/onboarding", bytes.NewReader(body))
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("feature flag update returned %d, want 200: %s", rec.Code, rec.Body.String())
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		sessionReq.AddCookie(siteCookie)
		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)
		if sessionRec.Code != http.StatusOK {
			t.Fatalf("site admin session returned %d", sessionRec.Code)
		}
		siteSession := decodeJSON[devSessionResponse](t, sessionRec)
		if slices.Contains(siteSession.AllowedRoutes, "/onboarding") {
			t.Fatalf("site admin allowed routes still include onboarding: %#v", siteSession.AllowedRoutes)
		}
		flagIndex := slices.IndexFunc(siteSession.FeatureFlags, func(flag struct {
			Key        string `json:"key"`
			Label      string `json:"label"`
			Enabled    bool   `json:"enabled"`
			Indicators []struct {
				TargetType  string `json:"target_type"`
				TargetID    string `json:"target_id"`
				TargetLabel string `json:"target_label"`
				Enabled     bool   `json:"enabled"`
				ReadOnly    bool   `json:"read_only"`
			} `json:"indicators"`
		}) bool {
			return flag.Key == "onboarding"
		})
		if flagIndex < 0 || siteSession.FeatureFlags[flagIndex].Enabled {
			t.Fatalf("site admin onboarding feature = %#v, want disabled", siteSession.FeatureFlags)
		}
		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding", nil)
		pageReq.AddCookie(siteCookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusForbidden {
			t.Fatalf("site admin onboarding page returned %d, want 403", pageRec.Code)
		}

		itSessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		itSessionReq.AddCookie(itCookie)
		itSessionRec := httptest.NewRecorder()
		handler.ServeHTTP(itSessionRec, itSessionReq)
		itSession := decodeJSON[devSessionResponse](t, itSessionRec)
		if !slices.Contains(itSession.AllowedRoutes, "/onboarding") {
			t.Fatalf("it admin allowed routes lost onboarding: %#v", itSession.AllowedRoutes)
		}
	})

	t.Run("feature flag target updates are independent and forced APIs are denied", func(t *testing.T) {
		isolateDevFeatureFlagState(t, handler)

		itCookie := loginAsPersona(t, handler, "it_admin")
		updateTarget := func(targetType string, targetID string, enabled bool) featureFlagResponse {
			t.Helper()
			return updateDevFeatureFlagTargetForTest(t, handler, itCookie, "onboarding", targetType, targetID, enabled)
		}

		payload := updateTarget("persona", "human_resources", false)
		if payload.Key != "onboarding" {
			t.Fatalf("unexpected update payload: %#v", payload)
		}
		onboardingFlag := payload
		humanResourcesIndex := slices.IndexFunc(onboardingFlag.PersonaTargets, func(target struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		}) bool {
			return target.ID == "human_resources"
		})
		siteAdminIndex := slices.IndexFunc(onboardingFlag.PersonaTargets, func(target struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		}) bool {
			return target.ID == "site_admin"
		})
		districtOfficeIndex := slices.IndexFunc(onboardingFlag.SiteTargets, func(target struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		}) bool {
			return target.ID == "district-office"
		})
		if humanResourcesIndex < 0 || onboardingFlag.PersonaTargets[humanResourcesIndex].Enabled {
			t.Fatalf("human resources target = %#v, want independently disabled", onboardingFlag.PersonaTargets)
		}
		if siteAdminIndex < 0 || !onboardingFlag.PersonaTargets[siteAdminIndex].Enabled {
			t.Fatalf("site admin target changed unexpectedly: %#v", onboardingFlag.PersonaTargets)
		}
		if districtOfficeIndex < 0 || !onboardingFlag.SiteTargets[districtOfficeIndex].Enabled {
			t.Fatalf("district office site target changed unexpectedly: %#v", onboardingFlag.SiteTargets)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		sessionReq.AddCookie(hrCookie)
		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)
		if sessionRec.Code != http.StatusOK {
			t.Fatalf("human resources session returned %d", sessionRec.Code)
		}
		hrSession := decodeJSON[devSessionResponse](t, sessionRec)
		if slices.Contains(hrSession.AllowedRoutes, "/onboarding") {
			t.Fatalf("human resources allowed routes still include onboarding: %#v", hrSession.AllowedRoutes)
		}

		forcedReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		forcedReq.AddCookie(hrCookie)
		forcedRec := httptest.NewRecorder()
		handler.ServeHTTP(forcedRec, forcedReq)
		if forcedRec.Code != http.StatusForbidden {
			t.Fatalf("forced manual onboarding API returned %d, want 403: %s", forcedRec.Code, forcedRec.Body.String())
		}

		payload = updateTarget("persona", "human_resources", true)
		onboardingFlag = payload
		humanResourcesIndex = slices.IndexFunc(onboardingFlag.PersonaTargets, func(target struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		}) bool {
			return target.ID == "human_resources"
		})
		districtOfficeIndex = slices.IndexFunc(onboardingFlag.SiteTargets, func(target struct {
			ID       string `json:"id"`
			Label    string `json:"label"`
			Enabled  bool   `json:"enabled"`
			ReadOnly bool   `json:"read_only"`
		}) bool {
			return target.ID == "district-office"
		})
		if humanResourcesIndex < 0 || !onboardingFlag.PersonaTargets[humanResourcesIndex].Enabled {
			t.Fatalf("human resources target = %#v, want independently re-enabled", onboardingFlag.PersonaTargets)
		}
		if districtOfficeIndex < 0 || !onboardingFlag.SiteTargets[districtOfficeIndex].Enabled {
			t.Fatalf("district office target changed during persona re-enable: %#v", onboardingFlag.SiteTargets)
		}
	})

	t.Run("feature flag isolation helper resets leaked state between subtests", func(t *testing.T) {
		t.Run("mutates without local restore", func(t *testing.T) {
			isolateDevFeatureFlagState(t, handler)

			itCookie := loginAsPersona(t, handler, "it_admin")
			payload := updateDevFeatureFlagTargetForTest(t, handler, itCookie, "onboarding", "persona", "site_admin", false)
			siteAdminIndex := slices.IndexFunc(payload.PersonaTargets, func(target struct {
				ID       string `json:"id"`
				Label    string `json:"label"`
				Enabled  bool   `json:"enabled"`
				ReadOnly bool   `json:"read_only"`
			}) bool {
				return target.ID == "site_admin"
			})
			if siteAdminIndex < 0 || payload.PersonaTargets[siteAdminIndex].Enabled {
				t.Fatalf("site admin target = %#v, want disabled before cleanup reset", payload.PersonaTargets)
			}
		})

		t.Run("starts from default state", func(t *testing.T) {
			isolateDevFeatureFlagState(t, handler)
		})
	})

	t.Run("feature flag updates reject duplicate targets and malformed payloads", func(t *testing.T) {
		itCookie := loginAsPersona(t, handler, "it_admin")
		cases := []struct {
			name       string
			body       string
			wantStatus int
		}{
			{
				name:       "duplicate matching target",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false},{"target_type":"persona","target_id":"site_admin","enabled":false}]}`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "duplicate conflicting target",
				body:       `{"targets":[{"target_type":"site","target_id":"district-office","enabled":false},{"target_type":"site","target_id":"district-office","enabled":true}]}`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "unknown request field",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false}],"updated_by":"surprise"}`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "unknown target field",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false,"reason":"surprise"}]}`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "malformed json",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false}]`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "multiple json objects",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false}]}{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":true}]}`,
				wantStatus: http.StatusBadRequest,
			},
			{
				name:       "payload too large",
				body:       `{"targets":[{"target_type":"persona","target_id":"site_admin","enabled":false}],"padding":"` + strings.Repeat("x", 20*1024) + `"}`,
				wantStatus: http.StatusRequestEntityTooLarge,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/feature-flags/onboarding", strings.NewReader(tc.body))
				req.AddCookie(itCookie)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != tc.wantStatus {
					t.Fatalf("feature flag update returned %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
				}
				payload := decodeJSON[errorResponse](t, rec)
				if payload.Code == "" {
					t.Fatalf("expected error response code, got %#v", payload)
				}
			})
		}
	})

	t.Run("unauthenticated data quality is 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/data-quality", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("data quality returned %d, want 401", rec.Code)
		}

		payload := decodeJSON[errorResponse](t, rec)
		if payload.Code != "not_authorized" {
			t.Fatalf("error code = %q, want not_authorized", payload.Code)
		}
	})

	t.Run("non it admin personas cannot directly load data quality", func(t *testing.T) {
		for _, personaID := range []string{"human_resources", "site_admin", "site_secretary"} {
			t.Run(personaID, func(t *testing.T) {
				cookie := loginAsPersona(t, handler, personaID)

				req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/data-quality", nil)
				req.AddCookie(cookie)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("data quality returned %d, want 403", rec.Code)
				}

				payload := decodeJSON[errorResponse](t, rec)
				if payload.Code != "forbidden" {
					t.Fatalf("error code = %q, want forbidden", payload.Code)
				}
			})
		}
	})

	t.Run("phone directory person search shows only people and common area rows", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "site_admin")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-person", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phone directory search returned %d", rec.Code)
		}

		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if payload.PageID != "phone-directory-by-person" {
			t.Fatalf("page id = %q, want phone-directory-by-person", payload.PageID)
		}
		if payload.Page.Mode != "person" {
			t.Fatalf("mode = %q, want person", payload.Page.Mode)
		}
		if payload.Page.Query != "" {
			t.Fatalf("query = %q, want empty query", payload.Page.Query)
		}
		if payload.Page.CurrentSiteID != "clover-hs" {
			t.Fatalf("current site id = %q, want clover-hs", payload.Page.CurrentSiteID)
		}
		if payload.Page.CurrentSiteName != "Clover High School" {
			t.Fatalf("current site name = %q, want Clover High School", payload.Page.CurrentSiteName)
		}
		if payload.Page.DirectoryScopeID != "clover-hs" {
			t.Fatalf("directory scope id = %q, want clover-hs", payload.Page.DirectoryScopeID)
		}
		if len(payload.Page.DirectoryScopeOptions) < 2 || payload.Page.DirectoryScopeOptions[0].ID != "district-wide" {
			t.Fatalf("directory scope options = %#v, want district-wide first", payload.Page.DirectoryScopeOptions)
		}
		if payload.Page.SelectedResult != nil {
			t.Fatalf("selected result = %#v, want nil on initial load", payload.Page.SelectedResult)
		}
		if len(payload.Page.Results) < 4 {
			t.Fatalf("expected at least 4 person-mode results, got %d", len(payload.Page.Results))
		}

		hasCommonArea := false
		for _, result := range payload.Page.Results {
			switch result.Type {
			case "person":
			case "common_area":
				hasCommonArea = hasCommonArea || result.Type == "common_area"
			default:
				t.Fatalf("person mode returned disallowed result type %q for %#v", result.Type, result)
			}
		}
		if !hasCommonArea {
			t.Fatal("expected at least one common area result in person mode")
		}
		hasOutOfSiteResult := false
		for _, result := range payload.Page.Results {
			if result.SiteID != "clover-hs" {
				hasOutOfSiteResult = true
				break
			}
		}
		if !hasOutOfSiteResult {
			t.Fatal("expected person mode to include out-of-site district results")
		}
		if payload.Page.Results[0].SiteID != "clover-hs" || payload.Page.Results[0].Type != "person" {
			t.Fatalf("unexpected first person result: %#v", payload.Page.Results[0])
		}
		if payload.Page.Results[1].SiteID != "clover-hs" || payload.Page.Results[1].Type != "person" {
			t.Fatalf("unexpected second person result: %#v", payload.Page.Results[1])
		}
		if payload.Page.Results[2].SiteID != "clover-hs" || payload.Page.Results[2].Type != "common_area" {
			t.Fatalf("unexpected third person result: %#v", payload.Page.Results[2])
		}
	})

	t.Run("onboarding page exposes manual intake options only to hr and it", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "human_resources")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("onboarding returned %d", rec.Code)
		}

		payload := decodeJSON[onboardingResponse](t, rec)
		if payload.PageID != "onboarding" {
			t.Fatalf("page id = %q, want onboarding", payload.PageID)
		}
		if !payload.Page.CanManageManual {
			t.Fatal("expected HR to manage manual onboarding")
		}
		if !slices.Contains(payload.Form.PreferredDevices, "Mac") || !slices.Contains(payload.Form.PreferredDevices, "Windows") {
			t.Fatalf("preferred devices = %#v, want Mac and Windows", payload.Form.PreferredDevices)
		}
		if !slices.Contains(payload.Form.RequestedAeriesAccess, "Teacher") || !slices.Contains(payload.Form.RequestedAeriesAccess, "Registrar") {
			t.Fatalf("requested Aeries options = %#v", payload.Form.RequestedAeriesAccess)
		}
		if len(payload.Page.Rows) == 0 || payload.Page.Rows[0].DateAdded == "" || payload.Page.Rows[0].DateAddedReason != "First Escape import" {
			t.Fatalf("first onboarding row date added metadata = %#v, want first Escape import", payload.Page.Rows)
		}
		foundAction := false
		for _, row := range payload.Page.Rows {
			for _, step := range row.WorkflowSteps {
				for _, action := range step.Actions {
					if strings.HasPrefix(action.Href, "https://mock.wusd.invalid/") {
						foundAction = true
					}
				}
			}
		}
		if !foundAction {
			t.Fatal("expected at least one onboarding workflow action to expose a deterministic mock link")
		}
		foundLeadTimeReview := false
		for _, row := range payload.Page.Rows {
			if row.Person != "Casey Quickstart" {
				continue
			}
			foundLeadTimeReview = true
			if row.Kind != "manual" || row.ManualDraftID != "manual-draft-lead-time-review" {
				t.Fatalf("lead-time review row = %#v, want deterministic editable manual draft", row)
			}
			dateAdded, err := time.Parse("Jan 2, 2006", row.DateAdded)
			if err != nil {
				t.Fatalf("lead-time review date_added = %q: %v", row.DateAdded, err)
			}
			startDate, err := time.Parse("2006-01-02", row.StartDate)
			if err != nil {
				t.Fatalf("lead-time review start_date = %q: %v", row.StartDate, err)
			}
			if days := int(startDate.Sub(dateAdded).Hours() / 24); days < 0 || days > 3 {
				t.Fatalf("lead-time review start/date-added gap = %d days, want 0..3", days)
			}
			if !row.LeadTimeWarning {
				t.Fatal("expected lead-time review row to request the start-date warning")
			}
		}
		if !foundLeadTimeReview {
			t.Fatal("expected deterministic lead-time review manual draft row")
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		siteReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		siteReq.AddCookie(siteCookie)
		siteRec := httptest.NewRecorder()
		handler.ServeHTTP(siteRec, siteReq)
		if siteRec.Code != http.StatusForbidden {
			t.Fatalf("site admin draft create returned %d, want 403", siteRec.Code)
		}
	})

	t.Run("onboarding scopes site persona rows search and room-only drawer updates", func(t *testing.T) {
		web.ResetDevOnboardingStateForTest()
		t.Cleanup(web.ResetDevOnboardingStateForTest)

		siteCookie := loginAsPersona(t, handler, "site_admin")
		sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		sessionReq.AddCookie(siteCookie)
		sessionRec := httptest.NewRecorder()
		handler.ServeHTTP(sessionRec, sessionReq)
		siteSession := decodeJSON[devSessionResponse](t, sessionRec)
		if len(siteSession.VisibleSites) != 1 || siteSession.VisibleSites[0].ID != "clover-hs" {
			t.Fatalf("site admin visible sites = %#v, want exactly Clover High School", siteSession.VisibleSites)
		}

		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding?site_id=clover-hs", nil)
		pageReq.AddCookie(siteCookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusOK {
			t.Fatalf("site admin onboarding returned %d, want 200: %s", pageRec.Code, pageRec.Body.String())
		}
		pagePayload := decodeJSON[onboardingResponse](t, pageRec)
		if len(pagePayload.Page.Rows) == 0 {
			t.Fatal("expected scoped onboarding rows for Clover High School")
		}
		for _, row := range pagePayload.Page.Rows {
			if row.SiteID != "clover-hs" {
				t.Fatalf("site admin received out-of-scope onboarding row: %#v", row)
			}
			if !row.CanUpdateRoom {
				t.Fatalf("site admin row cannot update room: %#v", row)
			}
		}
		for _, room := range pagePayload.Form.Rooms {
			if room.SiteID != "clover-hs" {
				t.Fatalf("site admin received out-of-scope room option: %#v", room)
			}
		}

		searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/search?q=Nia", nil)
		searchReq.AddCookie(siteCookie)
		searchRec := httptest.NewRecorder()
		handler.ServeHTTP(searchRec, searchReq)
		if searchRec.Code != http.StatusOK {
			t.Fatalf("site admin search returned %d, want 200: %s", searchRec.Code, searchRec.Body.String())
		}
		searchPayload := decodeJSON[globalSearchResponse](t, searchRec)
		for _, group := range searchPayload.Page.Groups {
			for _, result := range group.Results {
				if result.Source == "Onboarding" && strings.Contains(result.Context, "District Office") {
					t.Fatalf("site admin search leaked out-of-scope onboarding result: %#v", result)
				}
			}
		}

		forbiddenBody := strings.NewReader(`{"room_id":"iiq-room-cla-108","first_name":"Edited"}`)
		forbiddenReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/rows/jordan-miles/room", forbiddenBody)
		forbiddenReq.Header.Set("Content-Type", "application/json")
		forbiddenReq.AddCookie(siteCookie)
		forbiddenRec := httptest.NewRecorder()
		handler.ServeHTTP(forbiddenRec, forbiddenReq)
		if forbiddenRec.Code != http.StatusForbidden {
			t.Fatalf("site admin non-room update returned %d, want 403: %s", forbiddenRec.Code, forbiddenRec.Body.String())
		}

		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/rows/jordan-miles/room", strings.NewReader(`{"room_id":"iiq-room-cla-108"}`))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(siteCookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("site admin room update returned %d, want 200: %s", updateRec.Code, updateRec.Body.String())
		}
		updatePayload := decodeJSON[onboardingRoomUpdateResponse](t, updateRec)
		if updatePayload.Row.RoomID != "iiq-room-cla-108" || updatePayload.Row.RoomName != "CLA Room 108" {
			t.Fatalf("updated room = %#v, want CLA Room 108", updatePayload.Row)
		}
		for _, row := range updatePayload.Rows {
			if row.SiteID != "clover-hs" {
				t.Fatalf("room update response leaked out-of-scope row: %#v", row)
			}
		}

		secretaryCookie := loginAsPersona(t, handler, "site_secretary")
		secretaryReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/rows/jordan-miles/room", strings.NewReader(`{"room_id":"iiq-room-cla-101"}`))
		secretaryReq.Header.Set("Content-Type", "application/json")
		secretaryReq.AddCookie(secretaryCookie)
		secretaryRec := httptest.NewRecorder()
		handler.ServeHTTP(secretaryRec, secretaryReq)
		if secretaryRec.Code != http.StatusOK {
			t.Fatalf("site secretary room update returned %d, want 200: %s", secretaryRec.Code, secretaryRec.Body.String())
		}
	})

	t.Run("offboarding page enforces auth and role scoped employee id visibility", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/offboarding", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated offboarding returned %d, want 401", rec.Code)
		}

		secretaryCookie := loginAsPersona(t, handler, "site_secretary")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/offboarding", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site secretary offboarding returned %d, want 403", rec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/offboarding", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it offboarding returned %d, want 200", rec.Code)
		}
		itPayload := decodeJSON[offboardingResponse](t, rec)
		if !itPayload.Page.CanManageEndDates || !itPayload.Page.CanManageManual || !itPayload.Page.ShowEmployeeIDs {
			t.Fatalf("it flags = manage:%v manual:%v ids:%v, want true/true/true", itPayload.Page.CanManageEndDates, itPayload.Page.CanManageManual, itPayload.Page.ShowEmployeeIDs)
		}
		if len(itPayload.Page.Rows) < 5 {
			t.Fatalf("it rows = %d, want seeded escape and orphan rows", len(itPayload.Page.Rows))
		}
		if itPayload.Page.Rows[0].EmployeeID == "" {
			t.Fatalf("it row employee id missing: %#v", itPayload.Page.Rows[0])
		}
		foundOrphanAction := false
		foundNamedLicenseReclaim := false
		for _, row := range itPayload.Page.Rows {
			if row.Status == "Security risk" || row.ID == "orphan-riley-park" {
				t.Fatalf("offboarding returned security issue row %#v", row)
			}
			if row.Kind == "orphan" && row.Warning != "" && len(row.Actions) > 0 {
				foundOrphanAction = true
			}
			if row.ID == "escape-taylor-singh" && len(row.Actions) > 0 {
				actionText := row.Actions[0].Detail + " " + row.Actions[0].Resolution
				foundNamedLicenseReclaim = strings.Contains(actionText, "Zoom Workplace for Education Enterprise Essentials") &&
					strings.Contains(actionText, "Zoom Phone Basic")
			}
		}
		if !foundOrphanAction {
			t.Fatal("expected orphan row with warning and drawer actions")
		}
		if !foundNamedLicenseReclaim {
			t.Fatal("expected license reclaim drawer action to name the specific Zoom licenses")
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/offboarding", nil)
		req.AddCookie(siteCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("site admin offboarding returned %d, want 200", rec.Code)
		}
		sitePayload := decodeJSON[offboardingResponse](t, rec)
		if sitePayload.Page.CanManageEndDates || sitePayload.Page.CanManageManual || sitePayload.Page.ShowEmployeeIDs {
			t.Fatalf("site flags = manage:%v manual:%v ids:%v, want false/false/false", sitePayload.Page.CanManageEndDates, sitePayload.Page.CanManageManual, sitePayload.Page.ShowEmployeeIDs)
		}
		for _, row := range sitePayload.Page.Rows {
			if row.Status == "Security risk" || row.ID == "orphan-riley-park" {
				t.Fatalf("site offboarding returned security issue row %#v", row)
			}
			if row.EmployeeID != "" {
				t.Fatalf("site admin received employee id in row %#v", row)
			}
			if row.SiteID != "clover-hs" {
				t.Fatalf("site admin received out-of-scope row %#v", row)
			}
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/offboarding", nil)
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("hr offboarding returned %d, want 200", rec.Code)
		}
		hrPayload := decodeJSON[offboardingResponse](t, rec)
		if !hrPayload.Page.CanManageEndDates || !hrPayload.Page.CanManageManual || !hrPayload.Page.ShowEmployeeIDs {
			t.Fatalf("hr flags = manage:%v manual:%v ids:%v, want true/true/true", hrPayload.Page.CanManageEndDates, hrPayload.Page.CanManageManual, hrPayload.Page.ShowEmployeeIDs)
		}
		for _, row := range hrPayload.Page.Rows {
			if row.Status == "Security risk" || row.ID == "orphan-riley-park" {
				t.Fatalf("hr offboarding returned security issue row %#v", row)
			}
		}
	})

	t.Run("security issues report owns moved offboarding security rows", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/reports/security-issues", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated security report returned %d, want 401", rec.Code)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/reports/security-issues", nil)
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("hr security report returned %d, want 403", rec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/reports/security-issues", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it security report returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSON[offboardingResponse](t, rec)
		if payload.PageID != "reports-security-issues" {
			t.Fatalf("page id = %q, want reports-security-issues", payload.PageID)
		}
		if len(payload.Page.Rows) == 0 {
			t.Fatal("expected migrated security issue rows")
		}
		foundRiley := false
		for _, row := range payload.Page.Rows {
			if row.Status != "Security risk" {
				t.Fatalf("security report returned non-security row %#v", row)
			}
			if row.EndDateEditable {
				t.Fatalf("security report row is editable: %#v", row)
			}
			if row.ID == "orphan-riley-park" && row.Warning != "" && len(row.Actions) > 0 {
				foundRiley = true
			}
		}
		if !foundRiley {
			t.Fatalf("expected Riley Park security issue row with warning/actions, got %#v", payload.Page.Rows)
		}
	})

	t.Run("offboarding end date updates are limited to non escape rows", func(t *testing.T) {
		itCookie := loginAsPersona(t, handler, "it_admin")

		updateBody, err := json.Marshal(map[string]string{"end_date": "2026-07-15"})
		if err != nil {
			t.Fatalf("marshal update: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/offboarding/records/orphan-avery-cole/end-date", bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("orphan end date update returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		updated := decodeJSON[struct {
			Row struct {
				EndDate       string `json:"end_date"`
				EndDateSource string `json:"end_date_source"`
			} `json:"row"`
		}](t, rec)
		if updated.Row.EndDate != "2026-07-15" || updated.Row.EndDateSource != "Local override" {
			t.Fatalf("updated row = %#v, want local override date", updated.Row)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		hrBody, err := json.Marshal(map[string]string{"end_date": "2026-08-01"})
		if err != nil {
			t.Fatalf("marshal hr update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/offboarding/records/orphan-riley-park/end-date", bytes.NewReader(hrBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("hr security issue end date update returned %d, want 404: %s", rec.Code, rec.Body.String())
		}

		badBody, err := json.Marshal(map[string]string{"end_date": "07/15/2026"})
		if err != nil {
			t.Fatalf("marshal invalid update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/offboarding/records/orphan-avery-cole/end-date", bytes.NewReader(badBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid date returned %d, want 400", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/offboarding/records/escape-chris-morgan/end-date", bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("escape date update returned %d, want 409", rec.Code)
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/offboarding/records/orphan-riley-park/end-date", bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(siteCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site admin update returned %d, want 403", rec.Code)
		}
	})

	t.Run("manual offboarding actions are limited to HR and IT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/offboarding/candidates?mode=emergency", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated candidate search returned %d, want 401", rec.Code)
		}

		siteCookie := loginAsPersona(t, handler, "site_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/offboarding/candidates?mode=emergency", nil)
		req.AddCookie(siteCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site admin candidate search returned %d, want 403", rec.Code)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/offboarding/candidates?mode=contractor", nil)
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("hr contractor candidate search returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		candidates := decodeJSON[offboardingCandidatesResponse](t, rec)
		if len(candidates.Candidates) == 0 {
			t.Fatal("expected contractor candidates")
		}
		for _, candidate := range candidates.Candidates {
			if candidate.Kind != "contractor" {
				t.Fatalf("contractor search returned non-contractor candidate %#v", candidate)
			}
			if candidate.EmployeeID == "" {
				t.Fatalf("contractor candidate omitted employee id %#v", candidate)
			}
		}

		siteBody := bytes.NewBufferString(`{"person_id":"employee-chris-morgan"}`)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/offboarding/emergency-deprovision", siteBody)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(siteCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site admin emergency schedule returned %d, want 403", rec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/offboarding/emergency-deprovision", bytes.NewBufferString(`{"person_id":"employee-chris-morgan"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it emergency schedule returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		emergency := decodeJSON[offboardingScheduleResponse](t, rec)
		if emergency.Action.Kind != "emergency_deprovision" || emergency.Action.ScheduledFor != "immediate" || emergency.Action.Mode != "dev_mock_only" {
			t.Fatalf("emergency action = %#v, want immediate DEV mock action", emergency.Action)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/offboarding/contractor-offboarding", bytes.NewBufferString(`{"person_id":"employee-chris-morgan","end_date":"2026-07-15"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("employee contractor schedule returned %d, want 404", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/offboarding/contractor-offboarding", bytes.NewBufferString(`{"person_id":"contractor-sam-ortega","end_date":"07/15/2026"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("bad contractor date returned %d, want 400", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/offboarding/contractor-offboarding", bytes.NewBufferString(`{"person_id":"contractor-sam-ortega","end_date":"2026-07-15"}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("hr contractor schedule returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		contractor := decodeJSON[offboardingScheduleResponse](t, rec)
		if contractor.Action.Kind != "contractor_scheduled_deprovision" || contractor.Action.ScheduledFor != "2026-07-15" || contractor.Action.ActorID != "human_resources" {
			t.Fatalf("contractor action = %#v, want scheduled HR contractor action", contractor.Action)
		}
	})

	t.Run("departing seniors page is scoped to it and device wranglers", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated departing seniors returned %d, want 401", rec.Code)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors", nil)
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("hr departing seniors returned %d, want 403", rec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		sessionPayload := decodeJSON[devSessionResponse](t, rec)
		if !slices.Contains(sessionPayload.AllowedRoutes, "/departing-seniors") {
			t.Fatalf("it allowed routes missing departing seniors: %#v", sessionPayload.AllowedRoutes)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		itPayload := decodeJSON[departingSeniorsResponse](t, rec)
		if !itPayload.Page.CanManage {
			t.Fatal("it should be allowed to manage departing seniors")
		}
		if len(itPayload.Page.Rows) == 0 {
			t.Fatal("expected current senior rows")
		}
		if itPayload.Page.SchoolYear == "" || len(itPayload.Page.SchoolYearOptions) != 5 {
			t.Fatalf("school year options = %#v, selected = %q; want current plus four previous years", itPayload.Page.SchoolYearOptions, itPayload.Page.SchoolYear)
		}
		if !itPayload.Page.SchoolYearOptions[0].Current || itPayload.Page.SchoolYearOptions[0].ID != itPayload.Page.SchoolYear {
			t.Fatalf("first school year option should be current selected year: %#v selected %q", itPayload.Page.SchoolYearOptions[0], itPayload.Page.SchoolYear)
		}
		foundDevice := false
		for _, row := range itPayload.Page.Rows {
			if row.GraduationYear != itPayload.Page.GraduationYear {
				t.Fatalf("row graduation year = %s, page year = %s", row.GraduationYear, itPayload.Page.GraduationYear)
			}
			if row.SchoolYear != itPayload.Page.SchoolYear {
				t.Fatalf("row school year = %s, page year = %s", row.SchoolYear, itPayload.Page.SchoolYear)
			}
			if row.Email == "" || row.StudentID == "" {
				t.Fatalf("row missing searchable identity data: %#v", row)
			}
			if len(row.OutstandingDevices) > 0 {
				foundDevice = true
				if row.OutstandingDevices[0].Serial == "" {
					t.Fatalf("device row missing serial: %#v", row.OutstandingDevices[0])
				}
				if row.OutstandingDevices[0].AssetID != "" && (row.OutstandingDevices[0].Domain == "" || !strings.Contains(row.OutstandingDevices[0].AssetURL, "/agent/assets/")) {
					t.Fatalf("device row missing incidentiq link data for real asset id: %#v", row.OutstandingDevices[0])
				}
			}
		}
		if !foundDevice {
			t.Fatal("expected at least one senior with outstanding IncidentIQ device data")
		}
		foundPlainDevice := false
		for _, row := range itPayload.Page.Rows {
			if row.ID == "senior-sam-rivera" {
				if len(row.OutstandingDevices) != 1 || row.OutstandingDevices[0].AssetID != "" || row.OutstandingDevices[0].AssetURL != "" {
					t.Fatalf("plain device row = %#v, want serial without invented IncidentIQ link", row.OutstandingDevices)
				}
				foundPlainDevice = true
			}
			if row.Status == "Ready" {
				t.Fatalf("current senior row %s status = Ready, want senior grace-period or device-return state before cutoff", row.ID)
			}
		}
		if !foundPlainDevice {
			t.Fatal("expected current senior fixture without a real IncidentIQ asset link")
		}

		previousYear := itPayload.Page.SchoolYearOptions[1]
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year="+url.QueryEscape(previousYear.ID), nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("previous-year departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		previousPayload := decodeJSON[departingSeniorsResponse](t, rec)
		if previousPayload.Page.SchoolYear != previousYear.ID || previousPayload.Page.GraduationYear != previousYear.GraduationYear {
			t.Fatalf("previous-year selection = %s/%s, want %s/%s", previousPayload.Page.SchoolYear, previousPayload.Page.GraduationYear, previousYear.ID, previousYear.GraduationYear)
		}
		if len(previousPayload.Page.Rows) == 0 {
			t.Fatal("expected retained previous senior rows")
		}
		rowsByID := map[string]struct {
			EndDateSource string `json:"end_date_source"`
			Status        string `json:"status"`
			Deprovisioned bool   `json:"deprovisioned"`
		}{}
		for _, row := range previousPayload.Page.Rows {
			rowsByID[row.ID] = struct {
				EndDateSource string `json:"end_date_source"`
				Status        string `json:"status"`
				Deprovisioned bool   `json:"deprovisioned"`
			}{EndDateSource: row.EndDateSource, Status: row.Status, Deprovisioned: row.Deprovisioned}
		}
		if got, ok := rowsByID["senior-ava-rodriguez-2025"]; !ok || got.Status != "Device return required" || !got.Deprovisioned {
			t.Fatalf("previous device-return fixture = %#v, ok=%v; want deprovisioned device-return row", got, ok)
		}
		if got, ok := rowsByID["senior-emma-nguyen-2025-override"]; !ok || got.EndDateSource != "Local override" || got.Status != "Access retained by local override" || got.Deprovisioned {
			t.Fatalf("future override fixture = %#v, ok=%v; want visible intentionally active local override", got, ok)
		}
		if _, ok := rowsByID["senior-ben-owens-2025-expired-override"]; ok {
			t.Fatal("expired local override without devices should not appear in retained previous-year rows")
		}

		cleanCompletedYear := itPayload.Page.SchoolYearOptions[2]
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year="+url.QueryEscape(cleanCompletedYear.ID), nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("clean previous-year departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		cleanCompletedPayload := decodeJSON[departingSeniorsResponse](t, rec)
		for _, row := range cleanCompletedPayload.Page.Rows {
			if row.ID == "senior-noah-kim-2024" {
				t.Fatal("clean completed Noah Kim fixture should be hidden from retained-year rows")
			}
		}

		expiredYear := "2020-2021"
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year="+url.QueryEscape(expiredYear), nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expired-year departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		expiredPayload := decodeJSON[departingSeniorsResponse](t, rec)
		if expiredPayload.Page.SchoolYear == expiredYear || expiredPayload.Page.SchoolYear != itPayload.Page.SchoolYear {
			t.Fatalf("expired school year selected %q, want fallback current %q", expiredPayload.Page.SchoolYear, itPayload.Page.SchoolYear)
		}

		deviceCookie := loginAsPersona(t, handler, "device_wrangler")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
		req.AddCookie(deviceCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		deviceSession := decodeJSON[devSessionResponse](t, rec)
		if !slices.Contains(deviceSession.AllowedRoutes, "/departing-seniors") {
			t.Fatalf("device wrangler allowed routes missing departing seniors: %#v", deviceSession.AllowedRoutes)
		}
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors", nil)
		req.AddCookie(deviceCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("device wrangler departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("departing seniors updates end dates and removes rows only after deprovision without devices", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		itCookie := loginAsPersona(t, handler, "it_admin")

		updateBody, err := json.Marshal(map[string]string{"end_date": "2026-08-31"})
		if err != nil {
			t.Fatalf("marshal departing seniors update: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/end-date", bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("departing senior end date returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		updated := decodeJSON[struct {
			Row struct {
				EndDate       string `json:"end_date"`
				EndDateSource string `json:"end_date_source"`
			} `json:"row"`
		}](t, rec)
		if updated.Row.EndDate != "2026-08-31" || updated.Row.EndDateSource != "Local override" {
			t.Fatalf("departing senior update = %#v, want local override", updated.Row)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/deprovision", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("departing senior device deprovision returned %d, want 200", rec.Code)
		}
		deviceResponse := decodeJSON[struct {
			Removed bool `json:"removed"`
			Row     *struct {
				Deprovisioned bool   `json:"deprovisioned"`
				Status        string `json:"status"`
			} `json:"row"`
		}](t, rec)
		if deviceResponse.Removed || deviceResponse.Row == nil || !deviceResponse.Row.Deprovisioned {
			t.Fatalf("device row deprovision response = %#v, want retained deprovisioned row", deviceResponse)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/departing-seniors/records/senior-maya-chen/deprovision", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("departing senior clean deprovision returned %d, want 200", rec.Code)
		}
		cleanResponse := decodeJSON[struct {
			Removed bool `json:"removed"`
		}](t, rec)
		if !cleanResponse.Removed {
			t.Fatal("expected no-device senior to be removed after deprovision")
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		payload := decodeJSON[departingSeniorsResponse](t, rec)
		for _, row := range payload.Page.Rows {
			if row.ID == "senior-maya-chen" {
				t.Fatal("senior with no devices remained after deprovision")
			}
		}

		badBody, err := json.Marshal(map[string]string{"end_date": "08/31/2026"})
		if err != nil {
			t.Fatalf("marshal invalid departing seniors update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/departing-seniors/records/senior-priya-shah/end-date", bytes.NewReader(badBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid departing senior date returned %d, want 400", rec.Code)
		}

		hrCookie := loginAsPersona(t, handler, "human_resources")
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/deprovision", nil)
		req.AddCookie(hrCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("hr departing senior mutation returned %d, want 403", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/deprovision", nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous departing senior mutation returned %d, want 401", rec.Code)
		}
	})

	t.Run("departing seniors retains current rows with expired overrides until cutoff", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		itCookie := loginAsPersona(t, handler, "it_admin")
		updateBody, err := json.Marshal(map[string]string{"end_date": "2026-01-15"})
		if err != nil {
			t.Fatalf("marshal expired current-year override: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/departing-seniors/records/senior-maya-chen/end-date", bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expired current-year override returned %d, want 200: %s", rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year=2025-2026", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("current-year departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSON[departingSeniorsResponse](t, rec)
		found := false
		for _, row := range payload.Page.Rows {
			if row.ID == "senior-maya-chen" {
				found = true
				if row.Deprovisioned || row.Status != "Suppressed by senior exception" {
					t.Fatalf("expired current-year override row = %#v, want active senior exception row", row)
				}
			}
		}
		if !found {
			t.Fatal("expired current-year override hid Maya Chen before cutoff")
		}
	})

	t.Run("departing seniors clears fixture backed local overrides", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		itCookie := loginAsPersona(t, handler, "it_admin")
		clearBody, err := json.Marshal(map[string]string{"end_date": ""})
		if err != nil {
			t.Fatalf("marshal clear fixture override: %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dev/departing-seniors/records/senior-emma-nguyen-2025-override/end-date", bytes.NewReader(clearBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("clear fixture override returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		updated := decodeJSON[struct {
			Row struct {
				EndDateSource string `json:"end_date_source"`
				Deprovisioned bool   `json:"deprovisioned"`
			} `json:"row"`
		}](t, rec)
		if updated.Row.EndDateSource == "Local override" || !updated.Row.Deprovisioned {
			t.Fatalf("cleared fixture override row = %#v, want default-source deprovisioned row", updated.Row)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year=2024-2025", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("previous-year departing seniors after clear returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSON[departingSeniorsResponse](t, rec)
		for _, row := range payload.Page.Rows {
			if row.ID == "senior-emma-nguyen-2025-override" {
				t.Fatal("cleared fixture-backed local override remained visible as retained access")
			}
		}
	})

	t.Run("departing seniors evaluates cutoff in district timezone", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.September, 1, 6, 30, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		itCookie := loginAsPersona(t, handler, "it_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year=2025-2026", nil)
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("district-time cutoff departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSON[departingSeniorsResponse](t, rec)
		found := false
		for _, row := range payload.Page.Rows {
			if row.ID == "senior-maya-chen" {
				found = true
				if row.Deprovisioned || row.Status != "Suppressed by senior exception" {
					t.Fatalf("district-time cutoff row = %#v, want active until local cutoff day ends", row)
				}
			}
		}
		if !found {
			t.Fatal("Maya Chen disappeared while district timezone was still on cutoff day")
		}
	})

	t.Run("departing seniors auto-deprovisions after senior cutoff", func(t *testing.T) {
		web.ResetDevDepartingSeniorsStateForTest()
		t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)
		cleanupClock := web.SetDevDepartingSeniorsClockForTest(time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC))
		t.Cleanup(cleanupClock)

		itCookie := loginAsPersona(t, handler, "it_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/departing-seniors?school_year=2025-2026", nil)
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("post-cutoff departing seniors returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		payload := decodeJSON[departingSeniorsResponse](t, rec)
		for _, row := range payload.Page.Rows {
			if row.ID == "senior-maya-chen" || row.ID == "senior-jordan-miles" {
				t.Fatalf("post-cutoff clean senior %s should be hidden after auto-deprovision", row.ID)
			}
			if len(row.OutstandingDevices) > 0 && (!row.Deprovisioned || row.Status != "Device return required") {
				t.Fatalf("post-cutoff device row = %#v, want deprovisioned device-return work", row)
			}
		}
	})

	t.Run("manual onboarding draft validates sanitizes and finalizes into mock row", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		createReq.AddCookie(cookie)
		createRec := httptest.NewRecorder()
		handler.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("draft create returned %d, want 201", createRec.Code)
		}
		created := decodeJSON[onboardingDraftResponse](t, createRec)
		if created.Draft.ID == "" {
			t.Fatal("expected draft id")
		}

		invalidBody, err := json.Marshal(map[string]string{
			"start_date":              "2026-05-10",
			"ssn_last4":               "12ab",
			"employee_type":           "Contractor",
			"classification":          "Certificated",
			"first_name":              "  Quincy  ",
			"last_name":               "  Zephyr  ",
			"job_title":               "Counselor",
			"site_id":                 "district-office",
			"personal_email":          "not-an-email",
			"personal_phone":          "555",
			"preferred_device":        "Mac",
			"requested_aeries_access": "Teacher",
		})
		if err != nil {
			t.Fatalf("marshal invalid draft: %v", err)
		}
		invalidReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(invalidBody))
		invalidReq.Header.Set("Content-Type", "application/json")
		invalidReq.AddCookie(cookie)
		invalidRec := httptest.NewRecorder()
		handler.ServeHTTP(invalidRec, invalidReq)
		if invalidRec.Code != http.StatusBadRequest {
			t.Fatalf("invalid draft update returned %d, want 400", invalidRec.Code)
		}

		malformedPhoneBody, err := json.Marshal(map[string]string{
			"start_date":              "2026-05-10",
			"ssn_last4":               "1234",
			"employee_type":           "Contractor",
			"classification":          "Certificated",
			"first_name":              "Quincy",
			"last_name":               "Zephyr",
			"job_title":               "Counselor",
			"site_id":                 "district-office",
			"personal_email":          "quincy.zephyr@example.com",
			"personal_phone":          "707-555-0134",
			"preferred_device":        "Mac",
			"requested_aeries_access": "Teacher",
		})
		if err != nil {
			t.Fatalf("marshal malformed phone draft: %v", err)
		}
		malformedPhoneReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(malformedPhoneBody))
		malformedPhoneReq.Header.Set("Content-Type", "application/json")
		malformedPhoneReq.AddCookie(cookie)
		malformedPhoneRec := httptest.NewRecorder()
		handler.ServeHTTP(malformedPhoneRec, malformedPhoneReq)
		if malformedPhoneRec.Code != http.StatusBadRequest {
			t.Fatalf("malformed phone draft update returned %d, want 400", malformedPhoneRec.Code)
		}

		validBody, err := json.Marshal(map[string]string{
			"start_date":              "2026-05-10",
			"ssn_last4":               "1234",
			"employee_type":           "Contractor",
			"classification":          "Certificated",
			"first_name":              "  Quincy  ",
			"last_name":               "  Zephyr  ",
			"job_title":               "Counselor",
			"site_id":                 "district-office",
			"personal_email":          "  Quincy.Zephyr@Example.COM  ",
			"personal_phone":          " (707) 555-0134 ",
			"preferred_device":        "Mac",
			"requested_aeries_access": "Teacher",
			"notes":                   "  Needs   account  ",
		})
		if err != nil {
			t.Fatalf("marshal valid draft: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(validBody))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("valid draft update returned %d, want 200", updateRec.Code)
		}
		updated := decodeJSON[onboardingDraftResponse](t, updateRec)
		if updated.Draft.FirstName != "Quincy" || updated.Draft.LastName != "Zephyr" {
			t.Fatalf("expected names to be sanitized, got %#v", updated.Draft)
		}
		if updated.Draft.PersonalEmail != "quincy.zephyr@example.com" {
			t.Fatalf("personal email = %q, want lowercase sanitized email", updated.Draft.PersonalEmail)
		}
		if updated.Draft.PersonalPhone != "7075550134" {
			t.Fatalf("personal phone = %q, want canonical 10-digit phone", updated.Draft.PersonalPhone)
		}
		if len(updated.Draft.MissingFields) != 0 {
			t.Fatalf("missing fields = %#v, want none", updated.Draft.MissingFields)
		}
		if updated.Draft.GeneratedEmail != "qzephyr@wusd.org" {
			t.Fatalf("generated email = %q, want qzephyr@wusd.org", updated.Draft.GeneratedEmail)
		}
		if updated.Draft.ValidityState != "valid" {
			t.Fatalf("validity state = %q, want valid", updated.Draft.ValidityState)
		}

		finalizeReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID+"/finalize", nil)
		finalizeReq.AddCookie(cookie)
		finalizeRec := httptest.NewRecorder()
		handler.ServeHTTP(finalizeRec, finalizeReq)
		if finalizeRec.Code != http.StatusOK {
			t.Fatalf("finalize returned %d, want 200", finalizeRec.Code)
		}
		finalized := decodeJSON[onboardingDraftResponse](t, finalizeRec)
		if finalized.Draft.Status != "Ready to Provision" {
			t.Fatalf("status = %q, want Ready to Provision", finalized.Draft.Status)
		}
		if !strings.HasPrefix(finalized.Draft.GeneratedEmployeeID, "66") || len(finalized.Draft.GeneratedEmployeeID) != 7 {
			t.Fatalf("generated employee id = %q, want contractor-style 66xxxxx id", finalized.Draft.GeneratedEmployeeID)
		}
		if len(finalized.Rows) == 0 {
			t.Fatal("expected finalized response rows")
		}
		var manualRowFound bool
		for _, row := range finalized.Rows {
			if row.Kind != "manual" {
				continue
			}
			manualRowFound = true
			if row.DateAdded == "" || row.DateAddedReason != "Manual Non-Escape record added" {
				t.Fatalf("manual row date added metadata = %#v, want manual creation reason", row)
			}
			if len(row.WorkflowSteps) == 0 {
				t.Fatalf("manual row workflow steps = %#v, want at least one step", row.WorkflowSteps)
			}
		}
		if !manualRowFound {
			t.Fatal("expected finalized response to include manual row")
		}
	})

	t.Run("inactive escape contractor reactivation reuses existing identity", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "human_resources")

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		createReq.AddCookie(cookie)
		createRec := httptest.NewRecorder()
		handler.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("draft create returned %d, want 201", createRec.Code)
		}
		created := decodeJSON[onboardingDraftResponse](t, createRec)

		body, err := json.Marshal(map[string]string{
			"start_date":              "2026-05-11",
			"ssn_last4":               "5678",
			"employee_type":           "Contractor",
			"classification":          "Contractor",
			"first_name":              "Harper",
			"last_name":               "Sloan",
			"job_title":               "Counselor",
			"site_id":                 "business-office",
			"personal_email":          "harper.sloan@example.com",
			"personal_phone":          "7075550188",
			"preferred_device":        "Windows",
			"requested_aeries_access": "Staff",
		})
		if err != nil {
			t.Fatalf("marshal reactivation draft: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(body))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("reactivation draft update returned %d, want 200", updateRec.Code)
		}
		updated := decodeJSON[onboardingDraftResponse](t, updateRec)
		if updated.Draft.ChangeReason != "reactivate_non_escape" {
			t.Fatalf("change reason = %q, want reactivate_non_escape", updated.Draft.ChangeReason)
		}
		if updated.Draft.GeneratedEmail != "harper.sloan@wusd.org" {
			t.Fatalf("generated email = %q, want reused Escape email", updated.Draft.GeneratedEmail)
		}
		if updated.Draft.GeneratedEmployeeID != "104812" {
			t.Fatalf("generated employee id = %q, want reused Escape employee number", updated.Draft.GeneratedEmployeeID)
		}
		if updated.Draft.LinkedEscapeRecord == nil || updated.Draft.LinkedEscapeRecord.ID != "escape-harper-sloan" {
			t.Fatalf("linked escape record = %#v, want escape-harper-sloan", updated.Draft.LinkedEscapeRecord)
		}

		finalizeReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID+"/finalize", nil)
		finalizeReq.AddCookie(cookie)
		finalizeRec := httptest.NewRecorder()
		handler.ServeHTTP(finalizeRec, finalizeReq)
		if finalizeRec.Code != http.StatusOK {
			t.Fatalf("reactivation finalize returned %d, want 200", finalizeRec.Code)
		}
		finalized := decodeJSON[onboardingDraftResponse](t, finalizeRec)
		if finalized.Draft.GeneratedEmail != "harper.sloan@wusd.org" || finalized.Draft.GeneratedEmployeeID != "104812" {
			t.Fatalf("finalized draft identity reuse = %#v", finalized.Draft)
		}
	})

	t.Run("active escape contractor collision saves invalid draft and allows soft delete", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		createReq.AddCookie(cookie)
		createRec := httptest.NewRecorder()
		handler.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("draft create returned %d, want 201", createRec.Code)
		}
		created := decodeJSON[onboardingDraftResponse](t, createRec)

		body, err := json.Marshal(map[string]string{
			"start_date":              "2026-05-11",
			"ssn_last4":               "1234",
			"employee_type":           "Contractor",
			"classification":          "Contractor",
			"first_name":              "Nia",
			"last_name":               "Brooks",
			"job_title":               "Counselor",
			"site_id":                 "district-office",
			"personal_email":          "nia.brooks.contractor@example.com",
			"personal_phone":          "7075550199",
			"preferred_device":        "Mac",
			"requested_aeries_access": "Staff",
		})
		if err != nil {
			t.Fatalf("marshal collision draft: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(body))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("collision draft update returned %d, want 200", updateRec.Code)
		}
		updated := decodeJSON[onboardingDraftResponse](t, updateRec)
		if updated.Draft.Status != "Invalid" {
			t.Fatalf("status = %q, want Invalid", updated.Draft.Status)
		}
		if updated.Draft.ValidityState != "invalid" || updated.Draft.InvalidReason != "active_escape_contractor_collision" {
			t.Fatalf("collision validity = %#v", updated.Draft)
		}
		if updated.Draft.ChangeReason != "active_escape_contractor_collision" {
			t.Fatalf("change reason = %q, want active_escape_contractor_collision", updated.Draft.ChangeReason)
		}
		if updated.Draft.LinkedEscapeRecord == nil || updated.Draft.LinkedEscapeRecord.ID != "escape-nia-brooks" {
			t.Fatalf("linked escape record = %#v, want escape-nia-brooks", updated.Draft.LinkedEscapeRecord)
		}
		if !updated.Draft.CanDeleteManualEntry {
			t.Fatal("expected invalid contractor collision draft to be deletable")
		}
		if updated.Draft.GeneratedEmail != "" || updated.Draft.GeneratedEmployeeID != "" {
			t.Fatalf("collision draft should not generate identifiers: %#v", updated.Draft)
		}

		finalizeReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID+"/finalize", nil)
		finalizeReq.AddCookie(cookie)
		finalizeRec := httptest.NewRecorder()
		handler.ServeHTTP(finalizeRec, finalizeReq)
		if finalizeRec.Code != http.StatusConflict {
			t.Fatalf("collision finalize returned %d, want 409", finalizeRec.Code)
		}

		pageReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding", nil)
		pageReq.AddCookie(cookie)
		pageRec := httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		if pageRec.Code != http.StatusOK {
			t.Fatalf("onboarding page returned %d, want 200", pageRec.Code)
		}
		pagePayload := decodeJSON[onboardingResponse](t, pageRec)
		rowFound := false
		for _, row := range pagePayload.Page.Rows {
			if row.ManualDraftID != created.Draft.ID {
				continue
			}
			rowFound = true
			if row.ValidityState != "invalid" || row.InvalidReason != "active_escape_contractor_collision" {
				t.Fatalf("collision row validity = %#v", row)
			}
			if row.LinkedEscapeRecord == nil || row.LinkedEscapeRecord.ID != "escape-nia-brooks" {
				t.Fatalf("collision row linked record = %#v", row.LinkedEscapeRecord)
			}
			if !row.CanDeleteManualEntry {
				t.Fatal("expected collision row to expose delete action")
			}
		}
		if !rowFound {
			t.Fatal("expected invalid collision row on onboarding page")
		}

		deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, nil)
		deleteReq.AddCookie(cookie)
		deleteRec := httptest.NewRecorder()
		handler.ServeHTTP(deleteRec, deleteReq)
		if deleteRec.Code != http.StatusOK {
			t.Fatalf("delete draft returned %d, want 200", deleteRec.Code)
		}

		pageReq = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding", nil)
		pageReq.AddCookie(cookie)
		pageRec = httptest.NewRecorder()
		handler.ServeHTTP(pageRec, pageReq)
		pagePayload = decodeJSON[onboardingResponse](t, pageRec)
		for _, row := range pagePayload.Page.Rows {
			if row.ManualDraftID == created.Draft.ID {
				t.Fatalf("deleted collision row still visible: %#v", row)
			}
		}
	})

	t.Run("past-dated manual entry does not show stale late-start warning", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		createReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		createReq.AddCookie(cookie)
		createRec := httptest.NewRecorder()
		handler.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("draft create returned %d, want 201", createRec.Code)
		}
		created := decodeJSON[onboardingDraftResponse](t, createRec)
		pastDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
		body, err := json.Marshal(map[string]string{
			"start_date":              pastDate,
			"ssn_last4":               "9876",
			"employee_type":           "Contractor",
			"classification":          "Contractor",
			"first_name":              "Ari",
			"last_name":               "Pender",
			"job_title":               "Counselor",
			"site_id":                 "district-office",
			"personal_email":          "ari.pender@example.com",
			"personal_phone":          "7075550166",
			"preferred_device":        "Windows",
			"requested_aeries_access": "Staff",
		})
		if err != nil {
			t.Fatalf("marshal past-date draft: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/onboarding/manual-drafts/"+created.Draft.ID, bytes.NewReader(body))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("past-date update returned %d, want 200", updateRec.Code)
		}
		updated := decodeJSON[onboardingDraftResponse](t, updateRec)
		if updated.Draft.LateStart {
			t.Fatal("expected manual past-date draft to avoid stale late_start")
		}
		if updated.Draft.ScheduledFor != "" {
			t.Fatalf("scheduled_for = %q, want no stale next-cycle schedule", updated.Draft.ScheduledFor)
		}
		if updated.Draft.EffectiveDate != pastDate {
			t.Fatalf("effective date = %q, want %q", updated.Draft.EffectiveDate, pastDate)
		}
	})

	t.Run("escape-backed past-date row preserves source date without stale late-start warning", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "human_resources")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/onboarding", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("onboarding returned %d", rec.Code)
		}
		payload := decodeJSON[onboardingResponse](t, rec)
		for _, row := range payload.Page.Rows {
			if row.Person != "Nia Brooks" {
				continue
			}
			if row.ChangeReason != "reactivate_same_role" {
				t.Fatalf("change reason = %q, want reactivate_same_role", row.ChangeReason)
			}
			if row.LateStart {
				t.Fatal("expected Nia Brooks to avoid stale late_start")
			}
			if row.ScheduledFor != "" {
				t.Fatalf("scheduled_for = %q, want no stale next-cycle schedule", row.ScheduledFor)
			}
			if row.EffectiveDate != row.StartDate {
				t.Fatalf("effective date = %q, want preserved source date %q", row.EffectiveDate, row.StartDate)
			}
			return
		}
		t.Fatal("expected Nia Brooks reactivation row")
	})

	t.Run("manual onboarding generated email falls through collision order", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")
		firstEmail := createAndFinalizeManualOnboarding(t, handler, cookie, "Maren", "Lumen")
		secondEmail := createAndFinalizeManualOnboarding(t, handler, cookie, "Maren", "Lumen")
		if firstEmail != "mlumen@wusd.org" {
			t.Fatalf("first generated email = %q, want mlumen@wusd.org", firstEmail)
		}
		if secondEmail != "maren.lumen@wusd.org" {
			t.Fatalf("second generated email = %q, want maren.lumen@wusd.org", secondEmail)
		}
	})

	t.Run("phone directory room search shows only common area and classroom shared line rows", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "site_secretary")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-room", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phone directory room search returned %d", rec.Code)
		}

		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if payload.PageID != "phone-directory-by-room" {
			t.Fatalf("page id = %q, want phone-directory-by-room", payload.PageID)
		}
		if payload.Page.Mode != "room" {
			t.Fatalf("mode = %q, want room", payload.Page.Mode)
		}
		if payload.Page.SelectedResult != nil {
			t.Fatalf("selected result = %#v, want nil on initial load", payload.Page.SelectedResult)
		}
		if len(payload.Page.Results) < 4 {
			t.Fatalf("expected at least 4 room-mode results, got %d", len(payload.Page.Results))
		}
		if payload.Page.DirectoryScopeID != "clover-hs" {
			t.Fatalf("directory scope id = %q, want clover-hs", payload.Page.DirectoryScopeID)
		}

		hasClassroomSharedLine := false
		hasOutOfSiteResult := false
		for _, result := range payload.Page.Results {
			switch result.Type {
			case "common_area":
			case "classroom_slg":
				hasClassroomSharedLine = hasClassroomSharedLine || result.Type == "classroom_slg"
			default:
				t.Fatalf("room mode returned disallowed result type %q for %#v", result.Type, result)
			}
			if result.SiteID != "clover-hs" {
				hasOutOfSiteResult = true
			}
		}
		if !hasClassroomSharedLine {
			t.Fatal("expected at least one classroom shared line result in room mode")
		}
		if !hasOutOfSiteResult {
			t.Fatal("expected room mode to include out-of-site district results")
		}
		if payload.Page.Results[0].Type != "common_area" || payload.Page.Results[0].SiteID != "clover-hs" {
			t.Fatalf("unexpected first room result: %#v", payload.Page.Results[0])
		}
	})

	t.Run("phone directory department search shows only department shared lines and call queues", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "human_resources")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-department", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phone directory department search returned %d", rec.Code)
		}

		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if payload.PageID != "phone-directory-by-department" {
			t.Fatalf("page id = %q, want phone-directory-by-department", payload.PageID)
		}
		if payload.Page.Mode != "department" {
			t.Fatalf("mode = %q, want department", payload.Page.Mode)
		}
		if payload.Page.DirectoryScopeID != "district-wide" {
			t.Fatalf("directory scope id = %q, want district-wide", payload.Page.DirectoryScopeID)
		}
		if payload.Page.SelectedResult != nil {
			t.Fatalf("selected result = %#v, want nil on initial load", payload.Page.SelectedResult)
		}
		if len(payload.Page.Results) < 4 {
			t.Fatalf("expected at least 4 department-mode results, got %d", len(payload.Page.Results))
		}

		hasCallQueue := false
		for _, result := range payload.Page.Results {
			switch result.Type {
			case "department_slg":
				if result.TypeLabel == "Department / Shared Line" {
					t.Fatalf("department mode returned deprecated generic label for %#v", result)
				}
			case "call_queue":
				hasCallQueue = true
				if result.TypeLabel != "Call Queue" {
					t.Fatalf("call queue type label = %q, want Call Queue", result.TypeLabel)
				}
			default:
				t.Fatalf("department mode returned disallowed result type %q for %#v", result.Type, result)
			}
		}
		if !hasCallQueue {
			t.Fatal("expected at least one call queue result in department mode")
		}
		if payload.Page.Results[0].Type != "department_slg" || payload.Page.Results[0].SiteID != "district-office" {
			t.Fatalf("unexpected first department result: %#v", payload.Page.Results[0])
		}
	})

	t.Run("phone directory site id boosts the focused site without excluding district rows", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-person?site_id=clover-hs", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phone directory focused site request returned %d", rec.Code)
		}

		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if payload.Page.DirectoryScopeID != "clover-hs" {
			t.Fatalf("directory scope id = %q, want clover-hs", payload.Page.DirectoryScopeID)
		}
		if len(payload.Page.Results) < 4 {
			t.Fatalf("expected district-wide focused results, got %d", len(payload.Page.Results))
		}
		if payload.Page.Results[0].SiteID != "clover-hs" {
			t.Fatalf("first focused result site = %q, want clover-hs", payload.Page.Results[0].SiteID)
		}

		hasOutOfSiteResult := false
		for _, result := range payload.Page.Results {
			if result.SiteID != "clover-hs" {
				hasOutOfSiteResult = true
				break
			}
		}
		if !hasOutOfSiteResult {
			t.Fatal("expected focused directory results to include non-focused district sites")
		}
	})

	t.Run("phone directory invalid site id falls back to persona default directory scope", func(t *testing.T) {
		itCookie := loginAsPersona(t, handler, "it_admin")
		itReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-person?site_id=unknown-site", nil)
		itReq.AddCookie(itCookie)
		itRec := httptest.NewRecorder()
		handler.ServeHTTP(itRec, itReq)
		if itRec.Code != http.StatusOK {
			t.Fatalf("it admin invalid site request returned %d", itRec.Code)
		}
		itPayload := decodeJSON[phoneDirectoryResponse](t, itRec)
		if itPayload.Page.DirectoryScopeID != "district-wide" {
			t.Fatalf("it admin invalid scope fallback = %q, want district-wide", itPayload.Page.DirectoryScopeID)
		}

		secretaryCookie := loginAsPersona(t, handler, "site_secretary")
		secretaryReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-room?site_id=unknown-site", nil)
		secretaryReq.AddCookie(secretaryCookie)
		secretaryRec := httptest.NewRecorder()
		handler.ServeHTTP(secretaryRec, secretaryReq)
		if secretaryRec.Code != http.StatusOK {
			t.Fatalf("site secretary invalid site request returned %d", secretaryRec.Code)
		}
		secretaryPayload := decodeJSON[phoneDirectoryResponse](t, secretaryRec)
		if secretaryPayload.Page.DirectoryScopeID != "clover-hs" {
			t.Fatalf("site secretary invalid scope fallback = %q, want clover-hs", secretaryPayload.Page.DirectoryScopeID)
		}
	})

	t.Run("faculty staff phone directory returns all district person rows with home site first", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "faculty_staff")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-person", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("faculty phone directory request returned %d", rec.Code)
		}

		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if payload.Page.DirectoryScopeID != "clover-hs" {
			t.Fatalf("directory scope id = %q, want clover-hs", payload.Page.DirectoryScopeID)
		}
		if len(payload.Page.Results) < 4 {
			t.Fatalf("expected faculty district-wide results, got %d", len(payload.Page.Results))
		}
		if payload.Page.Results[0].SiteID != "clover-hs" {
			t.Fatalf("first faculty result site = %q, want clover-hs", payload.Page.Results[0].SiteID)
		}

		hasOutOfSiteResult := false
		for _, result := range payload.Page.Results {
			if result.SiteID != "clover-hs" {
				hasOutOfSiteResult = true
				break
			}
		}
		if !hasOutOfSiteResult {
			t.Fatal("expected faculty directory results to include other district sites")
		}
	})

	t.Run("phone directory extensions between four and six digits remain valid with six-digit mocks preferred", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		extensionLengths := map[int]int{}
		for _, path := range []string{
			"/api/v1/dev/pages/phone-directory/by-person",
			"/api/v1/dev/pages/phone-directory/by-room",
			"/api/v1/dev/pages/phone-directory/by-department",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s returned %d, want 200", path, rec.Code)
			}

			payload := decodeJSON[phoneDirectoryResponse](t, rec)
			for _, result := range payload.Page.Results {
				if result.ExtensionLength < 4 || result.ExtensionLength > 6 {
					t.Fatalf("extension length = %d for %#v, want between 4 and 6", result.ExtensionLength, result)
				}
				if !result.ExtensionValid {
					t.Fatalf("expected extension %q to remain valid in DEV data", result.Extension)
				}
				extensionLengths[result.ExtensionLength]++
			}
		}

		if extensionLengths[5] == 0 {
			t.Fatalf("expected at least one representative 5-digit extension, got %#v", extensionLengths)
		}
		if extensionLengths[6] == 0 {
			t.Fatalf("expected at least one 6-digit extension, got %#v", extensionLengths)
		}
		if extensionLengths[6] <= extensionLengths[5] {
			t.Fatalf("expected 6-digit extensions to remain the default majority, got %#v", extensionLengths)
		}
	})

	t.Run("it admin can access all phone directory modes", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")

		for _, path := range []string{
			"/api/v1/dev/pages/phone-directory/by-person?q=alex",
			"/api/v1/dev/pages/phone-directory/by-room?q=350",
			"/api/v1/dev/pages/phone-directory/by-department?q=350",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s returned %d, want 200", path, rec.Code)
			}
		}
	})

	t.Run("global search covers accessible search types", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")
		cases := []struct {
			name       string
			query      string
			groupID    string
			titleMatch string
		}{
			{name: "name", query: "Riley Vale", groupID: "people", titleMatch: "Riley Vale"},
			{name: "email", query: "riley.vale@mock.wusd.invalid", groupID: "people", titleMatch: "Riley Vale"},
			{name: "phone", query: "555-4017", groupID: "people", titleMatch: "Riley Vale"},
			{name: "extension", query: "34017", groupID: "people", titleMatch: "Riley Vale"},
			{name: "employee id", query: "EMP-MOCK-1002", groupID: "people", titleMatch: "Riley Vale"},
			{name: "student id", query: "S-2026-10088", groupID: "departing-seniors", titleMatch: "Luis Alvarez"},
			{name: "asset id", query: "IIQ-109284", groupID: "devices-assets", titleMatch: "IIQ-109284"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/search?q="+url.QueryEscape(tc.query), nil)
				req.AddCookie(cookie)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("global search returned %d, want 200", rec.Code)
				}
				payload := decodeJSON[globalSearchResponse](t, rec)
				if payload.PageID != "global-search" {
					t.Fatalf("page id = %q, want global-search", payload.PageID)
				}
				if !globalSearchHasResult(payload, tc.groupID, tc.titleMatch) {
					t.Fatalf("expected %q in group %q for query %q, got %#v", tc.titleMatch, tc.groupID, tc.query, payload.Page.Groups)
				}
			})
		}
	})

	t.Run("global search does not leak hidden employee id results", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "site_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/search?q=103118", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("global search returned %d, want 200", rec.Code)
		}
		payload := decodeJSON[globalSearchResponse](t, rec)
		if globalSearchHasResult(payload, "offboarding", "Chris Morgan") {
			t.Fatalf("site admin global search leaked an HR/IT-only employee ID match: %#v", payload.Page.Groups)
		}
	})

	t.Run("global search returns an empty payload for no matches", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/search?q=no-such-search-token", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("global search returned %d, want 200", rec.Code)
		}
		payload := decodeJSON[globalSearchResponse](t, rec)
		if len(payload.Page.Groups) != 0 {
			t.Fatalf("expected empty search groups, got %#v", payload.Page.Groups)
		}
	})

	t.Run("phone directory numeric extension search returns the matching row", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "it_admin")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/phone-directory/by-person?q=34017", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phone directory search returned %d, want 200", rec.Code)
		}
		payload := decodeJSON[phoneDirectoryResponse](t, rec)
		if !phoneDirectoryHasTitle(payload, "Riley Vale") {
			t.Fatalf("expected Riley Vale in numeric extension search results, got %#v", payload.Page.Results)
		}
	})

	t.Run("room moves enforce site scoped drafts and room defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated room moves returned %d, want 401", rec.Code)
		}

		secretaryCookie := loginAsPersona(t, handler, "site_secretary")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("site secretary room moves returned %d, want 200", rec.Code)
		}
		sitePayload := decodeJSON[roomMovesResponse](t, rec)
		if sitePayload.Page.CanManageDistrict {
			t.Fatal("site secretary should not receive district room-move controls")
		}
		if sitePayload.Page.ScopeSite.ID != "clover-hs" {
			t.Fatalf("scope site = %q, want clover-hs", sitePayload.Page.ScopeSite.ID)
		}
		for _, person := range sitePayload.Page.People {
			if person.SiteID != "clover-hs" {
				t.Fatalf("site secretary received out-of-site room move person %#v", person)
			}
		}
		foundSLGOnlyFixture := false
		for _, person := range sitePayload.Page.People {
			if person.ID == "casey-nguyen" && person.SourceRole == "slg_only" {
				foundSLGOnlyFixture = true
			}
		}
		if !foundSLGOnlyFixture {
			t.Fatalf("site secretary room move people = %#v, want SLG-only repeated-user fixture", sitePayload.Page.People)
		}
		if len(sitePayload.Page.Rooms) == 0 || sitePayload.Page.Rooms[0].ID != "none" {
			t.Fatalf("rooms = %#v, want None option first", sitePayload.Page.Rooms)
		}
		for _, room := range sitePayload.Page.Rooms {
			if room.SiteID != "clover-hs" {
				t.Fatalf("site secretary received out-of-site room option %#v", room)
			}
		}
		foundITAuthoredAssignedSiteRow := false
		for _, row := range sitePayload.Page.Rows {
			if row.CurrentSiteID != "clover-hs" {
				t.Fatalf("site secretary received out-of-site room move row %#v", row)
			}
			if row.AuthorID == "it_admin" && row.CurrentSiteID == "clover-hs" {
				foundITAuthoredAssignedSiteRow = true
				if row.CanEdit || row.CanCancel {
					t.Fatalf("site secretary IT-authored assigned-site row = %#v, want visible but read-only", row)
				}
			}
		}
		if !foundITAuthoredAssignedSiteRow {
			t.Fatalf("site secretary room moves rows = %#v, want visible IT-authored assigned-site row", sitePayload.Page.Rows)
		}

		siteAdminCookie := loginAsPersona(t, handler, "site_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(siteAdminCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("site admin room moves returned %d, want 200", rec.Code)
		}
		siteAdminPayload := decodeJSON[roomMovesResponse](t, rec)
		siteAdminFoundITAuthoredAssignedSiteRow := false
		for _, row := range siteAdminPayload.Page.Rows {
			if row.CurrentSiteID != "clover-hs" {
				t.Fatalf("site admin received out-of-site room move row %#v", row)
			}
			if row.AuthorID == "it_admin" && row.CurrentSiteID == "clover-hs" {
				siteAdminFoundITAuthoredAssignedSiteRow = true
				if row.CanEdit || row.CanCancel {
					t.Fatalf("site admin IT-authored assigned-site row = %#v, want visible but read-only", row)
				}
			}
		}
		if !siteAdminFoundITAuthoredAssignedSiteRow {
			t.Fatalf("site admin room moves rows = %#v, want visible IT-authored assigned-site row", siteAdminPayload.Page.Rows)
		}

		createBody, err := json.Marshal(map[string]any{
			"mode":      "mid_year_targeted_move",
			"person_id": "morgan-lee",
		})
		if err != nil {
			t.Fatalf("marshal single draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(createBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("site-scoped single draft with default current room returned %d, want 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Morgan Lee is already in B-210") {
			t.Fatalf("site-scoped single draft error = %s, want same-room validation for documented current-room default", rec.Body.String())
		}

		sameRoomBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{{
				"person_id":           "morgan-lee",
				"destination_site_id": "clover-hs",
				"destination_room_id": "cla-b210",
			}},
		})
		if err != nil {
			t.Fatalf("marshal same-room draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(sameRoomBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("same-room draft returned %d, want 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Morgan Lee is already in B-210") {
			t.Fatalf("same-room draft error = %s, want person and current-room validation text", rec.Body.String())
		}

		primaryConflictBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{{
				"person_id":           "morgan-lee",
				"destination_site_id": "clover-hs",
				"destination_room_id": "cla-b204",
			}},
		})
		if err != nil {
			t.Fatalf("marshal primary-conflict draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(primaryConflictBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("primary-conflict room move draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		primaryConflictDraft := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(primaryConflictDraft.Draft.Rows) != 1 {
			t.Fatalf("primary-conflict draft rows = %#v, want one Morgan Lee row", primaryConflictDraft.Draft.Rows)
		}
		conflictRow := primaryConflictDraft.Draft.Rows[0]
		if conflictRow.Phone != "Add to room shared line group; keep primary phone owner" {
			t.Fatalf("primary-conflict phone outcome = %q, want shared-line-group automation", conflictRow.Phone)
		}
		if !strings.Contains(conflictRow.AttentionReason, "Jordan Patel") || !strings.Contains(conflictRow.AutomationOutcome, "shared line group") {
			t.Fatalf("primary-conflict details = %#v, want owner and automation outcome", conflictRow)
		}
		if len(conflictRow.ResolutionSteps) == 0 || len(conflictRow.ExternalSystems) != 0 {
			t.Fatalf("primary-conflict operator guidance = %#v, want resolution steps without automated-path external systems", conflictRow)
		}
		if !primaryConflictDraft.Draft.CanEdit || !primaryConflictDraft.Draft.CanDelete || primaryConflictDraft.Draft.AuthorID != "site_secretary" {
			t.Fatalf("site secretary authored draft permissions = %#v, want editable self-authored draft", primaryConflictDraft.Draft)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+primaryConflictDraft.Draft.ID+"/schedule", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("site secretary self-authored draft schedule returned %d, want 200: %s", rec.Code, rec.Body.String())
		}

		crossSiteBody, err := json.Marshal(map[string]any{
			"mode":      "mid_year_targeted_move",
			"person_id": "taylor-quinn",
		})
		if err != nil {
			t.Fatalf("marshal cross-site draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(crossSiteBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("cross-site site-secretary draft returned %d, want 403", rec.Code)
		}

		readOnlyUpdateBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{{
				"person_id":           "alex-ramirez",
				"destination_site_id": "clover-hs",
				"destination_room_id": "cla-b204",
			}},
		})
		if err != nil {
			t.Fatalf("marshal read-only update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/room-moves/drafts/single-alex-ramirez", bytes.NewReader(readOnlyUpdateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site secretary update of IT-authored assigned-site row returned %d, want 403", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/single-alex-ramirez/cancel", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site secretary cancel of IT-authored assigned-site row returned %d, want 403", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves/bulk-draft?draft_id=rm-draft-103", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("site secretary visible IT-authored bulk draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		readOnlyBulk := decodeJSON[roomMovesBulkDraftResponse](t, rec)
		if readOnlyBulk.Page.Draft.AuthorID != "it_admin" || readOnlyBulk.Page.Draft.CanEdit || readOnlyBulk.Page.Draft.CanDelete {
			t.Fatalf("site secretary IT-authored bulk draft = %#v, want read-only", readOnlyBulk.Page.Draft)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/room-moves/drafts/rm-draft-103", bytes.NewReader(readOnlyUpdateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site secretary save of IT-authored bulk draft returned %d, want 403", rec.Code)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/rm-draft-103/apply", nil)
		req.AddCookie(secretaryCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site secretary apply of IT-authored bulk draft returned %d, want 403", rec.Code)
		}

		itCookie := loginAsPersona(t, handler, "it_admin")
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it room moves returned %d, want 200", rec.Code)
		}
		itRoomMoves := decodeJSON[roomMovesResponse](t, rec)
		noneRoomOptions := 0
		for _, room := range itRoomMoves.Page.Rooms {
			if room.ID == "none" {
				noneRoomOptions++
			}
		}
		if noneRoomOptions != 1 {
			t.Fatalf("it room options have %d None entries, want exactly 1: %#v", noneRoomOptions, itRoomMoves.Page.Rooms)
		}
		foundMorganLee := false
		for _, row := range itRoomMoves.Page.Rows {
			if row.DraftID == "single-morgan-lee" {
				foundMorganLee = true
				if row.State != "Ready" || row.Phone != "Add to room shared line group; keep primary phone owner" {
					t.Fatalf("Morgan Lee room move row = %#v, want ready shared-line-group automation", row)
				}
				if row.Warning == "Primary conflict" || row.Phone == "Manual ticket" {
					t.Fatalf("Morgan Lee room move row = %#v, want no generic conflict/manual-ticket copy", row)
				}
				if !strings.Contains(row.AttentionReason, "Jordan Patel") || !strings.Contains(row.AutomationOutcome, "shared line group") {
					t.Fatalf("Morgan Lee room move details = %#v, want primary owner and automation explanation", row)
				}
				if len(row.ResolutionSteps) == 0 || len(row.ExternalSystems) != 0 {
					t.Fatalf("Morgan Lee guidance = %#v, want resolution steps without automated-path external systems", row)
				}
			}
		}
		if !foundMorganLee {
			t.Fatalf("it room moves rows = %#v, want Morgan Lee seed row", itRoomMoves.Page.Rows)
		}
		foundSeedBulkMove := false
		for _, row := range itRoomMoves.Page.Rows {
			if row.DraftID == "rm-draft-103" {
				foundSeedBulkMove = true
				if row.Person != "Bulk Move" || row.Author == "" || row.State != "Scheduled" || row.ScheduledFor == "" {
					t.Fatalf("seed bulk row = %#v, want Bulk Move with author, Scheduled state, and scheduled timestamp", row)
				}
				if row.Warning == "" {
					t.Fatalf("seed bulk row = %#v, want warning available for drawer/bulk warning surfaces", row)
				}
			}
		}
		if !foundSeedBulkMove {
			t.Fatalf("it room moves rows = %#v, want seeded bulk move row", itRoomMoves.Page.Rows)
		}
		foundJamieReed := false
		for _, row := range itRoomMoves.Page.Rows {
			if row.DraftID == "single-jamie-reed" {
				foundJamieReed = true
				if row.State != "Ready" || row.Phone != "Remove phone and SLGs; convert room to common area" || row.Warning != "" {
					t.Fatalf("Jamie Reed room move row = %#v, want ready null-room automation without review warning", row)
				}
			}
		}
		if !foundJamieReed {
			t.Fatalf("it room moves rows = %#v, want Jamie Reed seed row", itRoomMoves.Page.Rows)
		}
		updateJamieBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{{
				"person_id":           "jamie-reed",
				"destination_site_id": "desert-view",
				"destination_room_id": "dve-c122",
			}},
		})
		if err != nil {
			t.Fatalf("marshal Jamie Reed update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/room-moves/drafts/single-jamie-reed", bytes.NewReader(updateJamieBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Jamie Reed existing-row update returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		updatedJamie := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(updatedJamie.Draft.Rows) != 1 || updatedJamie.Draft.Rows[0].DestinationRoomID != "dve-c122" {
			t.Fatalf("updated Jamie Reed draft = %#v, want C-122 destination on existing draft id", updatedJamie.Draft)
		}
		if updatedJamie.Draft.ScopeSiteID != "desert-view" {
			t.Fatalf("updated Jamie Reed scope = %q, want original desert-view scope", updatedJamie.Draft.ScopeSiteID)
		}
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it room moves after Jamie update returned %d, want 200", rec.Code)
		}
		afterJamieUpdate := decodeJSON[roomMovesResponse](t, rec)
		jamieRows := 0
		for _, row := range afterJamieUpdate.Page.Rows {
			if row.DraftID == "single-jamie-reed" {
				jamieRows++
				if row.DestinationRoomID != "dve-c122" || row.DestinationRoom != "C-122" {
					t.Fatalf("Jamie Reed updated row = %#v, want C-122 destination", row)
				}
			}
		}
		if jamieRows != 1 {
			t.Fatalf("found %d Jamie Reed rows after edit, want one existing row updated", jamieRows)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/single-jamie-reed/apply", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Jamie Reed existing-row apply returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves/bulk-draft?draft_id=rm-draft-103", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it seeded bulk draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		seedBulkDraft := decodeJSON[roomMovesBulkDraftResponse](t, rec)
		if seedBulkDraft.Page.Draft.ID != "rm-draft-103" || len(seedBulkDraft.Page.Draft.Rows) == 0 {
			t.Fatalf("seeded bulk draft = %#v, want rm-draft-103 with visible rows", seedBulkDraft.Page.Draft)
		}
		if len(seedBulkDraft.Page.Draft.Rows) < 2 {
			t.Fatalf("seeded bulk draft rows = %#v, want add and removal fixture rows", seedBulkDraft.Page.Draft.Rows)
		}
		addRow := seedBulkDraft.Page.Draft.Rows[0]
		if addRow.Action != "add" || addRow.CurrentRoomID != "none" || addRow.CurrentRoom != "" {
			t.Fatalf("seeded add row = %#v, want add action with cleared current room", addRow)
		}
		removalRow := seedBulkDraft.Page.Draft.Rows[1]
		if removalRow.Action != "removal" || removalRow.DestinationRoomID != "none" || removalRow.DestinationRoom != "None" {
			t.Fatalf("seeded removal row = %#v, want removal action with destination room None", removalRow)
		}
		expectedRemovalWarning := "Destination room for Morgan Lee is None; phone and room assignments will be removed."
		if removalRow.Warning != expectedRemovalWarning {
			t.Fatalf("seeded removal warning = %q, want person-specific warning", removalRow.Warning)
		}
		if len(seedBulkDraft.Page.Draft.Warnings) == 0 || seedBulkDraft.Page.Draft.Warnings[0] != expectedRemovalWarning {
			t.Fatalf("seeded bulk warnings = %#v, want person-specific warning bullet", seedBulkDraft.Page.Draft.Warnings)
		}
		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it room moves after seeded bulk open returned %d, want 200", rec.Code)
		}
		afterSeedBulkOpen := decodeJSON[roomMovesResponse](t, rec)
		seedBulkRows := 0
		for _, row := range afterSeedBulkOpen.Page.Rows {
			if row.DraftID == "rm-draft-103" {
				seedBulkRows++
				if row.Person != "Bulk Move" || row.State != "Scheduled" || row.ScheduledFor == "" || row.Warning == "" {
					t.Fatalf("seeded bulk row after draft cache = %#v, want scheduled seed status preserved", row)
				}
			}
		}
		if seedBulkRows != 1 {
			t.Fatalf("found %d seeded bulk rows after draft cache, want one preserved review row", seedBulkRows)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/rm-draft-103/cancel", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("cancel seeded bulk draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}

		itBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{
				{
					"person_id":           "morgan-lee",
					"destination_site_id": "desert-view",
					"destination_room_id": "dve-c118",
				},
			},
		})
		if err != nil {
			t.Fatalf("marshal it draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(itBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("it inter-site draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		itCreated := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if !itCreated.Draft.CanManageDistrict {
			t.Fatal("it draft should expose district controls")
		}
		if len(itCreated.Draft.Rows) != 1 || itCreated.Draft.Rows[0].DestinationSiteID != "desert-view" || itCreated.Draft.Rows[0].DestinationRoomID != "none" {
			t.Fatalf("it inter-site row = %#v, want destination site desert-view and room none", itCreated.Draft.Rows)
		}
		foundInterSiteWarning := false
		for _, warning := range itCreated.Draft.Warnings {
			if strings.Contains(warning, "Inter-site move") {
				foundInterSiteWarning = true
			}
		}
		if !foundInterSiteWarning {
			t.Fatalf("it inter-site warnings = %#v, want inter-site warning", itCreated.Draft.Warnings)
		}
	})

	t.Run("room moves bulk drafts support roster and manual list lifecycle", func(t *testing.T) {
		itCookie := loginAsPersona(t, handler, "it_admin")

		createBody, err := json.Marshal(map[string]any{
			"mode":          "end_of_year_site_move",
			"scope_site_id": "clover-hs",
		})
		if err != nil {
			t.Fatalf("marshal roster draft: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(createBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("bulk roster draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		roster := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if roster.Draft.Mode != "end_of_year_site_move" || len(roster.Draft.Rows) < 2 {
			t.Fatalf("roster draft = %#v, want clover roster rows", roster.Draft)
		}
		placeholderRow := roster.Draft.Rows[0]
		if placeholderRow.Action != "change" || placeholderRow.DestinationRoomID != "none" {
			t.Fatalf("roster placeholder row = %#v, want unchanged destination placeholder", placeholderRow)
		}
		if placeholderRow.Phone == "Remove phone and SLGs; convert room to common area" || strings.Contains(placeholderRow.Warning, "will be removed") {
			t.Fatalf("roster placeholder row = %#v, want neutral placeholder without removal outcome", placeholderRow)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+roster.Draft.ID+"/schedule", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("untouched roster schedule returned %d, want 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Choose a destination room") {
			t.Fatalf("untouched roster schedule error = %s, want placeholder validation", rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves/bulk-draft?draft_id="+url.QueryEscape(roster.Draft.ID), nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("bulk draft page returned %d, want 200", rec.Code)
		}
		page := decodeJSON[roomMovesBulkDraftResponse](t, rec)
		if page.Page.Draft.ID != roster.Draft.ID || len(page.Page.Draft.Rows) != len(roster.Draft.Rows) {
			t.Fatalf("bulk page draft = %#v, want roster draft %q", page.Page.Draft, roster.Draft.ID)
		}
		bulkNoneRoomOptions := 0
		for _, room := range page.Page.Rooms {
			if room.ID == "none" {
				bulkNoneRoomOptions++
			}
		}
		if bulkNoneRoomOptions != 1 {
			t.Fatalf("bulk page room options have %d None entries, want exactly 1: %#v", bulkNoneRoomOptions, page.Page.Rooms)
		}

		buildBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
		})
		if err != nil {
			t.Fatalf("marshal build-list draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(buildBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("build-list draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		build := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if build.Draft.Mode != "manual_move_list" || len(build.Draft.Rows) != 0 {
			t.Fatalf("build-list draft = %#v, want empty manual list", build.Draft)
		}

		updateBody, err := json.Marshal(map[string]any{
			"mode":           build.Draft.Mode,
			"scope_site_id":  build.Draft.ScopeSiteID,
			"effective_date": "2026-07-27",
			"rows": []map[string]string{
				{"person_id": "alex-ramirez", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108", "fallback_ticket": "IT-00001", "fallback_ticket_href": "javascript:alert(1)"},
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204", "action": "removal"},
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204", "action": "add"},
			},
		})
		if err != nil {
			t.Fatalf("marshal build-list update: %v", err)
		}
		req = httptest.NewRequest(http.MethodPut, "/api/v1/dev/room-moves/drafts/"+build.Draft.ID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("build-list update returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		updated := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(updated.Draft.Rows) != 3 || updated.Draft.Rows[0].DestinationRoomID != "cla-a108" {
			t.Fatalf("updated build-list rows = %#v, want selected destination room", updated.Draft.Rows)
		}
		if updated.Draft.Rows[0].FallbackTicket != "IT-00001" || updated.Draft.Rows[0].FallbackTicketHref != "" {
			t.Fatalf("updated fallback ticket = %#v, want label retained with unsafe href removed", updated.Draft.Rows[0])
		}
		if updated.Draft.Rows[1].Action != "removal" || updated.Draft.Rows[1].DestinationRoomID != "none" {
			t.Fatalf("updated removal row = %#v, want destination room cleared to none", updated.Draft.Rows[1])
		}
		if updated.Draft.Rows[2].Action != "add" || updated.Draft.Rows[2].CurrentRoomID != "none" || updated.Draft.Rows[2].CurrentRoom != "" {
			t.Fatalf("updated add row = %#v, want current room cleared", updated.Draft.Rows[2])
		}

		repeatedBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
			"rows": []map[string]string{
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-a104", "destination_role": "primary"},
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108", "destination_role": "secondary", "action": "add"},
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204", "destination_role": "tertiary", "action": "add"},
			},
		})
		if err != nil {
			t.Fatalf("marshal repeated-user draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(repeatedBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("repeated-user draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		repeated := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(repeated.Draft.Rows) != 3 {
			t.Fatalf("repeated-user rows = %#v, want all three rows preserved", repeated.Draft.Rows)
		}
		if repeated.Draft.Rows[0].DestinationRole != "primary" || repeated.Draft.Rows[0].Phone == "Review required before primary phone assignment" {
			t.Fatalf("repeated primary row = %#v, want resolved primary desk-phone owner", repeated.Draft.Rows[0])
		}
		if repeated.Draft.Rows[1].DestinationRole != "secondary" || repeated.Draft.Rows[1].Phone != "Add to room shared line group; keep common-area phone active" {
			t.Fatalf("repeated secondary CAP row = %#v, want SLG-only CAP-preserving outcome", repeated.Draft.Rows[1])
		}
		if repeated.Draft.Rows[2].DestinationRole != "tertiary" || repeated.Draft.Rows[2].Warning != "" || repeated.Draft.Rows[2].Phone != "Add to room shared line group; no desk phone assignment" {
			t.Fatalf("repeated tertiary occupied-room row = %#v, want SLG-only outcome without primary-conflict warning", repeated.Draft.Rows[2])
		}
		if len(repeated.Draft.Warnings) != 0 {
			t.Fatalf("repeated-user draft warnings = %#v, want finalized row warnings only after SLG-only rewrite clears stale conflicts", repeated.Draft.Warnings)
		}

		inferredPrimaryBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
			"rows": []map[string]string{
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-b210"},
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108", "action": "add"},
			},
		})
		if err != nil {
			t.Fatalf("marshal inferred-primary repeated-user draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(inferredPrimaryBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("inferred-primary same-room draft returned %d, want 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Morgan Lee is already in B-210") {
			t.Fatalf("inferred-primary same-room error = %s, want stable current-room validation", rec.Body.String())
		}

		ambiguousBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
			"rows": []map[string]string{
				{"person_id": "casey-nguyen", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108", "destination_role": "primary"},
				{"person_id": "casey-nguyen", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204", "destination_role": "primary"},
			},
		})
		if err != nil {
			t.Fatalf("marshal ambiguous repeated-user draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(ambiguousBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("ambiguous repeated-user draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		ambiguous := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(ambiguous.Draft.Warnings) != 1 || !strings.Contains(ambiguous.Draft.Warnings[0], "Ambiguous repeated-user primary room") {
			t.Fatalf("ambiguous repeated-user warnings = %#v, want multi-primary warning", ambiguous.Draft.Warnings)
		}
		for _, row := range ambiguous.Draft.Rows {
			if row.Phone != "Review required before primary phone assignment" || !strings.Contains(row.AutomationOutcome, "Hold primary phone assignment") {
				t.Fatalf("ambiguous repeated-user row = %#v, want held phone assignment and actionable outcome", row)
			}
		}

		noPrimaryBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
			"rows": []map[string]string{
				{"person_id": "casey-nguyen", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108"},
				{"person_id": "casey-nguyen", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204"},
			},
		})
		if err != nil {
			t.Fatalf("marshal no-primary repeated-user draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(noPrimaryBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("no-primary repeated-user draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		noPrimary := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if len(noPrimary.Draft.Warnings) != 1 || !strings.Contains(noPrimary.Draft.Warnings[0], "needs primary selection") {
			t.Fatalf("no-primary repeated-user warnings = %#v, want primary-selection warning", noPrimary.Draft.Warnings)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+build.Draft.ID+"/schedule", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("schedule returned %d, want 200", rec.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/api/v1/dev/room-moves/drafts/"+roster.Draft.ID, nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("delete roster draft returned %d, want 204", rec.Code)
		}
	})

	t.Run("room moves cancel pending drafts and schedule IT-only completed-job reversals", func(t *testing.T) {
		itCookie := loginAsPersona(t, handler, "it_admin")
		siteAdminCookie := loginAsPersona(t, handler, "site_admin")

		cancelBody, err := json.Marshal(map[string]any{
			"mode": "mid_year_targeted_move",
			"rows": []map[string]string{
				{"person_id": "alex-ramirez", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108"},
			},
		})
		if err != nil {
			t.Fatalf("marshal cancel draft: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(cancelBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("cancel fixture draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		cancelDraft := decodeJSON[roomMoveDraftTestResponse](t, rec)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+cancelDraft.Draft.ID+"/cancel", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("cancel pending draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		canceled := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if canceled.Draft.Status != "canceled" {
			t.Fatalf("canceled draft status = %q, want canceled", canceled.Draft.Status)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/pages/room-moves", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("room moves after cancel returned %d, want 200", rec.Code)
		}
		roomMovesAfterCancel := decodeJSON[roomMovesResponse](t, rec)
		for _, row := range roomMovesAfterCancel.Page.Rows {
			if row.DraftID == cancelDraft.Draft.ID {
				t.Fatalf("canceled draft %q still appeared in review rows", cancelDraft.Draft.ID)
			}
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/room-moves/completed", nil)
		req.AddCookie(siteAdminCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("site admin completed room moves returned %d, want 403", rec.Code)
		}

		applyBody, err := json.Marshal(map[string]any{
			"mode":          "manual_move_list",
			"scope_site_id": "clover-hs",
			"rows": []map[string]string{
				{"person_id": "morgan-lee", "destination_site_id": "clover-hs", "destination_room_id": "cla-b204"},
			},
		})
		if err != nil {
			t.Fatalf("marshal apply draft: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts", bytes.NewReader(applyBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("apply fixture draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		applyDraft := decodeJSON[roomMoveDraftTestResponse](t, rec)

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+applyDraft.Draft.ID+"/apply", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("apply draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/drafts/"+applyDraft.Draft.ID+"/cancel", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("cancel applied draft returned %d, want 409", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/dev/room-moves/completed", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("completed room moves returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		completed := decodeJSON[roomMoveCompletedJobsTestResponse](t, rec)
		var appliedJobID string
		for _, job := range completed.Jobs {
			if job.SourceDraftID == applyDraft.Draft.ID {
				appliedJobID = job.ID
				if !job.CanRevert || job.RowCount != 1 {
					t.Fatalf("applied completed job = %#v, want one reversible row", job)
				}
			}
		}
		if appliedJobID == "" {
			t.Fatalf("completed jobs %#v did not include source draft %q", completed.Jobs, applyDraft.Draft.ID)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/dev/room-moves/completed/"+appliedJobID+"/revert", nil)
		req.AddCookie(itCookie)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("revert completed job returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		revert := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if revert.Draft.Status != "scheduled" || len(revert.Draft.Rows) != 1 || revert.Draft.Rows[0].DestinationRoomID == "" {
			t.Fatalf("revert draft = %#v, want scheduled reversal rows", revert.Draft)
		}
	})

	t.Run("logout clears dev session", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "site_admin")

		req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/logout", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("logout returned %d", rec.Code)
		}

		payload := decodeJSON[devSessionResponse](t, rec)
		if payload.Authenticated || payload.Authorized {
			t.Fatalf("expected signed-out response, got authenticated=%v authorized=%v", payload.Authenticated, payload.Authorized)
		}

		clearCookie := findCookie(rec.Result().Cookies(), "wizard_dev_session")
		if clearCookie == nil || clearCookie.MaxAge != -1 {
			t.Fatalf("expected cleared session cookie, got %#v", clearCookie)
		}
	})
}

func TestDevSharedMockPersonaToolingSwitchesFrontendSessionReadback(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	web.ResetDevSharedMockSessionForTest()
	t.Cleanup(web.ResetDevSharedMockSessionForTest)
	web.ResetDevFeatureFlagStateForTest()
	t.Cleanup(web.ResetDevFeatureFlagStateForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})
	staleBrowserCookie := loginAsPersona(t, handler, "it_admin")

	rec, cookie := activateSharedMockPersona(t, handler, "site_admin")
	if rec.Code != http.StatusOK {
		t.Fatalf("shared persona switch returned %d: %s", rec.Code, rec.Body.String())
	}
	if cookie == nil || cookie.Value != "site_admin" {
		t.Fatalf("shared persona switch did not issue site_admin cookie: %#v", cookie)
	}

	payload := decodeJSON[devSessionResponse](t, rec)
	if !payload.Authenticated || !payload.Authorized {
		t.Fatalf("switch response authenticated=%v authorized=%v, want true/true", payload.Authenticated, payload.Authorized)
	}
	if payload.CurrentPersona == nil || payload.CurrentPersona.ID != "site_admin" || payload.CurrentPersona.DisplayName == "" {
		t.Fatalf("switch response missing structured persona confirmation: %#v", payload.CurrentPersona)
	}
	if payload.DefaultSiteID != "clover-hs" || payload.CurrentSiteID != "clover-hs" {
		t.Fatalf("switch response site context default=%q current=%q, want clover-hs", payload.DefaultSiteID, payload.CurrentSiteID)
	}
	if len(payload.VisibleSites) != 2 {
		t.Fatalf("site_admin should keep exactly two assigned visible sites, got %#v", payload.VisibleSites)
	}
	if !slices.Contains(payload.AllowedRoutes, "/student-data-cleanup") {
		t.Fatalf("switch response allowed routes missing site-scoped route: %#v", payload.AllowedRoutes)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
	sessionReq.AddCookie(staleBrowserCookie)
	sessionRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("frontend session readback returned %d", sessionRec.Code)
	}
	sessionPayload := decodeJSON[devSessionResponse](t, sessionRec)
	if sessionPayload.CurrentPersona == nil || sessionPayload.CurrentPersona.ID != "site_admin" {
		t.Fatalf("frontend session readback used stale cookie instead of shared persona: %#v", sessionPayload.CurrentPersona)
	}
}

func TestDevSharedMockPersonaToolingSupportsNoAccessAndAllPersonas(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	web.ResetDevSharedMockSessionForTest()
	t.Cleanup(web.ResetDevSharedMockSessionForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
	sessionRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("anonymous session returned %d", sessionRec.Code)
	}
	anonymous := decodeJSON[devSessionResponse](t, sessionRec)
	seenNoAccess := false
	for _, persona := range anonymous.Personas {
		rec, _ := activateSharedMockPersona(t, handler, persona.ID)
		if rec.Code != http.StatusOK {
			t.Fatalf("activate %s returned %d: %s", persona.ID, rec.Code, rec.Body.String())
		}
		payload := decodeJSON[devSessionResponse](t, rec)
		if payload.CurrentPersona == nil || payload.CurrentPersona.ID != persona.ID {
			t.Fatalf("activate %s returned persona %#v", persona.ID, payload.CurrentPersona)
		}
		if payload.DefaultSiteID == "" || payload.CurrentSiteID == "" {
			t.Fatalf("activate %s did not include default/current site context: %#v", persona.ID, payload)
		}
		if persona.ID == "no_access" {
			seenNoAccess = true
			if !payload.Authenticated || payload.Authorized {
				t.Fatalf("no_access should be authenticated but unauthorized, got authenticated=%v authorized=%v", payload.Authenticated, payload.Authorized)
			}
			if len(payload.AllowedRoutes) != 0 {
				t.Fatalf("no_access should have no allowed routes, got %#v", payload.AllowedRoutes)
			}
		}
	}
	if !seenNoAccess {
		t.Fatal("anonymous DEV persona list did not include no_access")
	}
}

func TestDevSharedMockPersonaToolingInvalidPersonaFailsClosed(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	web.ResetDevSharedMockSessionForTest()
	t.Cleanup(web.ResetDevSharedMockSessionForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})
	staleBrowserCookie := loginAsPersona(t, handler, "it_admin")

	rec, _ := activateSharedMockPersona(t, handler, "site_admin")
	if rec.Code != http.StatusOK {
		t.Fatalf("activate site_admin returned %d: %s", rec.Code, rec.Body.String())
	}

	body := strings.NewReader(`{"persona_id":"does_not_exist","activate_mock_session":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/login", body)
	req.Header.Set("Content-Type", "application/json")
	invalidRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidRec, req)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid shared persona returned %d, want 400: %s", invalidRec.Code, invalidRec.Body.String())
	}
	clearedCookie := findCookie(invalidRec.Result().Cookies(), "wizard_dev_session")
	if clearedCookie == nil || clearedCookie.MaxAge != -1 {
		t.Fatalf("invalid shared persona should clear the DEV session cookie, got %#v", clearedCookie)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/session", nil)
	sessionReq.AddCookie(staleBrowserCookie)
	sessionRec := httptest.NewRecorder()
	handler.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("session after invalid switch returned %d", sessionRec.Code)
	}
	payload := decodeJSON[devSessionResponse](t, sessionRec)
	if payload.Authenticated || payload.Authorized || payload.CurrentPersona != nil {
		t.Fatalf("invalid shared persona should force anonymous readback, got %#v", payload)
	}
}

func TestDevSharedMockPersonaToolingDeniedOutsideDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	web.ResetDevSharedMockSessionForTest()
	t.Cleanup(web.ResetDevSharedMockSessionForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})
	rec, _ := activateSharedMockPersona(t, handler, "site_admin")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("shared persona tooling in production returned %d, want 404", rec.Code)
	}
}

func TestDevMyProfileDirectEditMockState(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	handler := web.NewAppHandler(web.HealthDependencies{})

	t.Run("requires an authenticated DEV persona", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dev/my-profile", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous profile returned %d, want 401", rec.Code)
		}
	})

	t.Run("eligible persona can save preferred display name and pronouns", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "faculty_staff")

		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/my-profile", nil)
		getReq.AddCookie(cookie)
		getRec := httptest.NewRecorder()
		handler.ServeHTTP(getRec, getReq)
		if getRec.Code != http.StatusOK {
			t.Fatalf("profile get returned %d, want 200", getRec.Code)
		}
		initial := decodeJSON[myProfileResponse](t, getRec)
		if initial.PageID != "my-profile" || !initial.Profile.Editable {
			t.Fatalf("unexpected initial profile payload: %#v", initial)
		}
		if initial.Profile.DisplayName != "Avery Shah" {
			t.Fatalf("display name = %q, want Avery Shah", initial.Profile.DisplayName)
		}

		body, err := json.Marshal(map[string]string{
			"preferred_first_name": "  Ave ",
			"preferred_last_name":  " Shah-Lewis ",
			"pronouns":             " They / Them ",
		})
		if err != nil {
			t.Fatalf("marshal profile update: %v", err)
		}
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/my-profile", bytes.NewReader(body))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusOK {
			t.Fatalf("profile update returned %d, want 200", updateRec.Code)
		}
		updated := decodeJSON[myProfileResponse](t, updateRec)
		if updated.Profile.DisplayName != "Ave Shah-Lewis" {
			t.Fatalf("display name = %q, want Ave Shah-Lewis", updated.Profile.DisplayName)
		}
		if updated.Profile.Pronouns != "They / Them" {
			t.Fatalf("pronouns = %q, want They / Them", updated.Profile.Pronouns)
		}

		verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/dev/my-profile", nil)
		verifyReq.AddCookie(cookie)
		verifyRec := httptest.NewRecorder()
		handler.ServeHTTP(verifyRec, verifyReq)
		if verifyRec.Code != http.StatusOK {
			t.Fatalf("profile verify returned %d, want 200", verifyRec.Code)
		}
		verified := decodeJSON[myProfileResponse](t, verifyRec)
		if verified.Profile.DisplayName != "Ave Shah-Lewis" {
			t.Fatalf("persisted display name = %q, want Ave Shah-Lewis", verified.Profile.DisplayName)
		}
		if verified.Profile.LegalName != "Avery Shah" {
			t.Fatalf("legal name changed to %q, want source legal name Avery Shah", verified.Profile.LegalName)
		}
	})

	t.Run("validation rejects missing preferred name fields", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "faculty_staff")
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/dev/my-profile", strings.NewReader(`{"preferred_first_name":"","preferred_last_name":"","pronouns":""}`))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq.AddCookie(cookie)
		updateRec := httptest.NewRecorder()
		handler.ServeHTTP(updateRec, updateReq)
		if updateRec.Code != http.StatusBadRequest {
			t.Fatalf("invalid profile update returned %d, want 400", updateRec.Code)
		}
		if !strings.Contains(updateRec.Body.String(), "preferred_first_name") {
			t.Fatalf("validation body missing field errors: %s", updateRec.Body.String())
		}
	})
}

func TestDevRouteAPIAuthorizationInventoryCoverage(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	web.ResetDevFeatureFlagStateForTest()
	t.Cleanup(web.ResetDevFeatureFlagStateForTest)
	web.ResetDevDepartingSeniorsStateForTest()
	t.Cleanup(web.ResetDevDepartingSeniorsStateForTest)

	handler := web.NewAppHandler(web.HealthDependencies{})

	protectedEndpoints := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "search", method: http.MethodGet, path: "/api/v1/dev/search?q=alex"},
		{name: "onboarding page", method: http.MethodGet, path: "/api/v1/dev/pages/onboarding"},
		{name: "onboarding draft create", method: http.MethodPost, path: "/api/v1/dev/onboarding/manual-drafts"},
		{name: "onboarding draft update", method: http.MethodPut, path: "/api/v1/dev/onboarding/manual-drafts/draft-unknown", body: `{}`},
		{name: "onboarding room update", method: http.MethodPut, path: "/api/v1/dev/onboarding/rows/jordan-miles/room", body: `{"room_id":"iiq-room-cla-108"}`},
		{name: "offboarding page", method: http.MethodGet, path: "/api/v1/dev/pages/offboarding"},
		{name: "offboarding end date", method: http.MethodPut, path: "/api/v1/dev/offboarding/records/orphan-avery-cole/end-date", body: `{}`},
		{name: "departing seniors page", method: http.MethodGet, path: "/api/v1/dev/pages/departing-seniors"},
		{name: "departing seniors end date", method: http.MethodPut, path: "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/end-date", body: `{}`},
		{name: "departing seniors deprovision", method: http.MethodPost, path: "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/deprovision"},
		{name: "data quality page", method: http.MethodGet, path: "/api/v1/dev/pages/data-quality"},
		{name: "room moves page", method: http.MethodGet, path: "/api/v1/dev/pages/room-moves"},
		{name: "room moves bulk page", method: http.MethodGet, path: "/api/v1/dev/pages/room-moves/bulk-draft"},
		{name: "room moves draft create", method: http.MethodPost, path: "/api/v1/dev/room-moves/drafts", body: `{}`},
		{name: "room moves draft update", method: http.MethodPut, path: "/api/v1/dev/room-moves/drafts/rm-draft-103", body: `{}`},
		{name: "room moves completed list", method: http.MethodGet, path: "/api/v1/dev/room-moves/completed"},
		{name: "room moves completed revert", method: http.MethodPost, path: "/api/v1/dev/room-moves/completed/job-unknown/revert"},
		{name: "phone directory person", method: http.MethodGet, path: "/api/v1/dev/pages/phone-directory/by-person"},
		{name: "phone directory room", method: http.MethodGet, path: "/api/v1/dev/pages/phone-directory/by-room"},
		{name: "phone directory department", method: http.MethodGet, path: "/api/v1/dev/pages/phone-directory/by-department"},
		{name: "security issues report", method: http.MethodGet, path: "/api/v1/dev/pages/reports/security-issues"},
		{name: "feature flags", method: http.MethodGet, path: "/api/v1/dev/feature-flags"},
		{name: "feature flag update", method: http.MethodPut, path: "/api/v1/dev/feature-flags/onboarding", body: `{"targets":[]}`},
		{name: "my profile", method: http.MethodGet, path: "/api/v1/dev/my-profile"},
		{name: "my profile update", method: http.MethodPut, path: "/api/v1/dev/my-profile", body: `{}`},
	}

	for _, endpoint := range protectedEndpoints {
		t.Run("signed out 401 "+endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s returned %d, want 401: %s", endpoint.method, endpoint.path, rec.Code, rec.Body.String())
			}
		})
	}

	forbiddenEndpoints := []struct {
		name      string
		method    string
		path      string
		body      string
		personaID string
	}{
		{name: "onboarding page", method: http.MethodGet, path: "/api/v1/dev/pages/onboarding", personaID: "faculty_staff"},
		{name: "onboarding draft create", method: http.MethodPost, path: "/api/v1/dev/onboarding/manual-drafts", personaID: "faculty_staff"},
		{name: "onboarding room update", method: http.MethodPut, path: "/api/v1/dev/onboarding/rows/jordan-miles/room", body: `{"room_id":"iiq-room-cla-108"}`, personaID: "faculty_staff"},
		{name: "offboarding page", method: http.MethodGet, path: "/api/v1/dev/pages/offboarding", personaID: "faculty_staff"},
		{name: "offboarding end date", method: http.MethodPut, path: "/api/v1/dev/offboarding/records/orphan-avery-cole/end-date", body: `{}`, personaID: "site_admin"},
		{name: "departing seniors page", method: http.MethodGet, path: "/api/v1/dev/pages/departing-seniors", personaID: "faculty_staff"},
		{name: "departing seniors end date", method: http.MethodPut, path: "/api/v1/dev/departing-seniors/records/senior-luis-alvarez/end-date", body: `{}`, personaID: "faculty_staff"},
		{name: "data quality page", method: http.MethodGet, path: "/api/v1/dev/pages/data-quality", personaID: "site_admin"},
		{name: "room moves page", method: http.MethodGet, path: "/api/v1/dev/pages/room-moves", personaID: "faculty_staff"},
		{name: "room moves draft create", method: http.MethodPost, path: "/api/v1/dev/room-moves/drafts", body: `{}`, personaID: "faculty_staff"},
		{name: "room moves completed revert", method: http.MethodPost, path: "/api/v1/dev/room-moves/completed/job-unknown/revert", personaID: "site_admin"},
		{name: "security issues report", method: http.MethodGet, path: "/api/v1/dev/pages/reports/security-issues", personaID: "human_resources"},
		{name: "feature flags", method: http.MethodGet, path: "/api/v1/dev/feature-flags", personaID: "site_admin"},
		{name: "feature flag update", method: http.MethodPut, path: "/api/v1/dev/feature-flags/onboarding", body: `{"targets":[]}`, personaID: "site_admin"},
	}

	for _, endpoint := range forbiddenEndpoints {
		t.Run("forbidden 403 "+endpoint.name, func(t *testing.T) {
			cookie := loginAsPersona(t, handler, endpoint.personaID)
			req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
			req.AddCookie(cookie)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s as %s returned %d, want 403: %s", endpoint.method, endpoint.path, endpoint.personaID, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestDevFrontendRoutesDisabledOutsideDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	handler := web.NewAppHandler(web.HealthDependencies{})

	for _, path := range []string{
		"/api/v1/dev/session",
		"/api/v1/dev/login",
		"/api/v1/dev/logout",
		"/api/v1/dev/search",
		"/api/v1/dev/pages/onboarding",
		"/api/v1/dev/onboarding/manual-drafts",
		"/api/v1/dev/pages/data-quality",
		"/api/v1/dev/pages/phone-directory/by-person",
	} {
		method := http.MethodGet
		if path == "/api/v1/dev/login" || path == "/api/v1/dev/logout" || path == "/api/v1/dev/onboarding/manual-drafts" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d, want 404", path, rec.Code)
		}
	}
}

func TestDevFrontendRoutesDisabledWhenAppEnvUnset(t *testing.T) {
	handler := web.NewAppHandler(web.HealthDependencies{})

	for _, path := range []string{
		"/api/v1/dev/session",
		"/api/v1/dev/login",
		"/api/v1/dev/logout",
		"/api/v1/dev/search",
		"/api/v1/dev/pages/onboarding",
		"/api/v1/dev/onboarding/manual-drafts",
		"/api/v1/dev/pages/data-quality",
		"/api/v1/dev/pages/phone-directory/by-person",
	} {
		method := http.MethodGet
		if path == "/api/v1/dev/login" || path == "/api/v1/dev/logout" || path == "/api/v1/dev/onboarding/manual-drafts" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s returned %d, want 404", path, rec.Code)
		}
	}
}
