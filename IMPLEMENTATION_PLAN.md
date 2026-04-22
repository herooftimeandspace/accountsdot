# Go Employee Provisioner v1 Implementation Plan

## Summary
- Build a self-hosted Go service with server-rendered HTML, SSE live status, `/metrics`, and a JSON-only API in front of PostgreSQL.
- Treat the service as mission-critical and TDD-first: keep this file authoritative, write tests before code, confirm expected outputs before client logic, and never change tests to mask broken behavior.
- Use CSV/SFTP + Aeries OneRoster as upstream systems, with manual intake only for contractor/external records absent from upstream.
- Keep dependencies minimal: standard library first; approved runtime dependencies are `pgx/v5`, `google/uuid`, `x/oauth2`, `x/oauth2/google`, `x/crypto/ssh`, `pkg/sftp`, and local `zoom-sdk-golang`.

## Repo Layout
- `cmd/provisioner`
- `internal/core`
- `internal/db`
- `internal/provider`
- `internal/orchestrator`
- `internal/web`
- `pkg/zoom`
- root `README.md`, `.env.example`, `compose.yaml`, `.devcontainer/devcontainer.json`, `Makefile`

## Core Tables
- `people`, `employees`, `contractors`, `external_volunteers`
- `jobs`, `source_records`, `known_identifiers`
- `manual_overrides`, `audit_log`, `record_backups`
- `external_request_log`, `resource_registry`, `extension_inventory`
- `event_outbox`, `sheet_publish_log`, `system_controls`
- `workflow_runs`, `import_batches`, `approval_requests`, `provider_circuit_breakers`

## Identity and Precedence
- `people.uuid` uses UUIDv7.
- `known_identifiers` is keyed by `(source_system, source_id)` and maps to `people_uuid`.
- Intake must check `known_identifiers` before creating a person.
- Ambiguous matches move to `awaiting_review` with reason codes such as:
  - `MATCH_NAME_NO_DOB`
  - `MATCH_NAME_EMAIL_CONFLICT`
  - `MATCH_SOURCE_ID_REUSE`
- Field precedence:
  - HR/CSV: legal identity, employment status, dates, manager, full-time employee number
  - Aeries: site, school, room, roster context
  - Manual: contractor/external-only records and approved override fields

## FSMs
- Person states:
  - `intake_pending`, `normalized`, `reconciled`, `provision_pending_context`, `awaiting_review`
  - `preprovision_ready`, `preprovisioning`, `provision_ready`, `provisioning`, `active`
  - `transfer_pending`, `leave_pending`, `deprovisioning`, `terminated`, `failed`, `on_hold`
- Job states:
  - `queued`, `running`, `recovering`, `blocked`, `waiting_manual`, `succeeded`, `failed`, `skipped`, `canceled`
- Room resource states:
  - `vacant_cap_active`, `transitioning_to_human`, `human_active`, `cleanup_pending`

## Transactions and Ordering
- All FSM transitions, extension mutations, and room-scoped mutations run in `SERIALIZABLE` transactions.
- `jobs` and `event_outbox` use sequence-backed `global_tick`; strict ordering must use `global_tick`, never UUID sort order.
- `internal/db` must expose `WithRetry(ctx, pool, fn)` with jittered exponential backoff for `40001` and deadlock-class failures, up to 5 attempts.

## Allocation and Resource Safety
- `extension_inventory` is the only extension allocator.
- Allocation is two-phase:
  - `available -> reserved_for_job_id -> assigned_to_person_id`
  - canceled or expired reservations are reclaimed by a cleanup worker
- Same-site transfer keeps the human extension.
- Site-to-site transfer allocates a new extension from `extension_inventory`.
- Room-scoped mutations use Postgres advisory transaction locks keyed by room/site.

## Aeries Lag Handling
- People missing room context move to `provision_pending_context`.
- A dedicated context watcher rescans only `provision_pending_context` people against Aeries every 5 minutes with jitter and on every successful Aeries sync.
- The context watcher must hold an advisory lock so only one instance runs the scan cluster-wide.
- Records unresolved after 72 hours move to `awaiting_review` with `CONTEXT_TIMEOUT`.

## Recovery and Idempotency
- Every Zoom and Google write uses a deterministic idempotency key and `external_request_log`.
- Recovering jobs must use read-before-write reconciliation:
  - query the provider first
  - if the intended effect already exists and matches the job intent, mark the step succeeded
  - only perform a write if reconciliation proves it is still needed

## Zoom Rules
- V1 covers create user, set site/extension, assign calling plan, manage room SLG membership, and maintain one room-scoped CAP while a room is vacant.
- Before any SLG membership write, check projected membership against `zoom_slg_max_members`.
- If exceeded, set the job to `blocked` with `PROVIDER_LIMIT_EXCEEDED:ZOOM_SLG_FULL` and expose that clearly in the Tech UI.
- CAP-to-human cutover is safe-order only:
  - keep CAP active until human identity and room membership are confirmed
  - then deprovision CAP as the final step
  - if human cutover fails, a janitor loop restores or preserves the CAP and returns the room to `vacant_cap_active`

## Google Sheets Publishing
- Target tabs remain `Zoom_SLG`, `Zoom_Users`, `Zoom_CallQueues`, `Zoom_CommonArea`, `Zoom_AR`.
- Each publish writes to a versioned staging tab such as `Zoom_Users_STAGING_<global_tick>`.
- Each staging tab ends with a terminal sentinel row containing fixed marker, expected row count, checksum, and publish version.
- `sheet_publish_log` records staging completion, sentinel validation, and pointer-application state.
- Visible production tabs are formula-backed views, never directly rewritten.
- `Sync_Config` cells are fixed:
  - `B2` active sheet for `Zoom_SLG`
  - `B3` active sheet for `Zoom_Users`
  - `B4` active sheet for `Zoom_CallQueues`
  - `B5` active sheet for `Zoom_CommonArea`
  - `B6` active sheet for `Zoom_AR`
- Example visible-tab formula pattern:
  - `=QUERY(INDIRECT(Sync_Config!B3 & "!A:Z"), "select *", 0)` for `Zoom_Users`
- Recovering sheet-publish jobs must detect a valid staged sheet plus sentinel and finish the pointer update instead of rewriting data.
- `Members` and designated extension columns must always be written as text.

## Events, Health, and Controls
- SSE uses durable outbox plus Postgres `LISTEN/NOTIFY`.
- Replay backfill is capped to the last 100 events or 10 minutes, whichever is smaller; older clients receive `resync_required`.
- `system_controls.global_pause` is the global kill switch; the orchestrator must honor it before claiming work and between steps while leaving UI and diagnostics online.
- Health endpoints:
  - `/health/live`
  - `/health/ready`
  - `/health`
- `/health/ready` must validate DB connectivity, sequence access, local import-staging path read/write access, configured SFTP reachability in integration mode, and Google service-account token acquisition.

## Provider Protection
- Circuit breakers use exponential backoff `1s -> 2s -> 4s`, then pause only the affected queue for 15 minutes before a half-open probe.

## Orchestration Loops and Provider Workflows
- Orchestration is durable:
  - `workflow_runs` expand into ordered `jobs`
  - planner logic is the only component that creates job graphs
  - workers execute jobs, recovery reconciles expired leases, and janitor loops clean residual state
- Core orchestration types:
  - `WorkflowType`: `person_onboard`, `person_update`, `person_same_site_transfer`, `person_site_transfer`, `person_leave`, `person_terminate`, `room_coverage`, `directory_publish`, `context_refresh`
  - `WorkflowRunState`: `planned`, `running`, `waiting_manual`, `blocked`, `recovering`, `succeeded`, `failed`, `canceled`
  - `ApprovalState`: `not_required`, `pending`, `approved`, `rejected`, `expired`
  - `ProviderKind`: `hr_sftp`, `aeries`, `zoom`, `google_sheets`, `internal`
- Planner rules:
  - dedupe by `(workflow_type, subject_id, trigger_fingerprint)`
  - person workflows and room workflows are separate
  - room vacancy can emit a separate `room_coverage` workflow
- Required loops:
  - `hr_import_loop`: every `5m`, cluster-wide advisory lock, fingerprint files, create `import_batches`, update `known_identifiers` and `source_records`
  - `aeries_sync_loop`: every `5m` plus after successful HR import, cluster-wide advisory lock, update room/site context only
  - `context_watcher_loop`: every `5m + jitter`, cluster-wide advisory lock, recheck `provision_pending_context`
  - `workflow_planner_loop`: NOTIFY-driven with `30s` fallback sweep, no provider calls
  - `job_worker_loop`: continuous, lease-based job claims, provider/site concurrency enforcement
  - `recovery_loop`: every `30s`, move expired `running` jobs to `recovering` and reconcile with read-before-write
  - `approval_loop`: event-driven with `30s` fallback sweep for stale approvals
  - `janitor_loop`: every `2m`, reclaim extension reservations and restore CAP coverage
  - directory publishing is debounced into one workbook-scoped `directory_publish` workflow with `run_after = now + 60s`
- Provider workflow scope for v1:
  - `HR/SFTP`: normalize identity facts only
  - `Aeries`: sync context facts only
  - `Zoom`: create/link users, assign site/extension/calling plan, manage room SLGs, maintain room CAP coverage
  - `Google Sheets`: workbook-scoped staging, sentinel validation, and pointer updates
- Approval-gated destructive actions:
  - site-transfer cutover that changes active site/extension
  - CAP removal
  - removing a human from a room when coverage is affected
  - Zoom deprovision/license removal
  - extension release after leave/termination
- Provider contracts:
  - `ReadExisting(ctx, ref) -> ProviderSnapshot`
  - `ApplyIntent(ctx, intent) -> ApplyResult`
  - `ClassifyError(err) -> transient | blocked | manual | fatal`
  - Sheets-specific:
    - `StageWorkbook(ctx, publishSpec) -> StageResult`
    - `ValidateSentinel(ctx, stageRef) -> ValidationResult`
    - `ApplyPointers(ctx, pointerSpec) -> ApplyResult`

## Sync Transparency Dashboard
- Data sync is a first-class subsystem inside the workflow engine, not a sidecar report.
- Sync-specific types:
  - `SyncSubjectType`: `staff`, `student`
  - `SyncPhase`: `ingested`, `photo_processed`, `iiq_matched`, `room_mapped`, `zoom_provisioned`
  - `SyncOverallStatus`: `pending`, `in_progress`, `manual_action`, `completed`
  - `SyncIssueCode`: `room_mapping_required`, `licensing_error`, `primary_conflict`, `missing_asset`, `rollover_wait`
- `WorkflowType` additions:
  - `staff_sync_dry_run`
  - `student_sync_dry_run`
  - `sync_recheck`
  - `annual_reset_archive`
- `ProviderKind` additions:
  - `incident_iq`
  - `photo`
- `user_sync_status` is the sync dashboard projection table and must include:
  - `user_id`, `user_type`, `current_phase`, `overall_status`, `last_job_date`, `errors_warnings`, `is_archived`
  - `people_uuid nullable`, `school_year`, `display_name`, `site_code`, `queued_at`, `completion_date`, `completion_summary`, `archived_at`
  - primary key `(user_type, user_id, school_year)`
- `room_mapping_overrides` stores local room normalization and Incident IQ mapping decisions.
- Existing system-of-record tables remain authoritative:
  - `workflow_runs`
  - `jobs`
  - `manual_overrides`
  - `audit_log`
  - `event_outbox`
- Sync workflow rules:
  - Aeries/HR changes create sync workflows first
  - provisioning workflows consume sync readiness, rather than treating sync as an external precondition
  - on first Aeries sighting, create `user_sync_status` immediately with `current_phase=ingested`, `overall_status=pending`, and `queued_at=now`
  - required sync workflows:
    - `staff_sync_dry_run`
    - `student_sync_dry_run`
    - `sync_recheck`
    - `annual_reset_archive`
- Staff completion requires:
  - photo dry-run check complete
  - room mapped to Incident IQ
  - Zoom SLG membership dry-run validated
  - if primary, phone assignment dry-run validated
- Student completion requires:
  - photo dry-run check complete
  - Incident IQ person-record match validated
- Manual-action issue codes:
  - `room_mapping_required`
  - `licensing_error`
  - `primary_conflict`
  - `missing_asset`
- `rollover_wait` is an in-progress warning, not a manual-action block.
- Dashboard routes:
  - HTML:
    - `/sync-dashboard`
    - `/sync-dashboard/mappings`
  - JSON:
    - `/api/v1/sync-status/pending`
    - `/api/v1/sync-status/in-progress`
    - `/api/v1/sync-status/completed`
    - `/api/v1/sync-status/history`
    - `/api/v1/sync-status/{user_type}/{user_id}/override`
    - `/api/v1/room-mappings`
    - `/api/v1/annual-reset`
- Dashboard UI rules:
  - tabs: `Pending`, `In Progress / Manual Actions`, `Completed`, `History`
  - columns: `User`, `Current Step`, `Issue/Action`, `Date`
  - auto-poll every `15s`
  - Completed and History support filtering by `site_code` and `user_type`
- Local-only dashboard actions:
  - `Open Mapping Tool` persists local room mapping overrides
  - `Ignore/Override Exception` persists local manual overrides, updates sync issues, and enqueues `sync_recheck`
- Annual Reset archives completed rows, preserves room-mapping configuration, and clears per-user exception overrides.

## Test Plan
- Unit tests:
  - UUIDv7 generation and parsing
  - `known_identifiers` exact-match linkage
  - precedence decisions and duplicate reason codes
  - `WithRetry` behavior on `40001`
  - two-phase extension allocation and reservation reclaim
  - CAP room-state transitions
  - idempotency-key generation
  - checksum and sentinel builders
  - workflow graph creation for each `WorkflowType`
  - approval-required vs auto-run step classification
  - provider error classification
  - sync enum validation
  - mapping from sync evaluation results into `current_phase` and `overall_status`
  - staff vs student completion rules
  - annual reset archive and override-retention behavior
- Contract tests:
  - JSON API payload shapes
  - Zoom provider request/response mappings
  - SLG limit enforcement
  - Google Sheet row serialization and pointer-cutover contract
  - README disclaimer presence
  - dependency allowlist enforcement
  - workflow and approval API payloads
  - sync dashboard HTML and JSON payloads
- Integration tests:
  - `SERIALIZABLE` contention with retry success
  - advisory-lock room mutation
  - context watcher single-run advisory lock
  - read-before-write recovery after provider-side success
  - staged-sheet recovery that finishes pointer update without rewriting
  - outbox plus `LISTEN/NOTIFY`
  - replay-window overflow returning `resync_required`
  - global pause behavior
  - `/health/ready` dependency checks
  - single-run advisory locks for HR import and Aeries sync
  - worker lease expiry moving jobs to `recovering`
  - first Aeries sighting creates `user_sync_status`
  - room-mapping overrides resolve `room_mapping_required` on reevaluation
  - Annual Reset archives completed sync rows while preserving history
- Scenario tests:
  - new hire
  - same-site transfer
  - site-to-site transfer
  - leave
  - leave-with-no-replacement
  - missing Aeries room context
  - late upstream identifier conflict
  - crash-after-provider-success recovery
  - CAP-to-human cutover failure with janitor restoration
  - blocked `ZOOM_SLG_FULL`
  - directory publish recovery from staged-but-not-pointed state
  - staff sync dry run fully completes
  - student sync dry run fully completes
  - sync dashboard shows room mapping required, licensing error, primary conflict, missing asset, and rollover wait

## Assumptions and Defaults
- The visible Google Sheet tabs can be converted into formula-backed views controlled by `Sync_Config`.
- A staging publish is valid only when the terminal sentinel row matches expected count and checksum.
- Aeries may lag HR by days, so room-context resolution is independently re-polled.
- Defaults:
  - `zoom_slg_max_members = 10`
  - replay backfill = `100 events / 10 minutes`
  - context watcher cadence = `5 minutes + jitter`
  - `SERIALIZABLE` retry attempts = `5`
  - sync dashboard poll interval = `15s`
