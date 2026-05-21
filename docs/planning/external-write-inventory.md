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

`internal/db/schema.sql` defines local tables, constraints, and `external_request_log`. Database writes are expected to run through transaction helpers that match the architecture rules in `.agents/AGENTS.md`.

Current code mostly exercises schema and retry behavior through tests. Future repository-owned database write code must document:

- table names being changed,
- transaction isolation expectations,
- retry behavior through `internal/db.WithRetry`,
- audit or request-log rows,
- how to debug serialization/deadlock retries.

Implemented Phase 0 job-lease write primitives:

- `internal/db.StartScheduledWorkflowRun` acquires a transaction-level PostgreSQL advisory lock keyed by `job_family`, reads active scheduled `workflow_runs` rows for that family, and inserts either a `running` scheduled workflow run or a separate `deferred` overlap row. Deferred rows set `deferred_from_run_id`, `overlap_state = deferred_due_to_active_run`, and the next family `overlap_count` without mutating the active row, `jobs`, `external_request_log`, or any provider. Scheduler callers must execute it inside `internal/db.WithRetry` so the family lock, active-run read, and insert are retried together on serialization or deadlock errors.
- `internal/db.ClaimNextJob` updates one `jobs` row from `queued` to `running`, sets `lease_owner`, `lease_expires_at`, `lease_heartbeat_at`, and `updated_at`, and returns the claimed row ordered by `global_tick`. Worker callers must execute it inside `internal/db.WithRetry` so the claim participates in a `SERIALIZABLE` transaction and can be retried on serialization or deadlock errors.
- `internal/db.RecoverExpiredJobLeases` updates expired `running` rows in `jobs` to `recovering`, clears `lease_owner` and `lease_expires_at`, preserves the prior owner and nullable heartbeat in the returned evidence rows, and uses `FOR UPDATE SKIP LOCKED` so concurrent recovery loops do not handle the same job twice. It also returns already-`recovering` rows so an interrupted recovery pass can finish reconciliation on the next loop instead of leaving the job stranded.
- `internal/db.ReconcileRecoveredJob` reads `external_request_log` for a `succeeded` row tied to the crashed job. If that evidence exists, it updates `jobs.job_state` to `succeeded` without increasing `attempt_count`; otherwise it updates the job back to `queued`, clears lease fields, and increments `attempt_count` so the normal idempotent worker path can retry.
- These primitives do not call providers. Future provider workers must still perform provider read-before-write reconciliation before any live write and must keep `external_request_log` idempotency behavior documented here.

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

`internal/web/dev_frontend.go` handles DEV login/logout/session and feature flag state. Login/logout write or clear the local DEV session cookie. When `POST /api/v1/dev/login` receives `activate_mock_session=true` in `APP_ENV=development`, it also writes a process-local shared DEV mock session override so Codex or terminal tooling can switch the persona consumed by `/api/v1/dev/session` without browser clicks. Invalid tooling persona ids clear the cookie in the response and force the shared override to anonymous so stale Browser cookies do not silently restore authorized access. Normal DEV persona login remains development-only. Staging can consume DEV session/page routes only when a valid breakglass cookie is already present, so emergency access works without enabling persona switching. `internal/web/breakglass.go` handles the separate local emergency breakglass login route for `development` and `staging`; it writes the same local session cookie with a breakglass-scoped value, marks that cookie `Secure` for staging or HTTPS, and records sanitized breakglass audit events before access is granted. Feature flag routes mutate DEV feature flag configuration in memory when `DATABASE_URL` is unset. When `DATABASE_URL` is configured, feature flag refresh and update paths reconcile `feature_flags` and `feature_flag_targets`, update changed `feature_flag_targets.enabled` values through `db.WithRetry`, and write matching `audit_log` rows with `dev_feature_flag_update` diffs. Unchanged target updates are skipped so repeated requests do not create duplicate audit entries.

Expected debugging path: frontend persona or feature controls call `/api/v1/dev/...`; handler checks development mode or a valid breakglass session through `devSessionConsumerEnabled`, validates method and persona context, mutates local state or cookies, then returns JSON.

Mutation routes include:

- `POST /api/v1/breakglass/login`
- `POST /api/v1/dev/login`
- `POST /api/v1/dev/logout`
- `PUT /api/v1/dev/feature-flags/{key}`

Login and logout write or clear the local DEV session cookie for normal persona sessions; tooling activation additionally writes the in-memory shared DEV mock session override described above. Breakglass login accepts a named account id plus token, rejects account ids that collide after token env-name sanitization, rejects any listed named account with a missing or malformed SHA-256 token hash, verifies the configured token hash and source CIDR, writes a breakglass-scoped local session cookie, and records login/access/denial audit events in memory for database-free DEV or in `audit_log` when `DATABASE_URL` is configured. Direct breakglass clients are checked by `RemoteAddr`; `X-Forwarded-For` is used only when the immediate peer is listed in `BREAKGLASS_TRUSTED_PROXY_CIDRS`. Breakglass login and breakglass sign-out fail closed if audit storage cannot initialize or write the required event. DEV logout also records a sanitized `sign_out` audit event when the current cookie is breakglass-scoped. The feature flag route persists IT Admin-only DEV feature flag target state for persona and site visibility. These routes are documented so session-affecting and feature-flag mock behavior does not disappear from the route inventory when the route drift check runs.

### My Profile

`internal/web/dev_my_profile.go` serves the pre-phase 0 My Profile direct-edit mock API. The GET route returns non-secret mock profile fields for the signed-in DEV persona. The PUT route validates preferred/display first name, preferred/display last name, and pronouns, then overwrites only an in-memory DEV profile store for that persona. It does not update legal-name source data, source-system records, the database, Google, Zoom, Aeries, IncidentIQ, or any provider SDK. The route explicitly rejects unauthenticated callers and student-like personas before it mutates mock state.

Mutation routes include:

- `PUT /api/v1/dev/my-profile`

Repeated PUT requests with the same payload are idempotent in the DEV mock store. Debugging should start from the My Profile drawer save request, then follow `handleDevMyProfile`, field validation, and `buildDevMyProfilePayload` to confirm the returned display name changed while legal-name fields stayed source-authoritative.

### Onboarding Manual Drafts And Room Overrides

`internal/web/dev_onboarding.go` creates, updates, finalizes, and soft-deletes manual onboarding drafts. It also stores DEV-only onboarding row room overrides from the selected-person drawer. These routes model intake collision handling, workflow readiness, and room-correction permissions without writing to live HR, Google, Zoom, IncidentIQ, Aeries, or Verkada systems.

Mutation routes include:

- `POST /api/v1/dev/onboarding/manual-drafts`
- `PUT /api/v1/dev/onboarding/manual-drafts/{id}`
- `POST /api/v1/dev/onboarding/manual-drafts/{id}/finalize`
- `DELETE /api/v1/dev/onboarding/manual-drafts/{id}`
- `PUT /api/v1/dev/onboarding/rows/{id}/room`

The room update route is owned by DEV onboarding room overrides and mutates only the in-memory DEV onboarding store. Site Admin and Site Secretary callers must be able to see the row in their active site scope and may submit only `room_id`; any non-Room field attempt is rejected before the store is touched. HR and IT Admin keep their documented broader onboarding behavior while using the same mock room-override boundary for row-level room corrections.

Manual Non-Escape draft updates capture a required personal phone number for the planned Aeries upload payload. The DEV mock API accepts either canonical `10`-digit input or the drawer-submitted `(NNN) NNN-NNNN` display format, rejects other formatting, and stores only the canonical `10`-digit value inside the editable draft payload used by HR/IT. `internal/provider.BuildAeriesUploadPayload` includes that value only when the source is `manual_non_escape`. ESCAPE-sourced Aeries planning omits this field so imported phone data remains source-authoritative. Raw personal phone numbers must stay out of diagnostics, audit summaries, generated artifacts, and fixtures unless a future checked-in requirement documents a safe display need.

### Offboarding And Departing Seniors

`internal/web/dev_offboarding.go` and `internal/web/dev_departing_seniors.go` update mock end dates, mock immediate offboarding decisions, mock contractor offboarding schedules, and deprovisioning status. These model operator decisions and device/asset follow-up without writing to Escape, Google, Zoom, IncidentIQ, Aeries, Active Directory, Entra, or a production database.

The Offboarding candidate-search route is read-only but still HR/IT-only because it exposes employee IDs and active contractor records for the drawer search experience. Emergency and contractor offboarding schedule routes mutate only the in-memory DEV action list with actor, target, timestamp, schedule, status, and `dev_mock_only` mode. They are future live-write boundaries; any provider-backed implementation must add what-if validation, deterministic idempotency, audit persistence, rollback references, post-write verification, and the Phase 2 pilot allowlist check before a provider mutation can run.

Mutation routes include:

- `PUT /api/v1/dev/offboarding/records/{id}/end-date`
- `POST /api/v1/dev/offboarding/emergency-deprovision`
- `POST /api/v1/dev/offboarding/contractor-offboarding`
- `PUT /api/v1/dev/departing-seniors/records/{id}/end-date`
- `POST /api/v1/dev/departing-seniors/records/{id}/deprovision`

### Room Moves

`internal/web/dev_room_moves.go` creates, updates, transitions, cancels, deletes, applies, and schedules reverts for mock room-move drafts. These mutations model future room, extension, and Zoom workflow effects in memory only. Create, update, schedule, and apply paths reject same-room moves by stable current/destination room id so a no-op drawer edit cannot become a planned provider write. Updating an existing seeded single-move row stores the edited draft under the same draft id, preserves the original scoped site when a partial update omits it, and suppresses the seed row in the review table, which models an update to the selected workflow rather than a second workflow. Untouched site-rollover roster rows with destination room `none` remain neutral placeholders and cannot be scheduled or applied until an operator chooses a destination or explicit removal action. Site Admin and Site Secretary personas may view assigned-site rows authored by IT or another operator, but direct DEV mock mutation calls for those rows return `403`; only IT Admin or the original author can save, apply, schedule, cancel, or delete a draft.

Repeated-user bulk drafts remain DEV-only mock planning today. The mock normalizer groups rows for the same person, preserves all rows, allows one primary desk-phone destination, treats secondary/tertiary/later destinations as shared-line-group-only memberships, keeps common-area/CAP coverage active for non-primary destinations, and returns review warnings when the primary destination is ambiguous. This models planned future Zoom room SLG, primary phone assignment, CAP/common-area, and IncidentIQ room association writes; it does not call Zoom, IncidentIQ, or the database.

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
