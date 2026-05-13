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

## DEV Mock Mutations

DEV routes mutate in-memory stores to model operator workflows without touching live providers. These are still write paths because they change local runtime state and teach future production behavior.

### DEV Session And Feature Flags

`internal/web/dev_frontend.go` handles DEV login/logout/session and feature flag state. Login/logout write or clear the local DEV session cookie. Feature flag routes mutate DEV feature flag configuration in memory when `DATABASE_URL` is unset. When `DATABASE_URL` is configured, feature flag refresh and update paths reconcile `feature_flags` and `feature_flag_targets`, update changed `feature_flag_targets.enabled` values through `db.WithRetry`, and write matching `audit_log` rows with `dev_feature_flag_update` diffs. Unchanged target updates are skipped so repeated requests do not create duplicate audit entries.

Expected debugging path: frontend persona or feature controls call `/api/v1/dev/...`; handler checks `devModeEnabled`, validates method and persona context, mutates local state or cookies, then returns JSON.

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
