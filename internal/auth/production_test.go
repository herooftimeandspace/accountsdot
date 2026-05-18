package auth_test

import (
	"slices"
	"testing"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/auth"
)

func TestEvaluateGoogleIdentityAppliesDomainGateBeforeRoleMapping(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.GroupRoleMappings = []auth.GroupRoleMapping{{Group: "wizard-it-admins@wusd.org", Roles: []string{auth.RoleITAdmin}}}

	tests := []struct {
		name   string
		email  string
		reason string
	}{
		{name: "student domain is explicitly denied", email: "student@stu.wusd.org", reason: "denied_domain"},
		{name: "unknown domain is denied before group roles", email: "vendor@example.org", reason: "domain_not_allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
				Email:  tt.email,
				Groups: []string{"wizard-it-admins@wusd.org"},
			})
			if decision.Authorized {
				t.Fatalf("decision authorized %q; want denied", tt.email)
			}
			if decision.Reason != tt.reason {
				t.Fatalf("reason = %q, want %q", decision.Reason, tt.reason)
			}
		})
	}
}

func TestEvaluateGoogleIdentityMapsGroupsAttributesAndSites(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.GroupRoleMappings = []auth.GroupRoleMapping{
		{Group: "wizard-it-admins@wusd.org", Roles: []string{auth.RoleITAdmin}},
		{Group: "wizard-site-secretaries@wusd.org", Roles: []string{auth.RoleSiteSecretary}},
	}
	policy.AttributeRoleMappings = []auth.AttributeRoleMapping{
		{Attribute: "wizard_role", Values: []string{"Device Wrangler"}, Roles: []string{auth.RoleDeviceWrangler}},
	}
	policy.SiteScopeMappings = []auth.SiteScopeMapping{
		{SourceType: "group", Source: "wizard-bpl-scope@wusd.org", Sites: []string{"bpl"}},
		{SourceType: "attribute", Source: "wizard_site", Values: []string{"Clover HS"}, Sites: []string{"clover-hs"}},
	}

	decision := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email:  "Casey.Secretary@staff.wusd.org",
		Groups: []string{"wizard-site-secretaries@wusd.org", "wizard-bpl-scope@wusd.org"},
		Attributes: map[string][]string{
			"wizard_role": {"Device Wrangler"},
			"wizard_site": {"Clover HS"},
		},
	})

	if !decision.Authorized {
		t.Fatalf("decision denied: %#v", decision)
	}
	assertContains(t, decision.Roles, auth.RoleSiteSecretary)
	assertContains(t, decision.Roles, auth.RoleDeviceWrangler)
	assertContains(t, decision.SiteScopes, "bpl")
	assertContains(t, decision.SiteScopes, "clover-hs")
	if decision.Email != "casey.secretary@staff.wusd.org" {
		t.Fatalf("email = %q, want canonical lowercase address", decision.Email)
	}
}

func TestEvaluateGoogleIdentityUnionsCanonicalAttributeNames(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.AttributeRoleMappings = []auth.AttributeRoleMapping{
		{Attribute: "wizard_role", Values: []string{"Device Wrangler"}, Roles: []string{auth.RoleDeviceWrangler}},
		{Attribute: "wizard_role", Values: []string{"Site Secretary"}, Roles: []string{auth.RoleSiteSecretary}},
	}
	policy.SiteScopeMappings = []auth.SiteScopeMapping{
		{SourceType: "attribute", Source: "wizard_site", Values: []string{"Clover HS"}, Sites: []string{"clover-hs"}},
		{SourceType: "attribute", Source: "wizard_site", Values: []string{"BPL"}, Sites: []string{"bpl"}},
	}

	decision := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email: "mixed.claims@staff.wusd.org",
		Attributes: map[string][]string{
			"Wizard_Role": {"Device Wrangler"},
			"wizard_role": {"Site Secretary"},
			"Wizard_Site": {"Clover HS"},
			"wizard_site": {"BPL"},
		},
	})

	if !decision.Authorized {
		t.Fatalf("decision denied: %#v", decision)
	}
	assertContains(t, decision.Roles, auth.RoleDeviceWrangler)
	assertContains(t, decision.Roles, auth.RoleSiteSecretary)
	assertContains(t, decision.SiteScopes, "bpl")
	assertContains(t, decision.SiteScopes, "clover-hs")
}

func TestEvaluateGoogleIdentityReflectsChangedAssignments(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.GroupRoleMappings = []auth.GroupRoleMapping{
		{Group: "wizard-site-admins@wusd.org", Roles: []string{auth.RoleSiteAdmin}},
	}
	policy.SiteScopeMappings = []auth.SiteScopeMapping{
		{SourceType: "group", Source: "wizard-bpl-scope@wusd.org", Sites: []string{"bpl"}},
		{SourceType: "group", Source: "wizard-whs-scope@wusd.org", Sites: []string{"whs"}},
	}

	before := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email:  "admin@wusd.org",
		Groups: []string{"wizard-site-admins@wusd.org", "wizard-bpl-scope@wusd.org"},
	})
	after := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email:  "admin@wusd.org",
		Groups: []string{"wizard-site-admins@wusd.org", "wizard-whs-scope@wusd.org"},
	})

	if !before.Authorized || !after.Authorized {
		t.Fatalf("expected both decisions authorized, got before=%#v after=%#v", before, after)
	}
	if !slices.Equal(before.SiteScopes, []string{"bpl"}) {
		t.Fatalf("before scopes = %#v, want bpl", before.SiteScopes)
	}
	if !slices.Equal(after.SiteScopes, []string{"whs"}) {
		t.Fatalf("after scopes = %#v, want whs", after.SiteScopes)
	}
}

func TestEvaluateGoogleIdentityDeniesKnownUserWithoutRoleMapping(t *testing.T) {
	decision := auth.EvaluateGoogleIdentity(auth.DefaultPolicy(), auth.GoogleIdentity{
		Email: "staff.member@it.wusd.org",
	})
	if decision.Authorized {
		t.Fatalf("decision authorized without roles: %#v", decision)
	}
	if decision.Reason != "no_role_mapping" {
		t.Fatalf("reason = %q, want no_role_mapping", decision.Reason)
	}
}

func TestEvaluateGoogleIdentityAllowsBreakglassPastDomainGateOnly(t *testing.T) {
	policy := auth.DefaultPolicy()
	policy.GroupRoleMappings = []auth.GroupRoleMapping{{Group: "local-breakglass", Roles: []string{auth.RoleITAdmin}}}

	decision := auth.EvaluateGoogleIdentity(policy, auth.GoogleIdentity{
		Email:           "local.admin",
		Groups:          []string{"local-breakglass"},
		BreakglassLocal: true,
	})
	if !decision.Authorized {
		t.Fatalf("breakglass decision denied: %#v", decision)
	}
	assertContains(t, decision.Roles, auth.RoleITAdmin)
}

func TestParseMappingsValidateContracts(t *testing.T) {
	groupMappings, err := auth.ParseGroupRoleMappings(`[{"group":"Wizard-Admins@WUSD.org","roles":["it_admin","it_admin"]}]`)
	if err != nil {
		t.Fatalf("ParseGroupRoleMappings returned error: %v", err)
	}
	if groupMappings[0].Group != "wizard-admins@wusd.org" || !slices.Equal(groupMappings[0].Roles, []string{"it_admin"}) {
		t.Fatalf("unexpected group mapping normalization: %#v", groupMappings[0])
	}

	if _, err := auth.ParseAttributeRoleMappings(`[{"attribute":"wizard_role","values":[],"roles":["site_admin"]}]`); err == nil {
		t.Fatal("ParseAttributeRoleMappings accepted an empty value list")
	}
	if _, err := auth.ParseSiteScopeMappings(`[{"source_type":"attribute","source":"wizard_site","sites":["bpl"]}]`); err == nil {
		t.Fatal("ParseSiteScopeMappings accepted an attribute mapping with no values")
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	if !slices.Contains(values, expected) {
		t.Fatalf("%#v does not contain %q", values, expected)
	}
}
