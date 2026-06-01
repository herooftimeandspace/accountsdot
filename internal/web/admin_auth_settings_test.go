package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/web"
)

type authSettingsSnapshotResponse struct {
	RoleMappings []struct {
		ID          int64    `json:"id"`
		SourceType  string   `json:"source_type"`
		SourceValue string   `json:"source_value"`
		RoleKeys    []string `json:"role_keys"`
	} `json:"role_mappings"`
	SiteScopeMappings []struct {
		ID          int64    `json:"id"`
		SourceType  string   `json:"source_type"`
		SourceValue string   `json:"source_value"`
		SiteCodes   []string `json:"site_codes"`
	} `json:"site_scope_mappings"`
	ExternalSources []struct {
		ProviderKey     string `json:"provider_key"`
		SyncEnabled     bool   `json:"sync_enabled"`
		LastTestStatus  string `json:"last_test_status"`
		LastTestSummary string `json:"last_test_summary"`
		Credentials     []struct {
			FieldKey    string `json:"field_key"`
			Fingerprint string `json:"fingerprint"`
			Stored      bool   `json:"stored"`
		} `json:"credentials"`
	} `json:"external_sources"`
	AuditEvents []struct {
		TargetEntity string          `json:"target_entity"`
		TargetID     string          `json:"target_id"`
		Diff         json.RawMessage `json:"diff"`
	} `json:"audit_events"`
}

type authPreviewResponse struct {
	Authorized          bool     `json:"authorized"`
	Roles               []string `json:"roles"`
	SiteScopes          []string `json:"site_scopes"`
	ValidationFailures  []string `json:"validation_failures"`
	ProductionLoginMode string   `json:"production_login"`
}

func TestAdminAuthSettingsRequiresITAdminAndDefaultsExternalSourcesOff(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	web.ResetAdminAuthSettingsForTest()
	t.Cleanup(web.ResetAdminAuthSettingsForTest)
	handler := web.NewAppHandler(web.HealthDependencies{})

	unauth := httptest.NewRecorder()
	handler.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth-settings", nil))
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated auth settings = %d, want 401", unauth.Code)
	}

	siteCookie := loginAsPersona(t, handler, "site_secretary")
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth-settings", nil)
	forbiddenReq.AddCookie(siteCookie)
	forbidden := httptest.NewRecorder()
	handler.ServeHTTP(forbidden, forbiddenReq)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("site secretary auth settings = %d, want 403", forbidden.Code)
	}

	itCookie := loginAsPersona(t, handler, "it_admin")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth-settings", nil)
	req.AddCookie(itCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("it admin auth settings = %d, want 200", rec.Code)
	}
	payload := decodeJSON[authSettingsSnapshotResponse](t, rec)
	if len(payload.ExternalSources) != 4 {
		t.Fatalf("external sources = %d, want configured Phase 0 providers", len(payload.ExternalSources))
	}
	for _, source := range payload.ExternalSources {
		if source.SyncEnabled {
			t.Fatalf("source %s defaulted sync on", source.ProviderKey)
		}
	}
}

func TestAdminAuthSettingsPreviewAndCredentialsStaySanitized(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ENCRYPTION_KEY", "test-key:01234567890123456789012345678901")
	web.ResetAdminAuthSettingsForTest()
	t.Cleanup(web.ResetAdminAuthSettingsForTest)
	handler := web.NewAppHandler(web.HealthDependencies{})
	cookie := loginAsPersona(t, handler, "it_admin")

	postJSON(t, handler, cookie, "/api/v1/admin/auth-settings/role-mappings", map[string]any{
		"source_type":  "group",
		"source_value": "wizard-it-admins@wusd.org",
		"role_keys":    []string{"it_admin"},
		"reason":       "test role mapping",
	}, http.StatusOK)
	postJSON(t, handler, cookie, "/api/v1/admin/auth-settings/site-scope-mappings", map[string]any{
		"source_type":  "ou",
		"source_value": "/Staff/Clover",
		"site_codes":   []string{"clover-hs"},
		"reason":       "test site scope mapping",
	}, http.StatusOK)

	previewRec := postJSON(t, handler, cookie, "/api/v1/admin/auth-settings/preview", map[string]any{
		"email":  "casey@staff.wusd.org",
		"groups": []string{"wizard-it-admins@wusd.org"},
		"ous":    []string{"/Staff/Clover"},
	}, http.StatusOK)
	preview := decodeJSON[authPreviewResponse](t, previewRec)
	if !preview.Authorized || !slicesContain(preview.Roles, "it_admin") || !slicesContain(preview.SiteScopes, "clover-hs") {
		t.Fatalf("preview = %#v, want it_admin and clover-hs", preview)
	}
	if preview.ProductionLoginMode != "disabled" {
		t.Fatalf("production login mode = %q, want disabled", preview.ProductionLoginMode)
	}

	secret := "super-secret-aeries-cert-reference"
	credentialRec := putJSON(t, handler, cookie, "/api/v1/admin/external-sources/aeries/credentials", map[string]any{
		"fields": map[string]string{
			"base_url":              "https://aeries.example.invalid",
			"certificate_reference": secret,
		},
		"labels": map[string]string{
			"certificate_reference": "aeries cert in vault",
		},
		"reason": "test encrypted credential save",
	}, http.StatusOK)
	if strings.Contains(credentialRec.Body.String(), secret) {
		t.Fatalf("credential response leaked plaintext secret: %s", credentialRec.Body.String())
	}

	testRec := postJSON(t, handler, cookie, "/api/v1/admin/external-sources/aeries/test", map[string]any{
		"reason": "validate saved encrypted aeries credentials",
	}, http.StatusOK)
	if strings.Contains(testRec.Body.String(), secret) {
		t.Fatalf("test response leaked plaintext secret: %s", testRec.Body.String())
	}

	toggleRec := patchJSON(t, handler, cookie, "/api/v1/admin/external-sources/aeries", map[string]any{
		"sync_enabled": true,
		"reason":       "record explicit operator toggle",
	}, http.StatusOK)
	if strings.Contains(toggleRec.Body.String(), secret) {
		t.Fatalf("toggle response leaked plaintext secret: %s", toggleRec.Body.String())
	}
}

func TestAdminAuthSettingsRequiresChangeReason(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	web.ResetAdminAuthSettingsForTest()
	t.Cleanup(web.ResetAdminAuthSettingsForTest)
	handler := web.NewAppHandler(web.HealthDependencies{})
	cookie := loginAsPersona(t, handler, "it_admin")

	postJSON(t, handler, cookie, "/api/v1/admin/auth-settings/role-mappings", map[string]any{
		"source_type":  "group",
		"source_value": "wizard-it-admins@wusd.org",
		"role_keys":    []string{"it_admin"},
	}, http.StatusBadRequest)
	patchJSON(t, handler, cookie, "/api/v1/admin/external-sources/google", map[string]any{
		"sync_enabled": true,
	}, http.StatusBadRequest)
	postJSON(t, handler, cookie, "/api/v1/admin/external-sources/google/test", map[string]any{
		"reason": "missing encrypted credential test",
	}, http.StatusBadRequest)
}

func postJSON(t *testing.T, handler http.Handler, cookie *http.Cookie, path string, payload any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	return requestJSON(t, handler, cookie, http.MethodPost, path, payload, wantStatus)
}

func putJSON(t *testing.T, handler http.Handler, cookie *http.Cookie, path string, payload any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	return requestJSON(t, handler, cookie, http.MethodPut, path, payload, wantStatus)
}

func patchJSON(t *testing.T, handler http.Handler, cookie *http.Cookie, path string, payload any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	return requestJSON(t, handler, cookie, http.MethodPatch, path, payload, wantStatus)
}

func requestJSON(t *testing.T, handler http.Handler, cookie *http.Cookie, method string, path string, payload any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s %s: %v", method, path, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s returned %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}

func slicesContain(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
