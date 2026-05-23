# Editable Permissions Model

This document defines the model slice for GitHub issue #186. It is the durable bridge between the existing DEV persona/feature-flag scaffolding and the future IT Admin grant/revoke UI requested by #160.

The model intentionally separates three concepts:

- **Proposed access changes** are IT Admin edits that have been requested, validated, and audited, but may still need persistence, UI review, API authorization, or upstream Google cleanup.
- **Stored assignments** are grants and revocations persisted by the application after lockout validation succeeds.
- **Effective access** is the resolved permission set used by server-side authorization. Future route handlers must enforce effective access, not frontend state or proposed-only edits.

Permission grants, revocations, and site-scope mappings are one field family in the broader source-precedence contract in `docs/product/source-precedence.md`. This permissions model remains authoritative for the exact effective-access order, while the source-precedence document defines the shared reconciliation metadata, audit behavior, and UI presentation used when Google/SAML source signals disagree with dashboard-managed permission decisions.

## Source Inputs

Effective access is resolved from these inputs, in this order:

1. **Domain gate and account status**
   - Staff domains allowed: `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org`.
   - Student domain denied: `@stu.wusd.org`.
   - Disabled or revoked subjects receive no effective permissions unless they are an active breakglass subject.
   - Local breakglass accounts are exempt from the email-domain gate only when the configured recovery-network check succeeds.
2. **Breakglass recovery**
   - Breakglass grants an emergency `it_admin` district permission.
   - Breakglass assignment is read-only in the editable permissions UI.
   - Lockout protection must preserve at least one active breakglass recovery path.
3. **SAML identity**
   - SAML remains a canonical identity source for production authorization.
   - SAML role signals are read-only in v1. The dashboard may display them as grant sources but must not edit them directly.
4. **Google group or attribute assignment**
   - Google groups and attributes remain canonical production authorization sources where configured.
   - Google group or attribute role signals are read-only in v1.
   - Google group site-scope signals provide site applicability for site-scoped roles.
5. **Manual application grants**
   - Manual grants are editable by IT Admin in v1 for app-local roles and site scopes.
   - Manual grants can add a site-scoped permission or an approved district-scoped permission where the role is editable.
   - Manual grants may have a start time, expiration time, reason, actor, and audit entry.
6. **Manual application revocations**
   - Manual revocations are editable by IT Admin in v1 for app-local roles and site scopes.
   - Manual revocations deny the matching effective permission even when a SAML or Google source still reports the same role/scope. This is intentionally conservative for the in-app resolver and must be surfaced to IT as requiring upstream cleanup when the revoked source is still present.
   - Revocations may have a start time, expiration time, reason, actor, related ticket/form/workflow reference when available, review or expiration date for temporary revocations, supersession behavior, and audit entry.
7. **Feature rollout state**
   - Feature flags are not permission grants.
   - A user can hold a role and still be denied a route when the route's feature rollout is disabled for that persona/site.
   - IT Admin remains a documented read-only override for route-level feature flags and must not be stored as a normal editable target row.

## Persistent Data Structures

The narrow model implemented in `internal/core/permissions.go` uses these domain structures. Future persistence should map to tables or repository interfaces with the same semantics rather than storing UI-specific payloads.

- `PermissionSubject`
  - Stable subject id, email, disabled/revoked status, SAML roles, Google roles, Google site scopes, manual assignments, and breakglass state.
  - The production adapter should populate this from authenticated identity claims, Google directory/group data, local assignment rows, and breakglass configuration.
- `PermissionAssignment`
  - Subject id, role, scope, source, effect, reason, start time, and expiration time.
  - `Effect=grant` adds an editable local permission.
  - `Effect=revoke` denies the matching role/scope.
- `PermissionScope`
  - `district` for district-wide roles such as IT Admin and Human Resources.
  - `site` with a non-empty site id for site-scoped roles.
- `EffectivePermissionSet`
  - The resolver output used by future authorization checks.
  - Includes active grants, denials, the authorized flag, and whether the subject is using breakglass access.
- `PermissionAuditEvent`
  - Actor, target subject, assignment, before snapshot, after snapshot, timestamp, and reason.
  - This is the audit contract for future storage/API work. Persistence should write an audit event for successful changes and should also record rejected lockout attempts in the eventual permission-management audit trail.

Recommended future table boundaries:

- `permission_assignments`: local manual grants and revocations, including actor, reason, effective window, status, and soft-delete metadata.
- `permission_audit_events`: immutable before/after records for grant, revoke, update, expiration, soft-delete, and rejected lockout attempts.
- `site_scope_assignments`: explicit manual site scope rows if site scope needs independent lifecycle controls from role assignment rows.
- `permission_subject_snapshots`: optional debug snapshots if production troubleshooting needs to compare SAML/Google/manual sources without retaining raw credential-bearing payloads.

Do not persist secrets, SAML assertions, OAuth tokens, service-account JSON, or raw provider responses in these tables.

## Effective-Permission Rules

- A subject must pass the staff domain gate before normal roles are considered.
- Breakglass bypasses the domain gate only for named local emergency accounts and only from allowed recovery networks.
- Disabled or revoked subjects receive no permissions unless breakglass recovery is active.
- District-scoped roles resolve to district scope.
- Site-scoped roles require an explicit site scope. A site-scoped role without a site scope grants no effective site access, preventing cross-site leakage.
- `site_admin`, `site_secretary`, and `device_wrangler` are exactly-one-site operational roles. If SAML, Google group/attribute input, manual grants, or future editable scope rows resolve one of these roles to more than one active site after active revocations are applied, the resolver fails closed for that role and surfaces denial reasons for IT Admin cleanup instead of granting multi-site visibility.
- `faculty_staff` is not an operational site role. Faculty and Staff may be associated with more than one site, and the onboarding/current-assignment-derived default site is only an initial staff context for allowed self-service and directory surfaces. Multi-site Faculty/Staff association must not imply Site Admin, Site Secretary, Device Wrangler, Human Resources, or IT Admin permissions.
- Room Moves uses two checks for Site Admin and Site Secretary users: assigned-site scope controls which rows and drafts they can see, and author ownership controls whether they can save, apply, schedule, cancel, or delete the visible draft. IT Admin keeps district-wide Room Moves management authority.
- Manual grants are ignored before `StartsAt` and after `ExpiresAt`.
- Manual revocations are ignored before `StartsAt` and after `ExpiresAt`.
- Manual revocations remove the matching role/scope from the effective grant set and remain visible in the denial set for audit and operator explanation.
- When a manual revocation or site-scope mapping remains effective while Google or SAML still reports the old grant, the UI must surface `Upstream cleanup required` rather than implying the source system was changed.
- When Google/SAML source changes match a temporary manual revocation or site-scope correction, reconciliation may clear the temporary manual row only through an audited `clear_temporary_override` outcome.
- When Google/SAML source changes conflict with an active manual grant, revocation, or site mapping, reconciliation must preserve the last safe effective access decision, show the conflict to IT Admin, and fail closed for any site-scoped role that would otherwise become multi-site.
- The resolver returns deterministic ordering so tests, logs, and audit snapshots are stable.

## Lockout Protection

Before any future storage write commits, the service must project the proposed change across the known active subject set and reject it when either condition would become false:

- At least one non-breakglass subject still has effective `it_admin` district access.
- At least one breakglass subject still has effective `it_admin` district recovery access.

The model exposes `ValidatePermissionChangeForLockout` for this projection. Future write APIs should call it inside the same transaction that persists the assignment and audit event so concurrent admin edits cannot remove both recovery paths.

## V1 Editability

Editable in v1:

- Manual grants for site-scoped roles.
- Manual revocations for site-scoped roles.
- Manual site-scope assignments and removals.
- Human-readable reason text for the local assignment.
- Effective window for temporary access or temporary revocation.

Read-only in v1:

- SAML role claims.
- Google group memberships.
- Google attribute assignments.
- Breakglass account configuration.
- IT Admin feature-flag override.
- Route/API authorization policy.

Future-phase:

- Direct Google group membership mutation from this dashboard.
- Direct SAML attribute mutation from this dashboard.
- Audit-history restore or rollback.
- Profile-level provider entitlement editing beyond the provisioning-profile model already documented in `docs/planning/implementation-plan.md`.

## Remaining Work For #160

- Design and implement database migrations or repository interfaces for assignments and audit events.
- Add IT Admin-only route and API handlers guarded by server-side authorization.
- Add UI for viewing effective permissions, grant sources, denials, and audit history.
- Add transaction-level lockout validation around create/update/delete flows.
- Decide whether rejected lockout attempts are stored in the same audit table or a separate security event table.
- Decide how frequently Google/SAML source snapshots are refreshed and how stale-source warnings appear to IT Admin.
