# Permissions Matrix

This document records both the production authorization contract and the currently implemented DEV authorization behavior for The WIZARD. It does not replace the editable permissions model or breakglass runtime work. It gives reviewers a durable baseline for the Google SAML and Google group/attribute contract, the DEV persona-switcher behavior, and the route/API boundaries that must stay aligned with `PRODUCT_REQUIREMENTS.md`, `IMPLEMENTATION_PLAN.md`, and `TEST_MATRIX.md`.

## Source Order

- `PRODUCT_REQUIREMENTS.md` defines the staff-only product boundary, allowed domains, explicit student denial, same-URL access-denied requirement, lifecycle visibility rules, and HR/IT offboarding action requirements.
- `IMPLEMENTATION_PLAN.md` defines the implementation contract, staged rollout constraints, Phase 2 live-write pilot gate, route registry, and follow-up work still required before production SAML is live.
- `internal/auth/production.go` contains the current checked-in evaluator for verified Google identity data.
- `internal/web` contains the DEV persona-switcher route and API authorization behavior.
- `TEST_MATRIX.md` defines the scenarios that must be evidenced in dev and staging.

## Production Auth Flow

1. Google Workspace authenticates the user through SAML.
2. The application receives verified identity data from the future SAML middleware: email address, group memberships, and configured SAML attributes.
3. The application canonicalizes the email address and applies the domain gate before any normal role authorization.
4. The application denies `@stu.wusd.org` even if Google groups or attributes would otherwise map to an application role.
5. The application allows only `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org` for normal SAML users.
6. The future breakglass runtime may bypass the domain gate only for named local emergency accounts after its own local-auth and network-source checks pass.
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

## Current DEV Route Rules

- The dashboard is staff-only. Allowed staff domains are `@wusd.org`, `@it.wusd.org`, and `@staff.wusd.org`; `@stu.wusd.org` is denied before normal role authorization. Local breakglass accounts are the only documented domain-gate exception.
- Unauthenticated users receive `401` for protected DEV page and API routes.
- Authenticated users without route, persona, feature-flag, site-scope, or field-level permission receive `403` or the route's normal access-denied behavior.
- Sidebar hiding and disabled controls are defense-in-depth only. Server-side DEV APIs must enforce authorization before returning protected data or mutating DEV state.
- IT Admin has all current implemented routes and an override for route-level feature flags. Human Resources has district-wide lifecycle access. Site Admin, Site Secretary, Device Wrangler, and Faculty and Staff receive only the route set and site/field visibility listed below.

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
| Local breakglass | Documented as a required emergency access pattern, with named local accounts and network restrictions | Configurable emergency scope | Not implemented in the DEV persona switcher |

## DEV Page And API Matrix

| Surface | Allowed personas | Server-side behavior | Site and field notes | Status | Test coverage |
| --- | --- | --- | --- | --- | --- |
| `/api/v1/dev/session`, `/api/v1/dev/login`, `/api/v1/dev/logout` | DEV persona switcher users | Session payload reflects the selected DEV persona and feature-filtered routes | Login/logout write only the local DEV session cookie | Implemented | `TestDevSessionLoginLogoutAndDataQualityRoutesInDevelopment` |
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
| Static/frontend-only current-slice pages | Varies by session allowed routes | Frontend route guard sends unauthorized direct navigation to login or `403` | `/dashboard/site-admin`, `/student-data-cleanup`, and `/frequent-fliers` currently have documented backend coverage exceptions where no route-specific Go page API exists | Partially implemented | Route registry and page-specific frontend tests |

## Feature Flags

Feature flags currently control sidebar visibility, direct route access, and matching DEV page/API access where a route-backed API exists. A non-IT user receives a flagged feature only when their persona and current site are effectively enabled. IT Admin always sees every route-level feature and active indicator.

## Field-Level Visibility

| Field | IT Admin | Human Resources | Site Admin / Site Secretary | Device Wrangler | Faculty and Staff | Status |
| --- | --- | --- | --- | --- | --- | --- |
| Employee ID on Onboarding/Offboarding | Visible | Visible | Hidden unless a future documented surface grants it | Hidden | Hidden | Implemented on Offboarding; implemented where Onboarding payload exposes HR/IT workflow fields |
| Last 4 SSN | Full value allowed only in HR/IT manual intake workflows | Full value allowed only in HR/IT manual intake workflows | Hidden | Hidden | Hidden | Partially implemented in DEV manual draft payload; production encryption remains future DB work |
| Personal email / personal phone for manual Non-Escape intake | HR/IT manual draft only | HR/IT manual draft only | Hidden | Hidden | Hidden | Implemented for DEV manual onboarding draft APIs |
| Student IDs on Departing Seniors | Visible | No route access | No route access | Visible for allowed Departing Seniors route | No route access | Implemented |
| Security-risk orphan account details | Visible on `/reports/security-issues` | Hidden from Offboarding | Hidden | Hidden | Hidden | Implemented |

## Known Gaps

- This matrix is current DEV implementation documentation, not a statement that #158 is fully complete. Static frontend-only pages still need a broader route/API inventory pass before #158 can close.
- #160 is intentionally deferred. In-app persona grant/revoke management needs a separate editable model, lockout protection, persistent audit history, and effective-permission behavior.
- Google Workspace admin decisions are still needed for the exact group names, SAML attribute names, ACS URL, metadata source, and certificate delivery method.
- Persistent manual site-scope administration is still follow-up work. Until it exists, production site scopes should come from deployment-managed mapping JSON or Google group/attribute inputs.
- Local breakglass behavior is documented but not represented as a DEV persona-switcher flow.
- Production SAML/Google-group authorization and persistent site-scope mapping are future implementation work beyond the current DEV persona model.
