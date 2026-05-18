# Permissions Matrix

This document records both the production authorization contract and the currently implemented DEV authorization behavior for The WIZARD. It does not replace the editable permissions model or breakglass runtime work. It gives reviewers a durable baseline for the Google SAML and Google group/attribute contract, the DEV persona-switcher behavior, and the route/API boundaries that must stay aligned with `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, and `TEST_MATRIX.md`.

This matrix is current DEV implementation documentation. Issue #185 supplies the route/API inventory evidence that issue #158 needed before the parent permissions backlog can be evaluated; that detailed audit lives in `docs/route-api-authorization-inventory.md` and is checked by `npm run route-api-inventory:check`. Issue #158 should close only when the parent issue owner confirms the remaining parent acceptance criteria beyond this inventory are complete. Issue #160 is intentionally out of scope for this matrix because editable in-app persona grant/revoke management requires a separate persistent authorization model rather than a richer DEV persona switcher. Issue #188 remains open for live production SAML assertion validation, production session issuance, approved Google Workspace metadata/group/attribute decisions, and persistent manual site-scope administration.

## Source Order

- `PRODUCT_REQUIREMENTS.md` defines the staff-only product boundary, allowed domains, explicit student denial, same-URL access-denied requirement, lifecycle visibility rules, and HR/IT offboarding action requirements.
- `IMPLEMENTATION_PLAN.md` defines the implementation contract, staged rollout constraints, Phase 2 live-write pilot gate, route registry, and follow-up work still required before production SAML is live.
- `docs/route-api-authorization-inventory.md` records the route-by-route frontend, DEV API, direct-navigation, feature-flag, and static-page exception audit for issue #185.
- `internal/auth/production.go` contains the current checked-in evaluator for verified Google identity data.
- `internal/web` contains the DEV persona-switcher route and API authorization behavior.
- `TEST_MATRIX.md` defines the scenarios that must be evidenced in dev and staging.

## Domain Gate

- The dashboard is staff-only.
- Allowed staff domains are `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org`.
- `@stu.wusd.org` is explicitly denied before role authorization.
- Local breakglass accounts are the only domain-gate exception.

## Production Auth Flow

1. Google Workspace authenticates the user through SAML.
2. The application receives verified identity data from the future SAML middleware: email address, group memberships, and configured SAML attributes.
3. The application canonicalizes the email address and applies the domain gate before any normal role authorization.
4. The application denies `@stu.wusd.org` even if Google groups or attributes would otherwise map to an application role.
5. The application allows only `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org` for normal SAML users.
6. The breakglass runtime may bypass the domain gate only for named local emergency accounts after its own local-auth and network-source checks pass.
7. The application maps Google groups and SAML attributes to stable role ids.
8. A user with no mapped role is authenticated but not authorized and must receive access denied.
9. The application maps current Google groups and SAML attributes to site scopes on each authorization evaluation so changed assignments do not leave stale cross-site access.
10. Route and API handlers must enforce the resulting role and scope server-side. Frontend hiding is not a production authorization control.

## Stable Role IDs

| Role ID | Product Role | Scope Expectation |
| --- | --- | --- |
| `it_admin` | IT Admin | District-wide by default. |
| `human_resources` | Human Resources | District-wide lifecycle visibility for HR-owned workflows. |
| `site_admin` | Administrative Staff / Site Admin Staff | Site-scoped by default, with multi-site expansion only from approved Google group or attribute mappings. |
| `site_secretary` | Site Secretary | Site-scoped student cleanup and room-move participation. |
| `device_wrangler` | Device Wrangler | Site-scoped student device-accountability reporting. |
| `faculty_staff` | Faculty and Staff | Limited self-service only. |

## Configuration Contract

The checked-in environment contract is:

- `AUTH_ALLOWED_EMAIL_DOMAINS`: comma-separated normal SAML domains. Default: `wusd.org,it.wusd.org,staff.wusd.org`.
- `AUTH_DENIED_EMAIL_DOMAINS`: comma-separated explicit denied domains. Default: `stu.wusd.org`.
- `GOOGLE_SAML_ENTITY_ID`: service-provider entity id configured in Google Workspace.
- `GOOGLE_SAML_ACS_URL`: assertion consumer service URL configured in Google Workspace.
- `GOOGLE_SAML_IDP_METADATA_URL`: Google-hosted metadata URL when the deployment uses metadata discovery.
- `GOOGLE_SAML_IDP_SSO_URL`: Google IdP sign-in URL when metadata discovery is not used.
- `GOOGLE_SAML_IDP_CERT_FILE`: path to the deployment-managed IdP certificate file. Do not commit certificate material.
- `GOOGLE_AUTH_GROUP_ROLE_MAPPINGS_JSON`: JSON array of group-to-role mappings.
- `GOOGLE_AUTH_ATTRIBUTE_ROLE_MAPPINGS_JSON`: JSON array of SAML-attribute-to-role mappings.
- `GOOGLE_AUTH_SITE_SCOPE_MAPPINGS_JSON`: JSON array of group or SAML-attribute-to-site-scope mappings.
- `BREAKGLASS_ACCOUNTS`: comma-separated local emergency account ids.
- `BREAKGLASS_TOKEN_SHA256_<SANITIZED_ACCOUNT_ID>`: per-account SHA-256 token hash where non-alphanumeric account-id characters are converted to underscores and uppercased.
- `BREAKGLASS_ALLOWED_CIDRS`: optional comma-separated allowed source networks. Default: `10.23.0.0/16,10.19.100.0/24`.

Example mapping shape:

```json
[
  {
    "group": "wizard-it-admins@wusd.org",
    "roles": ["it_admin"]
  }
]
```

```json
[
  {
    "attribute": "wizard_role",
    "values": ["Site Secretary"],
    "roles": ["site_secretary"]
  }
]
```

```json
[
  {
    "source_type": "group",
    "source": "wizard-bpl-scope@wusd.org",
    "sites": ["bpl"]
  },
  {
    "source_type": "attribute",
    "source": "wizard_site",
    "values": ["Clover HS"],
    "sites": ["clover-hs"]
  }
]
```

## DEV Source Rules

- The dashboard is staff-only. Allowed staff domains are `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org`; `@stu.wusd.org` is denied before normal role authorization. Local breakglass accounts are the only documented domain-gate exception.
- Unauthenticated users receive `401` for protected DEV page and API routes.
- Authenticated users without route, persona, feature-flag, site-scope, or field-level permission receive `403` or the route's normal access-denied behavior.
- Sidebar hiding and disabled controls are defense-in-depth only. Server-side DEV APIs must enforce authorization before returning protected data or mutating DEV state.
- IT Admin has all current implemented routes and an override for route-level feature flags. Human Resources has district-wide lifecycle access. Site Admin, Site Secretary, Device Wrangler, and Faculty and Staff receive only the route set and site/field visibility listed below.
- Local breakglass access is implemented as a separate local emergency login route, not as a selectable DEV persona. The current DEV persona switcher remains only a mock-session convenience for implemented staff roles.
- Production authorization remains future work beyond this DEV persona model. The durable target is SAML identity plus Google group or attribute-based authorization, with persistent site-scope mapping where appropriate. The current DEV session payloads, route lists, feature flags, and mock site scopes are implementation scaffolding for route/API behavior, not a production SAML or Google-group integration.

## DEV Persona Route Matrix

| Persona | Implemented route access | Scope | Current status |
| --- | --- | --- | --- |
| IT Admin | All implemented routes: `/dashboard/it-admin`, `/dashboard/hr-lifecycle`, `/dashboard/site-admin`, `/search`, `/onboarding`, `/offboarding`, `/departing-seniors`, `/room-moves`, `/room-moves/bulk-draft`, phone-directory routes, `/data-quality`, `/frequent-fliers`, `/student-data-cleanup`, `/reports`, `/reports/security-issues`, `/reports/sync-transparency`, `/admin`, `/admin/feature-flags`, `/my-profile` | District-wide | Implemented |
| Human Resources | `/dashboard/hr-lifecycle`, `/search`, `/phone-directory/by-person`, `/phone-directory/by-room`, `/phone-directory/by-department`, `/my-profile`, `/onboarding`, `/offboarding` | District-wide | Implemented |
| Site Admin | `/dashboard/site-admin`, `/search`, phone-directory routes, `/my-profile`, `/student-data-cleanup`, `/frequent-fliers`, `/onboarding`, `/offboarding`, `/room-moves`, `/room-moves/bulk-draft` | Assigned sites only on site-scoped pages | Implemented |
| Site Secretary | `/search`, phone-directory routes, `/my-profile`, `/student-data-cleanup`, `/room-moves`, `/room-moves/bulk-draft` | Assigned sites only | Implemented |
| Device Wrangler | `/search`, phone-directory routes, `/my-profile`, `/frequent-fliers`, `/departing-seniors` | Assigned sites only where data is site-scoped | Implemented |
| Faculty and Staff | `/search`, phone-directory routes, `/my-profile` | Own-site/default staff context in this DEV slice | Implemented |
| No Access | No protected route access | None | Implemented as denied session state |
| Local breakglass | Separate local emergency login route with named accounts, token-hash verification, network restrictions, and audit events | Local IT Admin persona for this slice | Implemented outside the DEV persona switcher |

## DEV Page And API Matrix

| Surface | Allowed personas | Server-side behavior | Site and field notes | Status | Test coverage |
| --- | --- | --- | --- | --- | --- |
| `/api/v1/dev/session`, `/api/v1/dev/login`, `/api/v1/dev/logout` | DEV persona switcher users and local Codex tooling in `APP_ENV=development` | Session payload reflects the selected DEV persona and feature-filtered routes; `POST /api/v1/dev/login` with `activate_mock_session=true` also updates the shared in-process DEV mock session consumed by `/api/v1/dev/session` after Browser refresh/navigation | Normal login/logout write only the local DEV session cookie; tooling activation is development-only, supports all DEV personas including `no_access`, preserves default/current site context, and forces anonymous readback after invalid persona ids | Implemented | `TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment`, `TestDevSharedMockPersonaToolingSwitchesFrontendSessionReadback`, `TestDevSharedMockPersonaToolingInvalidPersonaFailsClosed`, `TestDevSharedMockPersonaToolingDeniedOutsideDevelopment` |
| `/api/v1/dev/pages/data-quality` | IT Admin | `401` signed out, `403` non-IT | IT Admin-only data-quality awareness surface | Implemented | Data Quality auth tests |
| `/api/v1/dev/search` | Personas with `/search` | Requires authenticated persona and filters by accessible route groups | Employee IDs remain visible/searchable only for IT Admin and HR where the owning payload exposes them | Partially implemented | Global search tests |
| `/api/v1/dev/pages/onboarding` | IT Admin, HR, Site Admin | Requires `/onboarding` route | IT Admin and HR can manage manual drafts; Site Admin gets scoped/read-only visibility | Implemented | Onboarding page and manual draft tests |
| `/api/v1/dev/onboarding/manual-drafts*` | IT Admin, HR | Mutations return `403` for other personas, even with valid payloads | Manual Non-Escape personal phone data is accepted only for HR/IT draft workflows | Implemented | Manual onboarding draft mutation tests |
| `/api/v1/dev/pages/offboarding` | IT Admin, HR, Site Admin | Requires `/offboarding` route | Employee ID and editable end dates are HR/IT only; Site Admin receives assigned-site rows without employee IDs; security-risk orphan rows are excluded | Implemented | Offboarding page tests |
| `/api/v1/dev/offboarding/records/{id}/end-date` | IT Admin, HR | Mutations return `403` for other personas and reject Escape-backed dates | Non-Escape/orphan local end-date mock updates only | Implemented | Offboarding end-date tests |
| `/api/v1/dev/offboarding/candidates` | IT Admin, HR | `401` signed out, `403` non-HR/non-IT before candidate data is returned | Exposes employee IDs and active contractor search corpus only to HR/IT | Implemented | Manual offboarding action tests |
| `/api/v1/dev/offboarding/emergency-deprovision` | IT Admin, HR | `401` signed out, `403` non-HR/non-IT before mutation handling | In-memory DEV mock schedule only; no provider write | Implemented | Manual offboarding action tests |
| `/api/v1/dev/offboarding/contractor-offboarding` | IT Admin, HR | `401` signed out, `403` non-HR/non-IT before mutation handling | Contractors only; requires explicit selected contractor and valid `YYYY-MM-DD` date; no provider write | Implemented | Manual offboarding action tests |
| `/api/v1/dev/pages/reports/security-issues` | IT Admin | `401` signed out, `403` non-IT | Owns recent-activity orphan account security rows; Offboarding does not expose these to HR | Implemented | Security Issues report tests |
| `/api/v1/dev/pages/departing-seniors` | IT Admin, Device Wrangler | Requires `/departing-seniors` route | Retained-year rows and student IDs are limited to this page's allowed personas | Implemented | Departing Seniors access tests |
| `/api/v1/dev/departing-seniors/records/{id}/end-date`, `/deprovision` | IT Admin, Device Wrangler | Mutations return `403` for other personas | DEV mock end-date/deprovision state only | Implemented | Departing Seniors mutation tests |
| `/api/v1/dev/pages/room-moves`, `/room-moves/bulk-draft` | IT Admin, Site Admin, Site Secretary | Requires route and site/draft scope | IT Admin can manage district/inter-site rows; site-scoped users can mutate only visible-site drafts | Implemented | Room Moves page and draft tests |
| `/api/v1/dev/room-moves/*` | IT Admin, Site Admin, Site Secretary according to draft scope | Mutations enforce route, site scope, and admin-only revert authority | DEV mock room-move planning only | Implemented | Room Moves mutation tests |
| `/api/v1/dev/pages/phone-directory/by-person`, `/by-room`, `/by-department` | All logged-in personas | Requires matching route | Directory results are scope-filtered by persona/site; unknown site requests fail closed for site-scoped users | Implemented | Phone Directory tests |
| `/api/v1/dev/feature-flags`, `/api/v1/dev/feature-flags/{key}` | IT Admin | Feature flag reads/writes are IT Admin-only | IT Admin override is read-only in target rows and is not stored as a normal editable target | Implemented | Feature flag handler and persistence tests |
| Static/frontend-only current-slice pages | Varies by session allowed routes | Frontend route guard sends unauthorized direct navigation to login or `403` | `/dashboard/it-admin`, `/dashboard/hr-lifecycle`, `/dashboard/site-admin`, `/frequent-fliers`, `/student-data-cleanup`, `/reports`, `/reports/sync-transparency`, and `/admin` have documented backend coverage exceptions where no route-specific Go page API exists | Partially implemented | Route registry, page-specific frontend tests, and `docs/route-api-authorization-inventory.md` |

## Feature Flags

Feature flags currently control sidebar visibility, direct route access, and matching DEV page/API access where a route-backed API exists. A non-IT user receives a flagged feature only when their persona and current site are effectively enabled. IT Admin always sees every route-level feature and active indicator.

Feature flags are not an in-app permissions administration model. They let DEV verify route availability and persona/site enablement, but they do not grant or revoke real user access, persist authorization history, prevent administrative lockout, or calculate effective permissions across SAML claims, Google groups, manual site mappings, and emergency access. Those behaviors belong to the deferred #160 workstream.

## Field-Level Visibility

| Field | IT Admin | Human Resources | Site Admin / Site Secretary | Device Wrangler | Faculty and Staff | Status |
| --- | --- | --- | --- | --- | --- | --- |
| Employee ID on Onboarding/Offboarding | Visible | Visible | Hidden unless a future documented surface grants it | Hidden | Hidden | Implemented on Offboarding; implemented where Onboarding payload exposes HR/IT workflow fields |
| Last 4 SSN | Full value allowed only in HR/IT manual intake workflows | Full value allowed only in HR/IT manual intake workflows | Hidden | Hidden | Hidden | Partially implemented in DEV manual draft payload; production encryption remains future DB work |
| Personal email / personal phone for manual Non-Escape intake | HR/IT manual draft only | HR/IT manual draft only | Hidden | Hidden | Hidden | Implemented for DEV manual onboarding draft APIs |
| Student IDs on Departing Seniors | Visible | No route access | No route access | Visible for allowed Departing Seniors route | No route access | Implemented |
| Security-risk orphan account details | Visible on `/reports/security-issues` | Hidden from Offboarding | Hidden | Hidden | Hidden | Implemented |

## Known Gaps

### Issue #158: Parent Permission Work Still Open

This matrix documents the currently implemented DEV behavior, and `docs/route-api-authorization-inventory.md` supplies the issue #185 route/API audit evidence for #158. The parent issue still should not close automatically from this matrix alone, because #158 also tracks broader permission enforcement concerns beyond the inventory artifact.

The completed inventory confirms every implemented frontend route has:

- A frontend route entry and direct-navigation behavior.
- A sidebar/navigation visibility rule where the route appears in the shell.
- A route-specific Go page API when the page is runtime-backed.
- Server-side `401` and `403` tests for route-backed APIs.
- Site-scope and field-visibility tests where the page exposes sensitive, personnel, student, room, device, or lifecycle data.
- An explicit documented exception for static frontend-only pages that intentionally do not have a route-specific Go API in the current slice.

The static/frontend-only row in the Page And API Matrix remains a documented implementation boundary. If one of those static pages later loads protected route-specific payload data, that future change must add a runtime DEV API row, matching authorization tests, and an updated inventory entry.

The remaining #158 closure pass should prove enforcement, not just inventory completeness. Before #158 closes, each protected route and API row should have current regression evidence that direct browser navigation, sidebar visibility, feature-flag gating, site-scope filtering, and server-side API authorization agree with this matrix. Static frontend-only exceptions should be rechecked at the same time to confirm they still do not load protected route-specific payloads.

### Issue #160: Editable Permissions Management Deferred

Issue #160 is intentionally deferred. The current DEV persona switcher, feature-flag target editor, and hardcoded mock persona definitions are not sufficient for in-app grant/revoke management.

Real in-app permission management needs a separate editable authorization model that includes:

- Persistent grants, revocations, and site-scope assignments.
- Lockout protection so IT cannot remove the last effective admin path.
- Persistent audit history that records who changed access, what changed, when it changed, and why.
- Effective-permission calculation across direct grants, SAML identity, Google groups or attributes, manual site-scope mappings, feature rollout state, and breakglass/emergency rules.
- Operator UX that distinguishes proposed access changes from active effective access.
- Tests for self-lockout, cross-site leakage, stale grants, revoked users, and conflicting group/manual assignments.

Until that model exists, the matrix should be read as DEV enforcement documentation, not as an editable access administration design.

### Breakglass Boundary

Local breakglass behavior is required by `PRODUCT_REQUIREMENTS.md` and `IMPLEMENTATION_PLAN.md`, but it is not represented as a DEV persona-switcher flow. The current implementation verifies local authentication, named emergency accounts, source-address restrictions, and operational audit events for the emergency path used when third-party authentication is unavailable.

The current row for Local breakglass records the implemented local route only. It does not mean an operator can select a breakglass persona in DEV or administer editable breakglass roles through the permissions model.

## Local Breakglass

- Breakglass is not the DEV persona switcher. DEV persona switching continues to use `/api/v1/dev/login` for mock demonstrations.
- Local emergency access uses `POST /api/v1/breakglass/login` and is enabled only in `development` and `staging`.
- In `staging`, normal DEV persona login remains disabled. Only requests that already carry a valid breakglass session cookie can consume the DEV session/page routes needed for emergency IT Admin access.
- Named emergency accounts are configured with `BREAKGLASS_ACCOUNTS`, for example `emergency-alex,emergency-morgan`.
- Each named account must have one matching SHA-256 token hash environment variable named `BREAKGLASS_TOKEN_SHA256_<SANITIZED_ACCOUNT_ID>`, where non-alphanumeric characters are converted to underscores and the result is uppercased.
- Raw breakglass tokens must be stored only in the approved secret manager or local operator handoff process for the environment. Do not commit them, paste them into tickets, use them as fixtures, or log them.
- The current implementation maps every named breakglass account to the local IT Admin persona. Editable role mapping is out of scope for this slice.
- The default allowed source networks are `10.23.0.0/16` and `10.19.100.0/24`. Set `BREAKGLASS_ALLOWED_CIDRS` to a comma-separated CIDR list only when an approved environment requires a different private source range.
- Token environment variable names are derived from the sanitized account id. Breakglass startup/request parsing rejects account ids that would collide after sanitization, such as `emergency-alex` and `emergency.alex`, so separate named accounts cannot silently share one token hash.
- Direct clients are evaluated by `RemoteAddr` for CIDR checks. `X-Forwarded-For` is honored only when the immediate peer address belongs to `BREAKGLASS_TRUSTED_PROXY_CIDRS`; staging reverse proxies must be listed there before forwarded client addresses are trusted.
- Breakglass cookies are marked `Secure` for `APP_ENV=staging` and for HTTPS requests.

## Audit Expectations

- Successful breakglass login records a `login_attempt` event and an `access_granted` event.
- Denied login records the reason without storing token material. Current denial reasons are `unknown_account`, `source_address_denied`, `token_denied`, and `persona_not_configured`.
- Explicit sign-out records a `sign_out` event when the current local session cookie is breakglass-scoped.
- When `DATABASE_URL` is set, audit events are persisted to `audit_log` with actor type `breakglass_local_account`. Breakglass login and breakglass sign-out fail closed when audit storage cannot initialize or write the required event, so emergency access is never silently granted without audit evidence.
- Without `DATABASE_URL`, DEV keeps audit events in process memory for local tests only.
- Cookie expiration is not actively audited in this slice because there is no durable session table or session-expiration worker. Promotion must either accept explicit sign-out evidence for this slice or define durable session persistence before requiring expiration audit evidence.

## Operator Notes

1. Activation: configure `APP_ENV=development` or `APP_ENV=staging`, set `BREAKGLASS_ACCOUNTS`, set every required per-account token hash, and keep `BREAKGLASS_ALLOWED_CIDRS` unset unless the environment has an approved non-default private source range.
2. Verification: from an allowed source address, post a named account and token to `/api/v1/breakglass/login`, confirm the response has `authentication_mode: "breakglass"`, confirm the session can reach an IT Admin-only route, confirm the cookie is `Secure` in staging or HTTPS, and confirm audit records exist.
3. Denial checks: test an unknown account and a request from outside the allowed CIDR list. Neither request should receive a session cookie.
4. Cleanup: remove the account id from `BREAKGLASS_ACCOUNTS`, remove or rotate the corresponding token hash secret, and verify the old cookie no longer resolves after the configuration reload or process restart.
5. Incident review: inspect `audit_log` for account id, source IP, action, outcome, failure code, request id when present, and timestamp. Never ask operators to provide the raw token as evidence.

### Production Authorization Boundary

Google Workspace admin decisions are still needed for the exact group names, SAML attribute names, ACS URL, metadata source, and certificate delivery method. Persistent manual site-scope administration is also follow-up work. Until it exists, production site scopes should come from deployment-managed mapping JSON or Google group/attribute inputs.

Issue #188 should remain open until the production path validates Google SAML assertions, creates production session cookies from verified identity data, handles configured SAML/SSO sign-in and sign-out flows, and proves that current Google group or attribute inputs recalculate roles and site scopes on every authorization evaluation. The DEV persona cookie, feature-flag target editor, and mock persona payloads are useful for local route/API testing only and are not acceptable production auth sources.

The durable product target remains:

- Google SAML authentication.
- Google group or attribute-based role assignment.
- Persistent site-scope mapping where Google groups do not fully express the needed scope.
- Site-access audit visibility for site staff and IT.
- Application-level staff-domain gating before normal role authorization.

The current DEV model uses mock personas, local session cookies, static allowed-route lists, DEV feature flags, and mock site scopes to validate route/API behavior. Those mechanisms must not be treated as the final production authorization source of truth.
