package config_test

import (
	"slices"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/auth"
	"github.com/herooftimeandspace/go-employee-provisioner/internal/config"
)

// TestLoadDefaults verifies that local startup receives the documented default
// ports, limits, and production-auth domain gate without requiring any Google
// SAML secrets or admin-provided mapping JSON.
func TestLoadDefaults(t *testing.T) {
	clearProductionAuthEnv(t)

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
