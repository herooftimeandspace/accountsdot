package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type breakglassErrorResponse struct {
	Code string `json:"code"`
}

type failingBreakglassAuditStore struct{}

func (failingBreakglassAuditStore) RecordBreakglassAudit(context.Context, breakglassAuditEvent) error {
	return errors.New("audit store unavailable")
}

func TestDomainGateAllowsBreakglassButKeepsStudentDenial(t *testing.T) {
	cases := []struct {
		name            string
		email           string
		localBreakglass bool
		want            bool
	}{
		{name: "staff domain", email: "alex@wusd.org", want: true},
		{name: "it subdomain", email: "alex@it.wusd.org", want: true},
		{name: "staff subdomain", email: "alex@staff.wusd.org", want: true},
		{name: "student domain denied", email: "student@stu.wusd.org", want: false},
		{name: "outside domain denied", email: "alex@example.com", want: false},
		{name: "local breakglass exception", email: "emergency-alex", localBreakglass: true, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domainGateAllowsDashboardEmail(tc.email, tc.localBreakglass); got != tc.want {
				t.Fatalf("domainGateAllowsDashboardEmail(%q, %v) = %v, want %v", tc.email, tc.localBreakglass, got, tc.want)
			}
		})
	}
}

func TestBreakglassLoginFailsClosedWhenAuditWriteFails(t *testing.T) {
	configureBreakglassForPackageTest(t, "emergency-alex", "local-test-token")
	breakglassAuditStoreMu.Lock()
	breakglassAuditStore = failingBreakglassAuditStore{}
	breakglassAuditStoreError = nil
	breakglassAuditStoreMu.Unlock()

	rec := postBreakglassLoginForPackageTest(t, "emergency-alex", "local-test-token", "10.23.4.5:62000", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("breakglass login with failing audit store returned %d, want 503: %s", rec.Code, rec.Body.String())
	}
	if cookie := findBreakglassCookieForPackageTest(rec); cookie != nil {
		t.Fatalf("breakglass login with failing audit store set cookie %#v", cookie)
	}
	payload := decodeBreakglassErrorForPackageTest(t, rec)
	if payload.Code != "breakglass_audit_unavailable" {
		t.Fatalf("error code = %q, want breakglass_audit_unavailable", payload.Code)
	}
}

func TestBreakglassLoginFailsClosedWhenAuditStoreInitializationFails(t *testing.T) {
	configureBreakglassForPackageTest(t, "emergency-alex", "local-test-token")
	t.Setenv("DATABASE_URL", "://not-a-valid-database-url")
	ResetBreakglassAuditForTest()

	rec := postBreakglassLoginForPackageTest(t, "emergency-alex", "local-test-token", "10.23.4.5:62000", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("breakglass login with invalid database URL returned %d, want 503: %s", rec.Code, rec.Body.String())
	}
	if cookie := findBreakglassCookieForPackageTest(rec); cookie != nil {
		t.Fatalf("breakglass login with invalid database URL set cookie %#v", cookie)
	}
}

func TestBreakglassSourceIPIgnoresUntrustedForwardedFor(t *testing.T) {
	configureBreakglassForPackageTest(t, "emergency-alex", "local-test-token")
	rec := postBreakglassLoginForPackageTest(t, "emergency-alex", "local-test-token", "192.0.2.10:62000", map[string]string{
		"X-Forwarded-For": "10.23.4.5",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("untrusted forwarded-for login returned %d, want 403: %s", rec.Code, rec.Body.String())
	}
	if cookie := findBreakglassCookieForPackageTest(rec); cookie != nil {
		t.Fatalf("untrusted forwarded-for login set cookie %#v", cookie)
	}
	audits := BreakglassAuditEventsForTest()
	if len(audits) != 1 || audits[0].SourceIP != "192.0.2.10" {
		t.Fatalf("untrusted forwarded-for audit = %#v, want RemoteAddr source", audits)
	}
}

func TestBreakglassSourceIPTrustsForwardedForFromConfiguredProxy(t *testing.T) {
	configureBreakglassForPackageTest(t, "emergency-alex", "local-test-token")
	t.Setenv("BREAKGLASS_TRUSTED_PROXY_CIDRS", "192.0.2.0/24")

	rec := postBreakglassLoginForPackageTest(t, "emergency-alex", "local-test-token", "192.0.2.10:62000", map[string]string{
		"X-Forwarded-For": "10.23.4.5",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("trusted forwarded-for login returned %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if cookie := findBreakglassCookieForPackageTest(rec); cookie == nil {
		t.Fatalf("trusted forwarded-for login did not set session cookie")
	}
	audits := BreakglassAuditEventsForTest()
	if len(audits) != 2 || audits[0].SourceIP != "10.23.4.5" || audits[1].SourceIP != "10.23.4.5" {
		t.Fatalf("trusted forwarded-for audit = %#v, want client source from forwarded header", audits)
	}
}

func TestBreakglassRejectsAccountIDsWithCollidingTokenEnvNames(t *testing.T) {
	configureBreakglassForPackageTest(t, "emergency-alex", "local-test-token")
	t.Setenv("BREAKGLASS_ACCOUNTS", "emergency-alex,emergency.alex")

	rec := postBreakglassLoginForPackageTest(t, "emergency-alex", "local-test-token", "10.23.4.5:62000", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("colliding account config returned %d, want 500: %s", rec.Code, rec.Body.String())
	}
	if cookie := findBreakglassCookieForPackageTest(rec); cookie != nil {
		t.Fatalf("colliding account config set cookie %#v", cookie)
	}
	payload := decodeBreakglassErrorForPackageTest(t, rec)
	if payload.Code != "breakglass_configuration_invalid" {
		t.Fatalf("error code = %q, want breakglass_configuration_invalid", payload.Code)
	}
}

func configureBreakglassForPackageTest(t *testing.T, accountID string, token string) {
	t.Helper()
	hash := sha256.Sum256([]byte(token))
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("BREAKGLASS_ACCOUNTS", accountID)
	t.Setenv(breakglassTokenHashEnvName(accountID), hex.EncodeToString(hash[:]))
	t.Setenv("BREAKGLASS_ALLOWED_CIDRS", "10.23.0.0/16,10.19.100.0/24")
	t.Setenv("BREAKGLASS_TRUSTED_PROXY_CIDRS", "")
	ResetBreakglassAuditForTest()
	t.Cleanup(ResetBreakglassAuditForTest)
}

func postBreakglassLoginForPackageTest(t *testing.T, accountID string, token string, remoteAddr string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"account_id": accountID, "token": token})
	if err != nil {
		t.Fatalf("marshal breakglass login request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/breakglass/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	NewAppHandler(HealthDependencies{}).ServeHTTP(rec, req)
	return rec
}

func findBreakglassCookieForPackageTest(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == devSessionCookieName {
			return cookie
		}
	}
	return nil
}

func decodeBreakglassErrorForPackageTest(t *testing.T, rec *httptest.ResponseRecorder) breakglassErrorResponse {
	t.Helper()
	var payload breakglassErrorResponse
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return payload
}
