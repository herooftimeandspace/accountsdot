# External Write Inventory

This inventory tracks code paths that write, plan to write, or simulate writes to systems outside a pure read-only flow. It includes live provider writes, planned provider-operation names, local database writes, and DEV mock store mutations.

## Current Live Provider Writes

No live Zoom, IncidentIQ, Google, Aeries, or SFTP SDK write call is implemented in this checkout. Provider-facing code currently defines contracts, planner operation names, and Google Sheets helper formulas, but it does not call a live provider SDK to mutate provider state.

Any future live provider write must document:

- exact provider method or HTTP endpoint,
- caller route/job,
- idempotency key source,
- `external_request_log` behavior,
- retry and rollback expectations,
- staging validation path,
- sanitized audit/debug output.

## Planned Provider Operations

`internal/orchestrator.PlanWorkflow` emits `core.WorkflowJob` records with provider-operation names. These are planned units of work. They are not live SDK calls today, but future workers are expected to implement them safely.

### Zoom

Planned Zoom operations include user lookup, user creation or linking, extension assignment, calling-plan assignment, room shared-line-group creation, room membership changes, site-extension cutover, room common-area-phone coverage, room verification, phone assignment removal, and user deprovisioning.

Write-capable operation names include:

- `zoom.create_or_link_user`
- `zoom.assign_site_extension`
- `zoom.assign_calling_plan`
- `zoom.ensure_room_slg`
- `zoom.add_room_membership`
- `zoom.remove_room_membership`
- `zoom.apply_site_extension_cutover`
- `zoom.ensure_room_cap`
- `zoom.remove_phone_assignment`
- `zoom.deprovision_user`

Validation/read-style operation names include:

- `zoom.read_user`
- `zoom.verify_room_membership`
- `zoom.validate_room_membership`
- `zoom.validate_primary_phone_assignment`
- `zoom.verify_room_cap`

Before any Zoom operation becomes live, it must use deterministic idempotency keys, write an `external_request_log` row, avoid production-first execution, and preserve sanitized provider diagnostics.

### IncidentIQ

Planned IncidentIQ operations currently resolve rooms, resolve room assets, and match people during sync dry runs. The code also uses IncidentIQ as the documented fallback system for work that automation cannot complete directly.

Known operation names:

- `incident_iq.resolve_room`
- `incident_iq.resolve_room_asset`
- `incident_iq.match_person`

These names are read/resolve style in the current planner. If future code creates or updates IncidentIQ tickets, assets, rooms, or users, document the route/job, required operator context, ticket body shape, idempotency strategy, and assignee-ready fallback description.

### Google Sheets

Directory publish planning emits workbook staging, sentinel validation, and pointer application operations:

- `google_sheets.stage_workbook`
- `google_sheets.validate_sentinel`
- `google_sheets.apply_pointers`

`internal/provider/sheets.go` builds sentinel rows and formulas only; it does not call Google APIs. Future live Google Sheets writes must document workbook/tab targets, sentinel checks, pointer-swap safety, rollback behavior, and how partial writes are detected.

### Aeries, SFTP, Photo, And Internal Sync

Aeries and SFTP are treated as inbound or read-oriented sources in the current codebase. Photo checks are represented as `photo.check_delta`. Internal sync operations update local projections and archives rather than external providers.

Known internal operation names include:

- `internal.sync_ingest_subject`
- `internal.sync_update_projection`
- `internal.sync_recheck_subject`
- `internal.archive_completed_sync_rows`
- `internal.clear_sync_exception_overrides`
- `internal.reconcile_subject`

Any future Aeries or SFTP write path requires explicit documentation because current repo policy treats Aeries non-production data as read-only masked previous-year data by default.

## Database Writes

`internal/db/schema.sql` defines local tables, constraints, and `external_request_log`. Database writes are expected to run through transaction helpers that match the architecture rules in `AGENTS.md`.

Current code mostly exercises schema and retry behavior through tests. Future repository-owned database write code must document:

- table names being changed,
- transaction isolation expectations,
- retry behavior through `internal/db.WithRetry`,
- audit or request-log rows,
- how to debug serialization/deadlock retries.

## HTTP Route Inventory

`scripts/check_external_write_inventory.mjs` derives every currently known `POST`, `PUT`, `DELETE`, and future `PATCH` route from the registered `internal/web` handlers, then compares those live mutating route boundaries with the backticked route bullets in this file.

Routes that intentionally do not call a live provider today still need entries here. Their entries should state whether the route is an accepted no-op, a planned workflow boundary, a local-only mock mutation, a cookie/session mutation, or a future external-write boundary. If a future mutating route is intentionally exempt from the inventory, document it with a nearby explanation and an HTML comment in this exact form:

`<!-- external-write-inventory-exception: METHOD /path/{param} -->`

Use exceptions sparingly. Most mutating routes should be normal inventory bullets because reviewers need to see the workflow owner, side effect, and future provider risk.

### Accepted Workflow And Sync Write Boundaries

These routes currently acknowledge operator intent and return accepted JSON responses. They do not call live providers or write the database in this checkout, but they are treated as write boundaries because future worker and persistence code will hang off the same operator actions.

Mutation routes include:

- `POST /api/v1/workflows/{workflow_run_id}/retry`
- `POST /api/v1/approvals/{approval_id}/approve`
- `POST /api/v1/approvals/{approval_id}/reject`
- `POST /api/v1/sync-status/{user_type}/{user_id}/override`
- `POST /api/v1/room-mappings`
- `POST /api/v1/annual-reset`

Current behavior:

- workflow retry accepts a workflow run id and returns `202 Accepted` without queuing live provider work;
- approval approve/reject accepts a decision and returns `202 Accepted` without persisting the approval decision;
- sync override accepts user type and user id path parameters and returns `202 Accepted` without changing provider state;
- room mappings accepts a proposed mapping action and returns `202 Accepted` without changing IncidentIQ or local database records;
- annual reset accepts a reset trigger and returns `202 Accepted` with workflow type `annual_reset_archive` without archiving database rows in this checkout.

When any of these routes become durable write paths, update this inventory with the exact table, provider, idempotency, audit, retry, rollback, and staging-validation behavior before or with the implementation change.

## DEV Mock Mutations

DEV routes mutate in-memory stores to model operator workflows without touching live providers. These are still write paths because they change local runtime state and teach future production behavior.

### DEV Session And Feature Flags

`internal/web/dev_frontend.go` handles DEV login/logout/session and feature flag state. Login/logout write or clear the local DEV session cookie. Feature flag routes mutate DEV feature flag configuration in memory.

Expected debugging path: frontend persona or feature controls call `/api/v1/dev/...`; handler checks `devModeEnabled`, validates method and persona context, mutates local state or cookies, then returns JSON.

Mutation routes include:

- `POST /api/v1/dev/login`
- `POST /api/v1/dev/logout`
- `PUT /api/v1/dev/feature-flags/{key}`

Login and logout only write or clear the local DEV session cookie. The feature flag route persists IT Admin-only DEV feature flag target state for persona and site visibility. These routes are documented so session-affecting and feature-flag mock behavior does not disappear from the route inventory when the route drift check runs.

### Onboarding Manual Drafts

`internal/web/dev_onboarding.go` creates, updates, finalizes, and soft-deletes manual onboarding drafts. These routes model intake collision handling and workflow readiness without writing to live HR, Google, Zoom, IncidentIQ, Aeries, or Verkada systems.

Mutation routes include:

- `POST /api/v1/dev/onboarding/manual-drafts`
- `PUT /api/v1/dev/onboarding/manual-drafts/{id}`
- `POST /api/v1/dev/onboarding/manual-drafts/{id}/finalize`
- `DELETE /api/v1/dev/onboarding/manual-drafts/{id}`

### Offboarding And Departing Seniors

`internal/web/dev_offboarding.go` and `internal/web/dev_departing_seniors.go` update mock end dates and deprovisioning status. These model operator decisions and device/asset follow-up without writing to Escape, Google, Zoom, or IncidentIQ.

Mutation routes include:

- `PUT /api/v1/dev/offboarding/records/{id}/end-date`
- `PUT /api/v1/dev/departing-seniors/records/{id}/end-date`
- `POST /api/v1/dev/departing-seniors/records/{id}/deprovision`

### Room Moves

`internal/web/dev_room_moves.go` creates, updates, transitions, cancels, deletes, applies, and schedules reverts for mock room-move drafts. These mutations model future room, extension, and Zoom workflow effects in memory only.

Mutation routes include:

- `POST /api/v1/dev/room-moves/drafts`
- `PUT /api/v1/dev/room-moves/drafts/{id}`
- `POST /api/v1/dev/room-moves/drafts/{id}/cancel`
- `POST /api/v1/dev/room-moves/drafts/{id}/schedule`
- `POST /api/v1/dev/room-moves/drafts/{id}/apply`
- `DELETE /api/v1/dev/room-moves/drafts/{id}`
- `POST /api/v1/dev/room-moves/completed/{id}/revert`

## Maintenance Checklist

Update this file whenever code adds, removes, renames, or changes:

- a provider operation name,
- an HTTP route that mutates state,
- a database write,
- a DEV mock store mutation,
- a provider SDK/API call,
- idempotency or retry behavior,
- external failure diagnostics.

For intentional mutating route changes:

1. Update `internal/web` route handlers so the registered route and method branch accurately describe the new, removed, renamed, or method-changed route.
2. Update the matching section in this file with a backticked bullet in `METHOD /path/{param}` form, plus the route owner, current side effect, provider/database impact, and whether the route is a live write, planned write boundary, local-only mock mutation, or documented exception.
3. If the route is intentionally exempt, add the `external-write-inventory-exception` HTML comment next to the explanation instead of adding a normal route bullet.
4. If a new mutating handler shape cannot be derived from existing checker rules, update `scripts/check_external_write_inventory.mjs` so the live route derivation and route owner metadata cover that shape.
5. Run `npm run write-inventory:check` and `npm run write-inventory:test` before opening or updating the PR. `make test` also runs the drift check so CI catches inventory drift.
