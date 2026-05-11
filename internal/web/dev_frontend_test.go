package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

type devSessionResponse struct {
	Authenticated   bool   `json:"authenticated"`
	Authorized      bool   `json:"authorized"`
	DefaultSiteID   string `json:"default_site_id"`
	DefaultSiteName string `json:"default_site_name"`
	CurrentSiteID   string `json:"current_site_id"`
	CurrentSiteName string `json:"current_site_name"`
	CurrentPersona  *struct {
		ID string `json:"id"`
	} `json:"current_persona,omitempty"`
	LandingPath   string   `json:"landing_path"`
	AllowedRoutes []string `json:"allowed_routes"`
	Personas      []struct {
		ID string `json:"id"`
	} `json:"personas"`
}

type dataQualityResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		Title string `json:"title"`
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
	} `json:"form"`
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

type departingSeniorsResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManage      bool   `json:"can_manage"`
		GraduationYear string `json:"graduation_year"`
		Rows           []struct {
			ID                 string `json:"id"`
			FirstName          string `json:"first_name"`
			LastName           string `json:"last_name"`
			Email              string `json:"email"`
			StudentID          string `json:"student_id"`
			GraduationYear     string `json:"graduation_year"`
			EndDate            string `json:"end_date"`
			EndDateSource      string `json:"end_date_source"`
			Status             string `json:"status"`
			CanOverrideEndDate bool   `json:"can_override_end_date"`
			CanDeprovision     bool   `json:"can_deprovision"`
			Deprovisioned      bool   `json:"deprovisioned"`
			OutstandingDevices []struct {
				AssetID string `json:"asset_id"`
				Serial  string `json:"serial"`
				Type    string `json:"type"`
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
		} `json:"people"`
		Rows []struct {
			ID            string `json:"id"`
			DraftID       string `json:"draft_id"`
			MoveType      string `json:"move_type"`
			CurrentSiteID string `json:"current_site_id"`
			Warning       string `json:"warning"`
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
	CanManageDistrict bool     `json:"can_manage_district"`
	Warnings          []string `json:"warnings"`
	Rows              []struct {
		PersonID          string `json:"person_id"`
		CurrentSiteID     string `json:"current_site_id"`
		CurrentRoomID     string `json:"current_room_id"`
		DestinationSiteID string `json:"destination_site_id"`
		DestinationRoomID string `json:"destination_room_id"`
		Warning           string `json:"warning"`
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

	handler := web.NewAppHandler(web.HealthDependencies{})

	t.Run("session is anonymous before login", func(t *testing.T) {
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
		if !slices.Contains(sessionPayload.AllowedRoutes, "/data-quality") {
			t.Fatalf("expected /data-quality in allowed routes: %#v", sessionPayload.AllowedRoutes)
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
		if pagePayload.Hotspots["refresh"].NodeID != "f104" {
			t.Fatalf("refresh hotspot node = %q, want f104", pagePayload.Hotspots["refresh"].NodeID)
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

	t.Run("human resources data quality is 403", func(t *testing.T) {
		cookie := loginAsPersona(t, handler, "human_resources")

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
		if !itPayload.Page.CanManageEndDates || !itPayload.Page.ShowEmployeeIDs {
			t.Fatalf("it flags = manage:%v ids:%v, want true/true", itPayload.Page.CanManageEndDates, itPayload.Page.ShowEmployeeIDs)
		}
		if len(itPayload.Page.Rows) < 6 {
			t.Fatalf("it rows = %d, want seeded escape and orphan rows", len(itPayload.Page.Rows))
		}
		if itPayload.Page.Rows[0].EmployeeID == "" {
			t.Fatalf("it row employee id missing: %#v", itPayload.Page.Rows[0])
		}
		foundOrphanAction := false
		foundNamedLicenseReclaim := false
		for _, row := range itPayload.Page.Rows {
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
		if sitePayload.Page.CanManageEndDates || sitePayload.Page.ShowEmployeeIDs {
			t.Fatalf("site flags = manage:%v ids:%v, want false/false", sitePayload.Page.CanManageEndDates, sitePayload.Page.ShowEmployeeIDs)
		}
		for _, row := range sitePayload.Page.Rows {
			if row.EmployeeID != "" {
				t.Fatalf("site admin received employee id in row %#v", row)
			}
			if row.SiteID != "clover-hs" && row.SiteID != "desert-view" {
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
		if !hrPayload.Page.CanManageEndDates || !hrPayload.Page.ShowEmployeeIDs {
			t.Fatalf("hr flags = manage:%v ids:%v, want true/true", hrPayload.Page.CanManageEndDates, hrPayload.Page.ShowEmployeeIDs)
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
		if rec.Code != http.StatusOK {
			t.Fatalf("hr orphan end date update returned %d, want 200: %s", rec.Code, rec.Body.String())
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

	t.Run("departing seniors page is scoped to it and device wranglers", func(t *testing.T) {
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
		foundDevice := false
		for _, row := range itPayload.Page.Rows {
			if row.GraduationYear != itPayload.Page.GraduationYear {
				t.Fatalf("row graduation year = %s, page year = %s", row.GraduationYear, itPayload.Page.GraduationYear)
			}
			if row.Email == "" || row.StudentID == "" {
				t.Fatalf("row missing searchable identity data: %#v", row)
			}
			if len(row.OutstandingDevices) > 0 {
				foundDevice = true
				if row.OutstandingDevices[0].AssetID == "" || row.OutstandingDevices[0].Serial == "" {
					t.Fatalf("device row missing incidentiq identifiers: %#v", row.OutstandingDevices[0])
				}
			}
		}
		if !foundDevice {
			t.Fatal("expected at least one senior with outstanding IncidentIQ device data")
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
	})

	t.Run("departing seniors updates end dates and removes rows only after deprovision without devices", func(t *testing.T) {
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

	t.Run("past-dated manual entry shows warning fields and schedules next cycle", func(t *testing.T) {
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
		if !updated.Draft.LateStart {
			t.Fatal("expected manual past-date draft to be marked late_start")
		}
		if updated.Draft.ScheduledFor == "" {
			t.Fatal("expected manual past-date draft to expose scheduled_for")
		}
		if updated.Draft.EffectiveDate != pastDate {
			t.Fatalf("effective date = %q, want %q", updated.Draft.EffectiveDate, pastDate)
		}
	})

	t.Run("escape-backed past-date row preserves source date and exposes next-cycle schedule", func(t *testing.T) {
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
			if !row.LateStart {
				t.Fatal("expected Nia Brooks to be marked late_start")
			}
			if row.ScheduledFor == "" {
				t.Fatal("expected Nia Brooks to expose scheduled_for")
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
		if len(sitePayload.Page.Rooms) == 0 || sitePayload.Page.Rooms[0].ID != "none" {
			t.Fatalf("rooms = %#v, want None option first", sitePayload.Page.Rooms)
		}
		for _, room := range sitePayload.Page.Rooms {
			if room.SiteID != "clover-hs" {
				t.Fatalf("site secretary received out-of-site room option %#v", room)
			}
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
		if rec.Code != http.StatusOK {
			t.Fatalf("site-scoped single draft returned %d, want 200: %s", rec.Code, rec.Body.String())
		}
		created := decodeJSON[roomMoveDraftTestResponse](t, rec)
		if created.Draft.CanManageDistrict {
			t.Fatal("site-scoped draft should not expose district controls")
		}
		if len(created.Draft.Rows) != 1 || created.Draft.Rows[0].DestinationRoomID != "cla-b210" {
			t.Fatalf("created draft rows = %#v, want current room as same-site default", created.Draft.Rows)
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
				{"person_id": "alex-ramirez", "destination_site_id": "clover-hs", "destination_room_id": "cla-a108"},
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
		if len(updated.Draft.Rows) != 1 || updated.Draft.Rows[0].DestinationRoomID != "cla-a108" {
			t.Fatalf("updated build-list rows = %#v, want selected destination room", updated.Draft.Rows)
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
			"mode":      "mid_year_targeted_move",
			"person_id": "alex-ramirez",
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
