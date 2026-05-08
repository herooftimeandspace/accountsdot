package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

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
		Mode            string `json:"mode"`
		Query           string `json:"query"`
		CurrentSiteID   string `json:"current_site_id"`
		CurrentSiteName string `json:"current_site_name"`
		SelectedResult  *struct {
			ID string `json:"id"`
		} `json:"selected_result,omitempty"`
		Results []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			TypeLabel       string `json:"type_label"`
			SiteID          string `json:"site_id"`
			Title           string `json:"title"`
			Extension       string `json:"extension"`
			ExtensionLength int    `json:"extension_length"`
			ExtensionValid  bool   `json:"extension_valid"`
		} `json:"results"`
	} `json:"page"`
}

type onboardingResponse struct {
	PageID string `json:"page_id"`
	Page   struct {
		CanManageManual bool `json:"can_manage_manual"`
		Rows            []struct {
			Kind            string `json:"kind"`
			DateAdded       string `json:"date_added"`
			DateAddedReason string `json:"date_added_reason"`
			ManualDraftID   string `json:"manual_draft_id"`
			WorkflowStatus  string `json:"workflow_status"`
			AssignedEmail   string `json:"assigned_email"`
			EmployeeNumber  string `json:"employee_number"`
			WorkflowSteps   []struct {
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
		ID                  string   `json:"id"`
		Status              string   `json:"status"`
		FirstName           string   `json:"first_name"`
		LastName            string   `json:"last_name"`
		PersonalEmail       string   `json:"personal_email"`
		GeneratedEmail      string   `json:"generated_email"`
		GeneratedEmployeeID string   `json:"generated_employee_id"`
		MissingFields       []string `json:"missing_fields"`
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

		siteCookie := loginAsPersona(t, handler, "site_admin")
		siteReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/onboarding/manual-drafts", nil)
		siteReq.AddCookie(siteCookie)
		siteRec := httptest.NewRecorder()
		handler.ServeHTTP(siteRec, siteReq)
		if siteRec.Code != http.StatusForbidden {
			t.Fatalf("site admin draft create returned %d, want 403", siteRec.Code)
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

		hasClassroomSharedLine := false
		for _, result := range payload.Page.Results {
			switch result.Type {
			case "common_area":
			case "classroom_slg":
				hasClassroomSharedLine = hasClassroomSharedLine || result.Type == "classroom_slg"
			default:
				t.Fatalf("room mode returned disallowed result type %q for %#v", result.Type, result)
			}
		}
		if !hasClassroomSharedLine {
			t.Fatal("expected at least one classroom shared line result in room mode")
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
