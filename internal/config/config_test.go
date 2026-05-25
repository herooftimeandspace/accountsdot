package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/auth"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/provider"
)

// TestLoadDefaults verifies that local startup receives the documented default
// ports, limits, and production-auth domain gate without requiring any Google
// SAML secrets or admin-provided mapping JSON.
func TestLoadDefaults(t *testing.T) {
	clearProductionAuthEnv(t)
	clearProviderReadinessEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppPort != "8080" {
		t.Fatalf("expected default app port 8080, got %q", cfg.AppPort)
	}
	if cfg.ZoomSLGMaxMembers != 10 {
		t.Fatalf("expected default zoom SLG max members 10, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 100 {
		t.Fatalf("expected replay limit 100, got %d", cfg.ReplayEventLimit)
	}
	if !slices.Equal(cfg.ProductionAuthPolicy.AllowedEmailDomains, []string{"it.wusd.org", "staff.wusd.org", "wusd.org"}) {
		t.Fatalf("unexpected allowed domains: %#v", cfg.ProductionAuthPolicy.AllowedEmailDomains)
	}
	if !slices.Equal(cfg.ProductionAuthPolicy.DeniedEmailDomains, []string{"stu.wusd.org"}) {
		t.Fatalf("unexpected denied domains: %#v", cfg.ProductionAuthPolicy.DeniedEmailDomains)
	}
	for _, readiness := range cfg.ProviderReadiness {
		if !readiness.UseMock {
			t.Fatalf("%s readiness default UseMock = false, want true for local DEV safety", readiness.Provider)
		}
		if !readiness.ReadOnly {
			t.Fatalf("%s readiness ReadOnly = false, want true", readiness.Provider)
		}
	}
}

// TestP000D001ProviderReadinessConfigDefaults records the startup-side
// evidence for Phase 0 provider mock readiness. Config loading returns one
// mock-backed read-only readiness entry per current provider without requiring
// real credentials, so provider clients can initialize safely in DEV.
func TestP000D001ProviderReadinessConfigDefaults(t *testing.T) {
	clearProductionAuthEnv(t)
	clearProviderReadinessEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	wantProviders := []string{
		provider.ProviderNameZoom,
		provider.ProviderNameGoogle,
		provider.ProviderNameAeries,
		provider.ProviderNameSFTP,
	}
	gotProviders := make([]string, 0, len(cfg.ProviderReadiness))
	for _, readiness := range cfg.ProviderReadiness {
		gotProviders = append(gotProviders, readiness.Provider)
		if !readiness.UseMock {
			t.Fatalf("%s readiness UseMock = false, want true", readiness.Provider)
		}
		if !readiness.ReadOnly {
			t.Fatalf("%s readiness ReadOnly = false, want true", readiness.Provider)
		}
	}
	if !slices.Equal(gotProviders, wantProviders) {
		t.Fatalf("provider readiness entries = %#v, want %#v", gotProviders, wantProviders)
	}
}

// TestP000D002ProviderReadinessConfigFailureSurfacing verifies config.Load
// carries enough sanitized provider metadata for /health/ready to fail closed
// when staging disables a mock flag with a missing credential label, malformed
// URL, or invalid certificate file.
func TestP000D002ProviderReadinessConfigFailureSurfacing(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		providerName string
		wantContains string
	}{
		{
			name: "missing credential label",
			env: map[string]string{
				"USE_MOCK_ZOOM": "false",
				"ZOOM_BASE_URL": "https://zoom.example.test/v2",
			},
			providerName: provider.ProviderNameZoom,
			wantContains: "ZOOM_ACCOUNT_ID",
		},
		{
			name: "bad url",
			env: map[string]string{
				"USE_MOCK_ZOOM":   "false",
				"ZOOM_ACCOUNT_ID": "zoom-staging-label",
				"ZOOM_BASE_URL":   "http://zoom.example.test/v2",
			},
			providerName: provider.ProviderNameZoom,
			wantContains: "ZOOM_BASE_URL must use https",
		},
		{
			name: "bad certificate",
			env: map[string]string{
				"USE_MOCK_AERIES":  "false",
				"AERIES_READ_ONLY": "true",
				"AERIES_BASE_URL":  "https://aeries.example.test/api",
				"AERIES_CLIENT_ID": "aeries-staging-label",
				"AERIES_CERT_FILE": writeConfigTempFile(t, "not a certificate"),
			},
			providerName: provider.ProviderNameAeries,
			wantContains: "AERIES_CERT_FILE must contain a PEM certificate",
		},
		{
			name: "missing sftp host",
			env: map[string]string{
				"USE_MOCK_SFTP": "false",
				"SFTP_USERNAME": "sftp-staging-label",
			},
			providerName: provider.ProviderNameSFTP,
			wantContains: "SFTP_HOST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearProductionAuthEnv(t)
			clearProviderReadinessEnv(t)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			got := provider.ConfigurationDiagnostics(cfg.ProviderReadiness)[tc.providerName]
			if provider.ConfigurationStatusReady(got) {
				t.Fatalf("expected blocked readiness diagnostic, got %q", got)
			}
			if !strings.Contains(got, tc.wantContains) {
				t.Fatalf("diagnostic %q does not contain %q", got, tc.wantContains)
			}
		})
	}
}

// TestP000C001DevAndStagingShareStaffDomainGate records the staging parity
// evidence available before live SAML assertion handling exists. Config loading
// keeps the same default allow/deny domain policy in development and staging,
// so both environments feed identical staff-domain rules to the production
// Google identity evaluator.
func TestP000C001DevAndStagingShareStaffDomainGate(t *testing.T) {
	for _, appEnv := range []string{"development", "staging"} {
		t.Run(appEnv, func(t *testing.T) {
			clearProductionAuthEnv(t)
			t.Setenv("APP_ENV", appEnv)

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg.AppEnv != appEnv {
				t.Fatalf("app env = %q, want %q", cfg.AppEnv, appEnv)
			}
			if !slices.Equal(cfg.ProductionAuthPolicy.AllowedEmailDomains, []string{"it.wusd.org", "staff.wusd.org", "wusd.org"}) {
				t.Fatalf("allowed domains = %#v", cfg.ProductionAuthPolicy.AllowedEmailDomains)
			}
			if !slices.Equal(cfg.ProductionAuthPolicy.DeniedEmailDomains, []string{"stu.wusd.org"}) {
				t.Fatalf("denied domains = %#v", cfg.ProductionAuthPolicy.DeniedEmailDomains)
			}
		})
	}
}

// TestLoadOverridesFromEnvironment verifies that startup keeps ordinary service
// settings and the production SAML/auth mapping contract configurable from the
// environment without embedding tenant-specific secrets in the repository.
func TestLoadOverridesFromEnvironment(t *testing.T) {
	clearProductionAuthEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("ZOOM_SLG_MAX_MEMBERS", "12")
	t.Setenv("REPLAY_EVENT_LIMIT", "55")
	t.Setenv("AUTH_ALLOWED_EMAIL_DOMAINS", "wusd.org,@staff.wusd.org")
	t.Setenv("AUTH_DENIED_EMAIL_DOMAINS", "@stu.wusd.org,blocked.example")
	t.Setenv("GOOGLE_SAML_ENTITY_ID", "wizard-prod")
	t.Setenv("GOOGLE_SAML_ACS_URL", "https://wizard.example.test/saml/acs")
	t.Setenv("GOOGLE_SAML_IDP_METADATA_URL", "https://accounts.google.com/o/saml2/idp?idpid=test")
	t.Setenv("GOOGLE_SAML_IDP_SSO_URL", "https://accounts.google.com/o/saml2/idp")
	t.Setenv("GOOGLE_SAML_IDP_CERT_FILE", "/run/secrets/google-saml-idp.crt")
	t.Setenv("GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON", `[{"group":"wizard-it-admins@wusd.org","roles":["it_admin"]}]`)
	t.Setenv("GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON", `[{"attribute":"wizard_role","values":["Human Resources"],"roles":["human_resources"]}]`)
	t.Setenv("GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON", `[{"source_type":"group","source":"wizard-bpl-scope@wusd.org","sites":["bpl"]}]`)
	t.Setenv("USE_MOCK_ZOOM", "false")
	t.Setenv("ZOOM_ACCOUNT_ID", "zoom-staging-label")
	t.Setenv("ZOOM_BASE_URL", "https://zoom.example.test/v2")
	t.Setenv("USE_MOCK_AERIES", "false")
	t.Setenv("AERIES_READ_ONLY", "true")
	t.Setenv("AERIES_DATABASE_YEAR_MODE", provider.AeriesDatabaseYearModePreviousSchoolYear)
	t.Setenv("AERIES_MASKED_PREVIOUS_YEAR_ONLY", "true")
	t.Setenv("AERIES_BASE_URL", "https://aeries.example.invalid/api")
	t.Setenv("AERIES_CLIENT_ID", "aeries-staging-label")
	t.Setenv("AERIES_CERT_FILE", "/run/secrets/aeries-client-cert.pem")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.AppEnv != "test" {
		t.Fatalf("expected app env test, got %q", cfg.AppEnv)
	}
	if cfg.AppPort != "9090" {
		t.Fatalf("expected app port 9090, got %q", cfg.AppPort)
	}
	if cfg.ZoomSLGMaxMembers != 12 {
		t.Fatalf("expected zoom limit 12, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 55 {
		t.Fatalf("expected replay limit 55, got %d", cfg.ReplayEventLimit)
	}
	if !slices.Equal(cfg.ProductionAuthPolicy.AllowedEmailDomains, []string{"staff.wusd.org", "wusd.org"}) {
		t.Fatalf("allowed domains = %#v", cfg.ProductionAuthPolicy.AllowedEmailDomains)
	}
	if !slices.Equal(cfg.ProductionAuthPolicy.DeniedEmailDomains, []string{"blocked.example", "stu.wusd.org"}) {
		t.Fatalf("denied domains = %#v", cfg.ProductionAuthPolicy.DeniedEmailDomains)
	}
	if cfg.ProductionAuthPolicy.SAML.EntityID != "wizard-prod" {
		t.Fatalf("SAML entity id = %q", cfg.ProductionAuthPolicy.SAML.EntityID)
	}
	if cfg.ProductionAuthPolicy.SAML.IDPCertFile != "/run/secrets/google-saml-idp.crt" {
		t.Fatalf("SAML cert file = %q", cfg.ProductionAuthPolicy.SAML.IDPCertFile)
	}
	if len(cfg.ProductionAuthPolicy.GroupRoleMappings) != 1 || cfg.ProductionAuthPolicy.GroupRoleMappings[0].Group != "wizard-it-admins@wusd.org" {
		t.Fatalf("group mappings = %#v", cfg.ProductionAuthPolicy.GroupRoleMappings)
	}
	if len(cfg.ProductionAuthPolicy.AttributeRoleMappings) != 1 || cfg.ProductionAuthPolicy.AttributeRoleMappings[0].Roles[0] != auth.RoleHumanResources {
		t.Fatalf("attribute mappings = %#v", cfg.ProductionAuthPolicy.AttributeRoleMappings)
	}
	if len(cfg.ProductionAuthPolicy.SiteScopeMappings) != 1 || cfg.ProductionAuthPolicy.SiteScopeMappings[0].Sites[0] != "bpl" {
		t.Fatalf("site scope mappings = %#v", cfg.ProductionAuthPolicy.SiteScopeMappings)
	}
	zoomReadiness := cfg.ProviderReadiness[0]
	if zoomReadiness.Provider != provider.ProviderNameZoom || zoomReadiness.UseMock || !zoomReadiness.ReadOnly {
		t.Fatalf("zoom readiness = %#v, want read-only non-mock config", zoomReadiness)
	}
	if zoomReadiness.Endpoint != "https://zoom.example.test/v2" || zoomReadiness.CredentialLabel != "zoom-staging-label" {
		t.Fatalf("zoom readiness metadata = %#v", zoomReadiness)
	}
	aeriesReadiness := cfg.ProviderReadiness[2]
	if aeriesReadiness.Provider != provider.ProviderNameAeries || aeriesReadiness.UseMock || !aeriesReadiness.ReadOnly {
		t.Fatalf("aeries readiness = %#v, want read-only non-mock config", aeriesReadiness)
	}
	if aeriesReadiness.Endpoint != "https://aeries.example.invalid/api" || aeriesReadiness.CredentialLabel != "aeries-staging-label" {
		t.Fatalf("aeries readiness metadata = %#v", aeriesReadiness)
	}
	if aeriesReadiness.DatabaseYearMode != provider.AeriesDatabaseYearModePreviousSchoolYear || !aeriesReadiness.MaskedPreviousYearOnly || !aeriesReadiness.CertificateFileConfigured {
		t.Fatalf("aeries previous-year safety flags = %#v", aeriesReadiness)
	}
}

// TestLoadKeepsMandatoryStudentDenyDomain records the Phase 0 dev and staging
// evidence for explicit student-domain denial before live SAML assertion
// handling exists. Deployment-specific denied domains can add local blocks, but
// neither development nor staging config can remove the documented
// @stu.wusd.org safety floor from the parsed production auth policy.
func TestLoadKeepsMandatoryStudentDenyDomain(t *testing.T) {
	for _, appEnv := range []string{"development", "staging"} {
		t.Run(appEnv, func(t *testing.T) {
			clearProductionAuthEnv(t)
			t.Setenv("APP_ENV", appEnv)
			t.Setenv("AUTH_ALLOWED_EMAIL_DOMAINS", "wusd.org,stu.wusd.org")
			t.Setenv("AUTH_DENIED_EMAIL_DOMAINS", "blocked.example")
			t.Setenv("GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON", `[{"group":"wizard-it-admins@wusd.org","roles":["it_admin"]}]`)

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg.AppEnv != appEnv {
				t.Fatalf("app env = %q, want %q", cfg.AppEnv, appEnv)
			}
			if !slices.Equal(cfg.ProductionAuthPolicy.DeniedEmailDomains, []string{"blocked.example", "stu.wusd.org"}) {
				t.Fatalf("denied domains = %#v, want blocked.example and mandatory stu.wusd.org", cfg.ProductionAuthPolicy.DeniedEmailDomains)
			}

			decision := auth.EvaluateGoogleIdentity(cfg.ProductionAuthPolicy, auth.GoogleIdentity{
				Email:  "otherwise.mapped@stu.wusd.org",
				Groups: []string{"wizard-it-admins@wusd.org"},
			})
			if decision.Authorized || decision.Reason != "denied_domain" {
				t.Fatalf("student decision = %#v, want denied_domain", decision)
			}
		})
	}
}

// TestLoadFallsBackOnInvalidIntegerValues confirms optional numeric tuning
// variables cannot prevent local startup when they contain non-integer values.
func TestLoadFallsBackOnInvalidIntegerValues(t *testing.T) {
	clearProductionAuthEnv(t)
	t.Setenv("ZOOM_SLG_MAX_MEMBERS", "bogus")
	t.Setenv("REPLAY_EVENT_LIMIT", "bogus")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ZoomSLGMaxMembers != 10 {
		t.Fatalf("expected zoom limit fallback 10, got %d", cfg.ZoomSLGMaxMembers)
	}
	if cfg.ReplayEventLimit != 100 {
		t.Fatalf("expected replay limit fallback 100, got %d", cfg.ReplayEventLimit)
	}
}

// TestLoadRejectsInvalidProductionAuthMappings fails closed for malformed
// Google role/site mapping JSON so production deployments cannot accidentally
// run with an incomplete authorization boundary.
func TestLoadRejectsInvalidProductionAuthMappings(t *testing.T) {
	clearProductionAuthEnv(t)
	t.Setenv("GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON", `[{"group":"wizard-it-admins@wusd.org","roles":[]}]`)

	if _, err := config.Load(); err == nil {
		t.Fatal("Load accepted invalid production auth mapping JSON")
	}
}

func clearProductionAuthEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"AUTH_ALLOWED_EMAIL_DOMAINS",
		"AUTH_DENIED_EMAIL_DOMAINS",
		"GOOGLE_SAML_ENTITY_ID",
		"GOOGLE_SAML_ACS_URL",
		"GOOGLE_SAML_IDP_METADATA_URL",
		"GOOGLE_SAML_IDP_SSO_URL",
		"GOOGLE_SAML_IDP_CERT_FILE",
		"GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON",
		"GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON",
		"GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON",
	} {
		t.Setenv(key, "")
	}
}

func clearProviderReadinessEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"USE_MOCK_ZOOM",
		"USE_MOCK_GOOGLE",
		"USE_MOCK_AERIES",
		"USE_MOCK_SFTP",
		"ZOOM_ACCOUNT_ID",
		"ZOOM_BASE_URL",
		"GOOGLE_REDIRECT_URL",
		"AERIES_BASE_URL",
		"AERIES_CLIENT_ID",
		"AERIES_READ_ONLY",
		"AERIES_DATABASE_YEAR_MODE",
		"AERIES_MASKED_PREVIOUS_YEAR_ONLY",
		"AERIES_CERT_FILE",
		"SFTP_HOST",
		"SFTP_USERNAME",
	} {
		t.Setenv(key, "")
	}
}

func writeConfigTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config-readiness-test.pem")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config readiness temp file: %v", err)
	}
	return path
}
