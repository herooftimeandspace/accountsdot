package core

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

type PermissionRole string

const (
	PermissionRoleITAdmin        PermissionRole = "it_admin"
	PermissionRoleHumanResources PermissionRole = "human_resources"
	PermissionRoleSiteAdmin      PermissionRole = "site_admin"
	PermissionRoleSiteSecretary  PermissionRole = "site_secretary"
	PermissionRoleDeviceWrangler PermissionRole = "device_wrangler"
	PermissionRoleFacultyStaff   PermissionRole = "faculty_staff"
)

type PermissionScopeKind string

const (
	PermissionScopeDistrict PermissionScopeKind = "district"
	PermissionScopeSite     PermissionScopeKind = "site"
)

type PermissionGrantSource string

const (
	PermissionGrantSourceSAML             PermissionGrantSource = "saml"
	PermissionGrantSourceGoogleGroup      PermissionGrantSource = "google_group"
	PermissionGrantSourceGoogleAttribute  PermissionGrantSource = "google_attribute"
	PermissionGrantSourceManualGrant      PermissionGrantSource = "manual_grant"
	PermissionGrantSourceManualRevocation PermissionGrantSource = "manual_revocation"
	PermissionGrantSourceBreakglass       PermissionGrantSource = "breakglass"
)

type PermissionAssignmentEffect string

const (
	PermissionAssignmentGrant  PermissionAssignmentEffect = "grant"
	PermissionAssignmentRevoke PermissionAssignmentEffect = "revoke"
)

type PermissionScope struct {
	Kind   PermissionScopeKind
	SiteID string
}

type PermissionAssignment struct {
	SubjectID string
	Role      PermissionRole
	Scope     PermissionScope
	Source    PermissionGrantSource
	Effect    PermissionAssignmentEffect
	Reason    string
	StartsAt  time.Time
	ExpiresAt time.Time
}

type PermissionSubject struct {
	ID                     string
	Email                  string
	SAMLRoles              []PermissionRole
	GoogleGroupRoles       []PermissionRole
	GoogleAttributeRoles   []PermissionRole
	GoogleGroupSiteScopes  []string
	ManualAssignments      []PermissionAssignment
	Disabled               bool
	Revoked                bool
	Breakglass             bool
	BreakglassNetworkValid bool
}

type EffectivePermission struct {
	Role    PermissionRole
	Scope   PermissionScope
	Sources []PermissionGrantSource
	Denied  bool
	Reasons []string
}

type EffectivePermissionSet struct {
	SubjectID   string
	Email       string
	Authorized  bool
	Denied      bool
	Breakglass  bool
	Permissions []EffectivePermission
	Denials     []EffectivePermission
}

type PermissionChange struct {
	TargetSubjectID string
	Assignment      PermissionAssignment
}

type PermissionAuditEvent struct {
	ID              string
	ActorSubjectID  string
	TargetSubjectID string
	Assignment      PermissionAssignment
	Before          EffectivePermissionSet
	After           EffectivePermissionSet
	CreatedAt       time.Time
	Reason          string
}

var (
	ErrPermissionLockout           = errors.New("permission change would lock out IT administration or breakglass recovery")
	ErrPermissionUnknownSubject    = errors.New("permission change targets an unknown subject")
	ErrPermissionInvalidAssignment = errors.New("permission assignment is invalid")
)

// ResolveEffectivePermissions is the model-layer entrypoint for issue #186 tests and future IT Admin grant/revoke routes. It combines trusted SAML and Google identity signals with audited manual grants and revocations, returning the effective roles and site scopes that authorization code can enforce without reading UI-only state.
func ResolveEffectivePermissions(subject PermissionSubject, now time.Time) EffectivePermissionSet {
	result := EffectivePermissionSet{
		SubjectID:  subject.ID,
		Email:      subject.Email,
		Breakglass: subject.Breakglass && subject.BreakglassNetworkValid,
	}
	if (subject.Disabled || subject.Revoked) && !result.Breakglass {
		result.Denied = true
		return result
	}
	if !result.Breakglass && !staffEmailAllowed(subject.Email) {
		result.Denied = true
		return result
	}

	grants := map[string]EffectivePermission{}
	denials := map[string]EffectivePermission{}
	addGrant := func(role PermissionRole, scope PermissionScope, source PermissionGrantSource, reason string) {
		scope = canonicalPermissionScope(scope)
		if !role.Valid() || !scope.Valid() || !role.ScopeCompatible(scope) {
			return
		}
		key := permissionKey(role, scope)
		permission := grants[key]
		permission.Role = role
		permission.Scope = scope
		if !slices.Contains(permission.Sources, source) {
			permission.Sources = append(permission.Sources, source)
		}
		if reason != "" && !slices.Contains(permission.Reasons, reason) {
			permission.Reasons = append(permission.Reasons, reason)
		}
		grants[key] = permission
	}
	addDenial := func(role PermissionRole, scope PermissionScope, source PermissionGrantSource, reason string) {
		scope = canonicalPermissionScope(scope)
		if !role.Valid() || !scope.Valid() || !role.ScopeCompatible(scope) {
			return
		}
		key := permissionKey(role, scope)
		denial := denials[key]
		denial.Role = role
		denial.Scope = scope
		denial.Denied = true
		if !slices.Contains(denial.Sources, source) {
			denial.Sources = append(denial.Sources, source)
		}
		if reason != "" && !slices.Contains(denial.Reasons, reason) {
			denial.Reasons = append(denial.Reasons, reason)
		}
		denials[key] = denial
	}
	addSourceRole := func(role PermissionRole, source PermissionGrantSource, reason string) {
		if role.DistrictScoped() {
			addGrant(role, PermissionScope{Kind: PermissionScopeDistrict}, source, reason)
			return
		}
		for _, siteID := range subject.GoogleGroupSiteScopes {
			addGrant(role, PermissionScope{Kind: PermissionScopeSite, SiteID: siteID}, source, reason)
		}
	}

	if result.Breakglass {
		addGrant(PermissionRoleITAdmin, PermissionScope{Kind: PermissionScopeDistrict}, PermissionGrantSourceBreakglass, "local emergency account inside an allowed recovery network")
	}
	for _, role := range subject.SAMLRoles {
		addSourceRole(role, PermissionGrantSourceSAML, "SAML role assignment")
	}
	for _, role := range subject.GoogleGroupRoles {
		addSourceRole(role, PermissionGrantSourceGoogleGroup, "Google group role assignment")
	}
	for _, role := range subject.GoogleAttributeRoles {
		addSourceRole(role, PermissionGrantSourceGoogleAttribute, "Google attribute role assignment")
	}
	for _, assignment := range subject.ManualAssignments {
		if assignment.SubjectID == "" || assignment.SubjectID != subject.ID {
			continue
		}
		if !assignment.Active(now) {
			continue
		}
		switch assignment.Effect {
		case PermissionAssignmentRevoke:
			addDenial(assignment.Role, assignment.Scope, PermissionGrantSourceManualRevocation, assignment.Reason)
		case PermissionAssignmentGrant:
			addGrant(assignment.Role, assignment.Scope, PermissionGrantSourceManualGrant, assignment.Reason)
		}
	}
	for key, denial := range denials {
		delete(grants, key)
		result.Denials = append(result.Denials, denial)
	}

	for _, grant := range grants {
		sort.Slice(grant.Sources, func(i, j int) bool { return grant.Sources[i] < grant.Sources[j] })
		result.Permissions = append(result.Permissions, grant)
	}
	sortEffectivePermissions(result.Permissions)
	sortEffectivePermissions(result.Denials)
	result.Authorized = len(result.Permissions) > 0
	return result
}

// ValidatePermissionChangeForLockout projects a proposed grant or revocation across the known subject set before a future write-capable IT Admin API persists it. The function has no storage side effects; callers should block the mutation when it returns a lockout error and then record the rejected attempt in the permission audit flow.
func ValidatePermissionChangeForLockout(subjects []PermissionSubject, change PermissionChange, now time.Time) error {
	found := false
	projected := make([]PermissionSubject, len(subjects))
	for i, subject := range subjects {
		projected[i] = subject
		if subject.ID == change.TargetSubjectID {
			found = true
			assignment := change.Assignment
			assignment.SubjectID = change.TargetSubjectID
			if !assignment.Valid() {
				return ErrPermissionInvalidAssignment
			}
			projected[i].ManualAssignments = append(append([]PermissionAssignment{}, subject.ManualAssignments...), assignment)
		}
	}
	if !found {
		return ErrPermissionUnknownSubject
	}

	admins := 0
	recoveryPaths := 0
	for _, subject := range projected {
		effective := ResolveEffectivePermissions(subject, now)
		if !effective.Breakglass && effective.HasRole(PermissionRoleITAdmin) {
			admins++
		}
		if effective.Breakglass && effective.HasRole(PermissionRoleITAdmin) {
			recoveryPaths++
		}
	}
	if admins == 0 || recoveryPaths == 0 {
		return fmt.Errorf("%w: effective_admins=%d recovery_paths=%d", ErrPermissionLockout, admins, recoveryPaths)
	}
	return nil
}

// HasRole lets tests and future route guards check the resolved model without duplicating source-precedence rules. It reads only the calculated permission slice and returns true when a grant carries the requested role on a scope that is valid for that role.
func (set EffectivePermissionSet) HasRole(role PermissionRole) bool {
	for _, permission := range set.Permissions {
		if permission.Role == role && !permission.Denied && role.ScopeCompatible(permission.Scope) {
			return true
		}
	}
	return false
}

// Valid lets future persistence and route decoders reject unknown role keys before they can affect authorization. It is called by assignment validation and resolver helpers, and it returns false for any role not documented in the current permission model.
func (role PermissionRole) Valid() bool {
	switch role {
	case PermissionRoleITAdmin,
		PermissionRoleHumanResources,
		PermissionRoleSiteAdmin,
		PermissionRoleSiteSecretary,
		PermissionRoleDeviceWrangler,
		PermissionRoleFacultyStaff:
		return true
	default:
		return false
	}
}

// DistrictScoped identifies roles whose effective access is district-wide instead of tied to a site assignment. The resolver uses this distinction to avoid turning a site-scoped role signal into unintended cross-site access when no site scope is present.
func (role PermissionRole) DistrictScoped() bool {
	return role == PermissionRoleITAdmin || role == PermissionRoleHumanResources
}

// ScopeCompatible keeps district-wide and site-scoped roles from being mixed during validation and resolution. IT Admin and Human Resources are district-only, while the operational school roles must name a concrete site.
func (role PermissionRole) ScopeCompatible(scope PermissionScope) bool {
	if !role.Valid() || !scope.Valid() {
		return false
	}
	if role.DistrictScoped() {
		return scope.Kind == PermissionScopeDistrict
	}
	return scope.Kind == PermissionScopeSite
}

// Valid checks whether a permission scope is structurally usable before an assignment participates in effective access. District scopes must not carry a site id, and site scopes must name the site that future authorization code can compare against page or record scope.
func (scope PermissionScope) Valid() bool {
	switch scope.Kind {
	case PermissionScopeDistrict:
		return scope.SiteID == ""
	case PermissionScopeSite:
		return strings.TrimSpace(scope.SiteID) != ""
	default:
		return false
	}
}

// Active evaluates the assignment's effective window for the resolver and lockout projection. It treats zero timestamps as open-ended so permanent grants and revocations do not require placeholder dates.
func (assignment PermissionAssignment) Active(now time.Time) bool {
	if !assignment.StartsAt.IsZero() && assignment.StartsAt.After(now) {
		return false
	}
	if !assignment.ExpiresAt.IsZero() && !assignment.ExpiresAt.After(now) {
		return false
	}
	return true
}

// Valid checks the persisted assignment contract before a future write API accepts a proposed permission edit. It rejects subject-less rows, unknown roles, invalid scopes, role/scope mismatches, and effects other than grant or revoke.
func (assignment PermissionAssignment) Valid() bool {
	if strings.TrimSpace(assignment.SubjectID) == "" {
		return false
	}
	scope := canonicalPermissionScope(assignment.Scope)
	if !assignment.Role.Valid() || !scope.Valid() || !assignment.Role.ScopeCompatible(scope) {
		return false
	}
	return assignment.Effect == PermissionAssignmentGrant || assignment.Effect == PermissionAssignmentRevoke
}

// staffEmailAllowed enforces the documented staff-domain gate before ordinary SAML, Google, or manual assignments are considered. Breakglass callers bypass this helper only after the recovery-network check has already succeeded.
func staffEmailAllowed(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if strings.HasSuffix(normalized, "@stu.wusd.org") {
		return false
	}
	return strings.HasSuffix(normalized, "@wusd.org") ||
		strings.HasSuffix(normalized, "@it.wusd.org") ||
		strings.HasSuffix(normalized, "@staff.wusd.org")
}

// canonicalPermissionScope normalizes internal site identifiers before permissions are merged. The resolver keeps operator-facing source values elsewhere, but permission identity must not split grants and revocations because of casing or incidental whitespace.
func canonicalPermissionScope(scope PermissionScope) PermissionScope {
	if scope.Kind != PermissionScopeSite {
		return scope
	}
	scope.SiteID = strings.ToLower(strings.TrimSpace(scope.SiteID))
	return scope
}

// permissionKey creates the deterministic identity used for grant/revocation merging and audit snapshots. It keeps role, scope kind, and canonical site id together so revoking one site-scoped permission does not affect another site.
func permissionKey(role PermissionRole, scope PermissionScope) string {
	scope = canonicalPermissionScope(scope)
	return string(role) + "|" + string(scope.Kind) + "|" + scope.SiteID
}

// sortEffectivePermissions gives tests, logs, and future audit events stable output ordering. It only reorders the in-memory resolver result and has no persistence side effects.
func sortEffectivePermissions(permissions []EffectivePermission) {
	sort.Slice(permissions, func(i, j int) bool {
		left := permissionKey(permissions[i].Role, permissions[i].Scope)
		right := permissionKey(permissions[j].Role, permissions[j].Scope)
		return left < right
	})
}
