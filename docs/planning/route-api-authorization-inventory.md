# Route/API Authorization Inventory

This artifact is the issue #185 audit evidence for issue #158. It inventories every implemented frontend route in `frontend/src/lib/routeRegistry.js`, records the matching DEV page/API coverage under `internal/web`, and calls out static frontend-only exceptions for the current DEV slice.

Run `npm run route-api-inventory:check` after changing `frontend/src/lib/routeRegistry.js`, `internal/web/app.go`, or this document. The checker verifies that every route-registry entry has a table row and that every API path named here is registered in the Go mux.

Issue #158 remains open after this inventory because the parent acceptance criteria require a full enforcement and regression pass, not only a checked-in route/API table. The closure pass should rerun or add current tests for direct route access, sidebar visibility, feature-flag gating, site-scope filtering, field-level visibility, signed-out `401` behavior, unauthorized `403` behavior, and every registered protected API row. Static frontend-only exceptions should be revalidated during that pass; if a static page starts loading protected route-specific payload data, it must move out of the exception list and gain a runtime DEV API plus authorization tests in the same change.

## Interpretation Rules

- `401` means no valid DEV session cookie is present.
- `403` means an authenticated DEV persona lacks the route, role, feature-flag, site-scope, or field-level permission.
- Direct frontend navigation uses the session `allowed_routes` list issued by `/api/v1/dev/session`. A signed-out direct route resolves to the app's `401` view, and an authenticated but unauthorized direct route resolves to the app's `403` view.
- Sidebar and nested navigation visibility use the same `allowed_routes` list. Phone Directory mode routes are the documented exception: they are route-backed, but the sidebar shows one top-level Phone Directory row while the in-page mode control owns `/by-person`, `/by-room`, and `/by-department`.
- Feature-flagged routes are checked at session route construction and again by matching runtime-backed DEV APIs. IT Admin has the documented override.

## Frontend Route And DEV API Inventory

| Route | Allowed personas | Sidebar/nav visibility | Direct navigation | Feature flag behavior | Runtime/API backing | Site scope and field evidence |
| --- | --- | --- | --- | --- | --- | --- |
| `/login` | Public logged-out route | Not in logged-in sidebar | Public login page; authenticated authorized users redirect through `/dashboard` | None | No protected page API | No protected data |
| `/dashboard` | Authenticated authorized personas | Not a sidebar destination; resolves to role landing route | Redirects to role landing route; signed-out users see `401` | None | Frontend redirect only | No route-specific payload |
| `/dashboard/it-admin` | IT Admin | Dashboard parent visible only when a dashboard child is allowed | Signed-out `401`; non-IT `403` | None | Static frontend-only current-slice exception | IT Admin-only shell/static dashboard |
| `/dashboard/hr-lifecycle` | IT Admin, Human Resources | Dashboard parent visible for IT/HR dashboard routes | Signed-out `401`; unauthorized personas `403` | None | Static frontend-only current-slice exception | District-wide HR lifecycle shell/static dashboard |
| `/dashboard/site-admin` | IT Admin, Site Admin | Dashboard parent visible for Site Admin and IT | Signed-out `401`; unauthorized personas `403` | `dashboard.site_admin` gates non-IT route access; IT override | Static frontend-only flagged exception | Site Admin assigned-site shell/static dashboard |
| `/search` | All logged-in personas | Shared header search entrypoint; no sidebar row | Signed-out `401`; all authenticated DEV personas can access | None | `GET` `/api/v1/dev/search` | Search filters result groups by route access; employee IDs remain IT/HR-only where owning payload exposes them |
| `/onboarding` | IT Admin, Human Resources, Site Admin, Site Secretary | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `onboarding` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/onboarding`; `POST` `/api/v1/dev/onboarding/manual-drafts`; `PUT/POST/DELETE` `/api/v1/dev/onboarding/manual-drafts/`; `PUT` `/api/v1/dev/onboarding/rows/` | HR/IT can manage manual drafts and sensitive intake fields; Site Admin and Site Secretary receive active-site-scoped visibility and Room-only drawer mutation access |
| `/offboarding` | IT Admin, Human Resources, Site Admin | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `offboarding` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/offboarding`; `PUT` `/api/v1/dev/offboarding/records/`; `GET` `/api/v1/dev/offboarding/candidates`; `POST` `/api/v1/dev/offboarding/emergency-deprovision`; `POST` `/api/v1/dev/offboarding/contractor-offboarding` | HR/IT see employee IDs, search emergency/contractor candidates, schedule immediate or future-dated DEV mock emergency deprovisioning, schedule DEV mock contractor offboarding, and manage local orphan end dates; HR candidate search omits IT Admin targets, and direct HR schedule payloads for IT Admin targets return denial before mock state is created; Site Admin sees assigned-site rows without employee IDs and cannot use HR/IT manual action APIs |
| `/departing-seniors` | IT Admin, Device Wrangler | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `departing_seniors` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/departing-seniors`; `PUT/POST` `/api/v1/dev/departing-seniors/records/` | Student IDs and device-return lifecycle rows are limited to allowed personas |
| `/room-moves` | IT Admin, Site Admin, Site Secretary | Sidebar row visible when `/room-moves` or `/room-moves/bulk-draft` is allowed | Signed-out `401`; unauthorized personas `403` | `room_moves` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/room-moves`; `POST` `/api/v1/dev/room-moves/drafts`; `PUT/POST/DELETE` `/api/v1/dev/room-moves/drafts/`; `GET` `/api/v1/dev/room-moves/completed`; `POST` `/api/v1/dev/room-moves/completed/` | IT can manage district/inter-site drafts and completed-job reverts; site-scoped users can mutate visible-site drafts only |
| `/room-moves/bulk-draft` | IT Admin, Site Admin, Site Secretary | Same Room Moves sidebar row; bulk route is not a separate top-level row | Signed-out `401`; unauthorized personas `403` | `room_moves` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/room-moves/bulk-draft`; shared draft APIs `/api/v1/dev/room-moves/drafts` and `/api/v1/dev/room-moves/drafts/` | Uses the same room-move district/site scope rules as `/room-moves` |
| `/phone-directory/by-person` | All logged-in personas | One Phone Directory sidebar row; in-page mode control selects route | Signed-out `401`; all authenticated DEV personas can access | `phone_directory` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/phone-directory/by-person` | Directory results are persona/site filtered; unknown site requests fail closed for site-scoped users |
| `/phone-directory/by-room` | All logged-in personas | One Phone Directory sidebar row; in-page mode control selects route | Signed-out `401`; all authenticated DEV personas can access | `phone_directory` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/phone-directory/by-room` | Directory results are persona/site filtered; unknown site requests fail closed for site-scoped users |
| `/phone-directory/by-department` | All logged-in personas | One Phone Directory sidebar row; in-page mode control selects route | Signed-out `401`; all authenticated DEV personas can access | `phone_directory` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/phone-directory/by-department` | Department rows exclude people, common-area phones, classroom shared line groups, and auto attendants |
| `/data-quality` | IT Admin | Sidebar row visible for IT Admin only | Signed-out `401`; non-IT `403` | None | `GET` `/api/v1/dev/pages/data-quality` | IT Admin-only awareness/escalation surface |
| `/frequent-fliers` | IT Admin, Site Admin, Device Wrangler | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `frequent_fliers` gates non-IT route access; IT override | Static frontend-only flagged exception | Site-scoped visibility is documented; no route-specific protected payload API in current slice |
| `/meraki-last-seen` | IT Admin, Site Admin, Device Wrangler | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `meraki_last_seen` gates non-IT route/API access; IT override | `GET` `/api/v1/dev/pages/meraki-last-seen` | Site-scoped users receive only assigned-site assigned-student, classroom-spare, and ambiguous device rows; IT Admin receives all sites |
| `/student-data-cleanup` | IT Admin, Site Admin, Site Secretary | Sidebar row visible when route is allowed | Signed-out `401`; unauthorized personas `403` | `student_data_cleanup` gates non-IT route access; IT override | Static frontend-only flagged exception | Site-scoped visibility is documented; no route-specific protected payload API in current slice |
| `/reports` | IT Admin | Reports parent row visible for IT Admin | Signed-out `401`; non-IT `403` | None | Static frontend-only current-slice exception | Report hub shell/static inventory only |
| `/reports/security-issues` | IT Admin | Reports child button visible only when child route is allowed | Signed-out `401`; non-IT `403` | None | `GET` `/api/v1/dev/pages/reports/security-issues` | IT Admin-only security-risk orphan account rows; HR Offboarding does not expose these rows |
| `/reports/zoom-desk-phone-renames` | IT Admin | Reports child button visible only when child route is allowed | Signed-out `401`; non-IT `403` | None | `GET` `/api/v1/dev/pages/reports/zoom-desk-phone-renames` | IT Admin-only pending/manual/error Zoom desk phone rename rows with IncidentIQ asset links; healthy/completed/non-actionable phones are filtered out |
| `/reports/sync-transparency` | IT Admin | Reports child button visible only when child route is allowed | Signed-out `401`; non-IT `403` | None | Static frontend-only current-slice exception | Static report shell only |
| `/admin` | IT Admin | Admin parent row visible for IT Admin | Signed-out `401`; non-IT `403` | None | Static frontend-only current-slice exception | IT Admin-only control-surface shell |
| `/admin/feature-flags` | IT Admin | Admin child button visible only when child route is allowed | Signed-out `401`; non-IT `403` | Feature flag management surface; IT Admin override is read-only and not stored as an editable target | `GET` `/api/v1/dev/feature-flags`; `PUT` `/api/v1/dev/feature-flags/` | IT Admin-only route-level rollout controls; non-IT target rows are editable by IT only |
| `/my-profile` | All logged-in personas | Available through profile/user chrome, not a main sidebar row | Signed-out `401`; all authenticated DEV personas can access | None | `GET/PUT` `/api/v1/dev/my-profile` | Self-service mock profile data for the current persona only; no cross-persona data |

## Static Frontend-Only Exceptions

These routes intentionally have no route-specific protected Go page API in the current DEV slice:

- `/dashboard/it-admin`
- `/dashboard/hr-lifecycle`
- `/dashboard/site-admin`
- `/frequent-fliers`
- `/student-data-cleanup`
- `/reports`
- `/reports/sync-transparency`
- `/admin`

The exception is valid only while the route is static frontend shell/layout with no protected route-specific payload. If any of these pages begins loading personnel, student, room, device, workflow, feature-control, or provider-backed data, the same change must add a runtime DEV API row and focused `401`/`403` tests.

## Runtime Auth Test Coverage

`internal/web/dev_frontend_test.go` includes focused route/API authorization coverage for:

- signed-out `401` behavior on protected runtime-backed DEV endpoints;
- authenticated-but-unauthorized `403` behavior on runtime-backed endpoints that are not available to every staff persona;
- feature-flag route/API denial for non-IT users when a flag target is disabled;
- site-scope and field-visibility evidence for Onboarding, Offboarding, Departing Seniors, Room Moves, Phone Directory, Search, and Security Issues.

The inventory checker does not replace handler tests. It prevents drift between the frontend route registry, Go mux registrations, and this checked-in audit table.

## Session And DEV Support APIs

These DEV API registrations support the route authorization model but do not correspond to one protected page route:

| API path | Purpose | Expected auth behavior |
| --- | --- | --- |
| `/api/v1/dev/session` | Returns anonymous session state before login and the active persona, allowed routes, feature flags, shell context, and site context after login or terminal-tooling activation. | Returns `200` for anonymous and authenticated DEV sessions; in `APP_ENV=development`, a shared tooling override takes precedence over stale Browser cookies so refresh/navigation reads the persona selected from terminal tooling. |
| `/api/v1/dev/login` | Sets the mock DEV persona session cookie; when request JSON includes `activate_mock_session=true`, also sets the process-local active DEV mock session used by Codex/browser evidence workflows. | DEV-only mock login endpoint; invalid persona requests fail without granting protected route access, and invalid tooling activation forces anonymous readback instead of leaving a stale authorized persona active. |
| `/api/v1/dev/logout` | Clears the mock DEV persona session cookie and clears an active tooling-selected DEV mock session to anonymous. | DEV-only mock logout endpoint; returns anonymous session state after clearing the cookie or shared mock session. |
| `/api/v1/breakglass/login` | Sets a breakglass-scoped local IT Admin session cookie for a configured named emergency account after token-hash, source-address, and audit checks pass. | Enabled only in `development` and `staging`; it is separate from DEV persona switching, rejects invalid account configuration before issuing a cookie, records sanitized audit evidence, and allows staging DEV session/page routes only through an authenticated breakglass cookie. |
