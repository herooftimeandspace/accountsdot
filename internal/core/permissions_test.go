package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

func TestResolveEffectivePermissionsCombinesTrustedSourcesAndManualGrants(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:                    "casey",
		Email:                 "casey.nguyen@wusd.org",
		SAMLRoles:             []core.PermissionRole{core.PermissionRoleFacultyStaff},
		GoogleGroupRoles:      []core.PermissionRole{core.PermissionRoleSiteSecretary},
		GoogleGroupSiteScopes: []string{"windsor-high"},
		ManualAssignments: []core.PermissionAssignment{
			{
				SubjectID: "casey",
				Role:      core.PermissionRoleDeviceWrangler,
				Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "windsor-high"},
				Effect:    core.PermissionAssignmentGrant,
				Reason:    "IT Admin approved temporary device accountability access",
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if !effective.Authorized || effective.Denied {
		t.Fatalf("expected authorized subject, got authorized=%v denied=%v", effective.Authorized, effective.Denied)
	}
	assertPermission(t, effective, core.PermissionRoleFacultyStaff, "windsor-high")
	assertPermission(t, effective, core.PermissionRoleSiteSecretary, "windsor-high")
	assertPermission(t, effective, core.PermissionRoleDeviceWrangler, "windsor-high")
}

func TestResolveEffectivePermissionsBlocksCrossSiteLeakageWithoutScope(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:               "morgan",
		Email:            "morgan.lee@wusd.org",
		GoogleGroupRoles: []core.PermissionRole{core.PermissionRoleSiteAdmin},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.Authorized || len(effective.Permissions) != 0 {
		t.Fatalf("expected no effective site permission without a site scope, got %#v", effective.Permissions)
	}
}

func TestResolveEffectivePermissionsIgnoresStaleManualGrant(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:    "riley",
		Email: "riley.patel@wusd.org",
		ManualAssignments: []core.PermissionAssignment{
			{
				SubjectID: "riley",
				Role:      core.PermissionRoleSiteAdmin,
				Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "mattiemay"},
				Effect:    core.PermissionAssignmentGrant,
				ExpiresAt: now.Add(-time.Minute),
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.Authorized || len(effective.Permissions) != 0 {
		t.Fatalf("expected expired grant to be ignored, got %#v", effective.Permissions)
	}
}

func TestResolveEffectivePermissionsManualRevocationOverridesGroupGrant(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:                    "avery",
		Email:                 "avery.chen@staff.wusd.org",
		GoogleGroupRoles:      []core.PermissionRole{core.PermissionRoleSiteAdmin},
		GoogleGroupSiteScopes: []string{"brooks"},
		ManualAssignments: []core.PermissionAssignment{
			{
				SubjectID: "avery",
				Role:      core.PermissionRoleSiteAdmin,
				Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "brooks"},
				Effect:    core.PermissionAssignmentRevoke,
				Reason:    "temporary suspension pending Google group cleanup",
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.HasRole(core.PermissionRoleSiteAdmin) {
		t.Fatalf("expected manual revocation to remove site admin permission, got %#v", effective.Permissions)
	}
	if len(effective.Denials) != 1 || !effective.Denials[0].Denied {
		t.Fatalf("expected denial audit surface for manual revocation, got %#v", effective.Denials)
	}
}

func TestResolveEffectivePermissionsIgnoresSubjectlessManualAssignment(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:    "jordan",
		Email: "jordan.rivera@wusd.org",
		ManualAssignments: []core.PermissionAssignment{
			{
				Role:   core.PermissionRoleSiteAdmin,
				Scope:  core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "windsor-high"},
				Effect: core.PermissionAssignmentGrant,
				Reason: "malformed row missing subject id",
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.Authorized || effective.HasRole(core.PermissionRoleSiteAdmin) {
		t.Fatalf("expected subject-less manual grant to fail closed, got %#v", effective.Permissions)
	}
}

func TestResolveEffectivePermissionsIgnoresUnknownManualAssignmentEffect(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:    "taylor",
		Email: "taylor.morgan@wusd.org",
		ManualAssignments: []core.PermissionAssignment{
			{
				SubjectID: "taylor",
				Role:      core.PermissionRoleSiteSecretary,
				Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "brooks"},
				Effect:    core.PermissionAssignmentEffect("approve"),
				Reason:    "bad migration value must not grant access",
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.Authorized || effective.HasRole(core.PermissionRoleSiteSecretary) {
		t.Fatalf("expected unknown manual effect to fail closed, got %#v", effective.Permissions)
	}
}

func TestResolveEffectivePermissionsCanonicalizesSiteIDsForRevocations(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subject := core.PermissionSubject{
		ID:                    "sam",
		Email:                 "sam.owens@staff.wusd.org",
		GoogleGroupRoles:      []core.PermissionRole{core.PermissionRoleSiteAdmin},
		GoogleGroupSiteScopes: []string{" Windsor-High "},
		ManualAssignments: []core.PermissionAssignment{
			{
				SubjectID: "sam",
				Role:      core.PermissionRoleSiteAdmin,
				Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "windsor-high"},
				Effect:    core.PermissionAssignmentRevoke,
				Reason:    "canonical revocation should match Google scope",
			},
		},
	}

	effective := core.ResolveEffectivePermissions(subject, now)

	if effective.HasRole(core.PermissionRoleSiteAdmin) {
		t.Fatalf("expected canonical revocation to remove matching grant, got %#v", effective.Permissions)
	}
	if len(effective.Denials) != 1 || effective.Denials[0].Scope.SiteID != "windsor-high" {
		t.Fatalf("expected canonical denial scope, got %#v", effective.Denials)
	}
}

func TestResolveEffectivePermissionsDeniesRevokedOrStudentDomainUsers(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	cases := []core.PermissionSubject{
		{
			ID:        "disabled",
			Email:     "disabled.admin@wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
			Disabled:  true,
		},
		{
			ID:        "revoked",
			Email:     "revoked.admin@wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
			Revoked:   true,
		},
		{
			ID:        "student",
			Email:     "student@stu.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleFacultyStaff},
		},
	}

	for _, subject := range cases {
		effective := core.ResolveEffectivePermissions(subject, now)
		if !effective.Denied || effective.Authorized {
			t.Fatalf("%s: expected denied unauthorized subject, got authorized=%v denied=%v", subject.ID, effective.Authorized, effective.Denied)
		}
	}
}

func TestPermissionAssignmentValidEnforcesRoleScopeCompatibility(t *testing.T) {
	validSiteGrant := core.PermissionAssignment{
		SubjectID: "valid-site",
		Role:      core.PermissionRoleSiteSecretary,
		Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "brooks"},
		Effect:    core.PermissionAssignmentGrant,
	}
	invalidSiteScopedITAdmin := core.PermissionAssignment{
		SubjectID: "invalid-admin",
		Role:      core.PermissionRoleITAdmin,
		Scope:     core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "brooks"},
		Effect:    core.PermissionAssignmentGrant,
	}
	invalidDistrictScopedSiteRole := core.PermissionAssignment{
		SubjectID: "invalid-site",
		Role:      core.PermissionRoleDeviceWrangler,
		Scope:     core.PermissionScope{Kind: core.PermissionScopeDistrict},
		Effect:    core.PermissionAssignmentGrant,
	}

	if !validSiteGrant.Valid() {
		t.Fatalf("expected compatible site assignment to be valid")
	}
	if invalidSiteScopedITAdmin.Valid() {
		t.Fatalf("expected site-scoped IT Admin assignment to be invalid")
	}
	if invalidDistrictScopedSiteRole.Valid() {
		t.Fatalf("expected district-scoped site role assignment to be invalid")
	}
}

func TestValidatePermissionChangeForLockoutPreventsRemovingLastEffectiveAdmin(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subjects := []core.PermissionSubject{
		{
			ID:                    "only-admin",
			Email:                 "only.admin@it.wusd.org",
			SAMLRoles:             []core.PermissionRole{core.PermissionRoleITAdmin},
			GoogleGroupSiteScopes: []string{"district"},
		},
		{
			ID:                     "recovery",
			Email:                  "breakglass.local",
			Breakglass:             true,
			BreakglassNetworkValid: true,
		},
	}
	change := core.PermissionChange{
		TargetSubjectID: "only-admin",
		Assignment: core.PermissionAssignment{
			Role:   core.PermissionRoleITAdmin,
			Scope:  core.PermissionScope{Kind: core.PermissionScopeDistrict},
			Effect: core.PermissionAssignmentRevoke,
			Reason: "attempted self-lockout",
		},
	}

	err := core.ValidatePermissionChangeForLockout(subjects, change, now)

	if !errors.Is(err, core.ErrPermissionLockout) {
		t.Fatalf("expected lockout error, got %v", err)
	}
}

func TestValidatePermissionChangeForLockoutRejectsInvalidSiteScopedITAdmin(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subjects := []core.PermissionSubject{
		{
			ID:        "target-admin",
			Email:     "target.admin@it.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
		},
		{
			ID:                     "recovery",
			Email:                  "breakglass.local",
			Breakglass:             true,
			BreakglassNetworkValid: true,
		},
	}
	change := core.PermissionChange{
		TargetSubjectID: "target-admin",
		Assignment: core.PermissionAssignment{
			Role:   core.PermissionRoleITAdmin,
			Scope:  core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "windsor-high"},
			Effect: core.PermissionAssignmentGrant,
			Reason: "invalid attempt to preserve admin through a site scope",
		},
	}

	err := core.ValidatePermissionChangeForLockout(subjects, change, now)

	if !errors.Is(err, core.ErrPermissionInvalidAssignment) {
		t.Fatalf("expected invalid assignment error, got %v", err)
	}
}

func TestValidatePermissionChangeForLockoutRequiresBreakglassRecovery(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subjects := []core.PermissionSubject{
		{
			ID:        "admin-one",
			Email:     "admin.one@it.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
		},
		{
			ID:        "admin-two",
			Email:     "admin.two@it.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
		},
	}
	change := core.PermissionChange{
		TargetSubjectID: "admin-one",
		Assignment: core.PermissionAssignment{
			Role:   core.PermissionRoleSiteAdmin,
			Scope:  core.PermissionScope{Kind: core.PermissionScopeSite, SiteID: "windsor-high"},
			Effect: core.PermissionAssignmentGrant,
			Reason: "ordinary site coverage edit still requires recovery path validation",
		},
	}

	err := core.ValidatePermissionChangeForLockout(subjects, change, now)

	if !errors.Is(err, core.ErrPermissionLockout) {
		t.Fatalf("expected missing recovery path to block permission writes, got %v", err)
	}
}

func TestValidatePermissionChangeForLockoutAllowsAdminRevocationWithAnotherAdminAndRecovery(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	subjects := []core.PermissionSubject{
		{
			ID:        "target-admin",
			Email:     "target.admin@it.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
		},
		{
			ID:        "remaining-admin",
			Email:     "remaining.admin@it.wusd.org",
			SAMLRoles: []core.PermissionRole{core.PermissionRoleITAdmin},
		},
		{
			ID:                     "recovery",
			Email:                  "breakglass.local",
			Breakglass:             true,
			BreakglassNetworkValid: true,
		},
	}
	change := core.PermissionChange{
		TargetSubjectID: "target-admin",
		Assignment: core.PermissionAssignment{
			Role:   core.PermissionRoleITAdmin,
			Scope:  core.PermissionScope{Kind: core.PermissionScopeDistrict},
			Effect: core.PermissionAssignmentRevoke,
			Reason: "approved admin rotation",
		},
	}

	if err := core.ValidatePermissionChangeForLockout(subjects, change, now); err != nil {
		t.Fatalf("expected safe admin rotation, got %v", err)
	}
}

func assertPermission(t *testing.T, set core.EffectivePermissionSet, role core.PermissionRole, siteID string) {
	t.Helper()
	for _, permission := range set.Permissions {
		if permission.Role == role && permission.Scope.Kind == core.PermissionScopeSite && permission.Scope.SiteID == siteID {
			return
		}
	}
	t.Fatalf("missing %s permission for %s in %#v", role, siteID, set.Permissions)
}
