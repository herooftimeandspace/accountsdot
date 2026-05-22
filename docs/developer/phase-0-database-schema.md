# Phase 0 Database Schema Review Guide

This guide is the human-readable review companion for `internal/db/schema.sql`
and Phase 0 tracker `P0-DOC-DB` in GitHub issue #282. It exists so reviewers
can inspect a freshly built Phase 0 PostgreSQL database directly without
reverse-engineering table names, lifecycle fields, ordering fields, retry
fields, or sensitive-field handling from the DDL.

Keep this document aligned in the same pull request as any migration or schema
change that touches Phase 0 database tables, database write paths, or reviewer
SQL expectations. Any related Phase 0 issue that changes tables, columns,
indexes, constraints, job/outbox behavior, audit behavior, sensitive persisted
fields, lease/retry semantics, or reviewer database queries should link back to
issue #282 and this file.

## Source Of Truth

- `internal/db/schema.sql` is the authoritative baseline DDL for new databases.
- `internal/db/migrations/` contains forward migrations for existing dev and
  staging databases.
- `docs/planning/implementation-plan.md` defines Phase 0 database, workflow,
  recovery, `global_tick`, and rollback expectations.
- `docs/planning/external-write-inventory.md` defines current database write
  paths and future provider-write request-log requirements.
- `docs/developer/code-paths/job-lease-recovery.md` explains the implemented
  Phase 0 job lease, crash-recovery, and scheduled-overlap paths.

## Ordering And Review Rules

`global_tick_seq` is the sequence-backed ordering primitive. `jobs.global_tick`
and `event_outbox.global_tick` default to `nextval('global_tick_seq')`.
Reviewers should use `global_tick` whenever they need strict work or event
ordering. UUIDs, timestamps, and `bigserial` ids are useful identifiers, but
they are not the strict sequencing contract.

Phase 0 database writes that mutate finite-state-machine rows, room-scoped
state, extension allocation, job claims, recovery, or scheduled-run overlap
decisions are expected to run inside `SERIALIZABLE` transactions through
`internal/db.WithRetry`. That helper retries PostgreSQL serialization and
deadlock-class failures with bounded jittered exponential backoff.

## Constraint And Default Families

Reviewers should inspect constraints and defaults before reading data. Phase 0
uses a small set of explicit schema patterns:

- identity tables use primary keys plus foreign keys back to `people.uuid`
  where the subtype row belongs to one canonical person
- source identifier lookup uses a composite primary key on
  `known_identifiers(people_uuid, source_system, source_id)` plus a separate
  unique index on `(source_system, source_id)` so one upstream id cannot map to
  multiple people
- workflow and job child rows use foreign keys to `workflow_runs`, `jobs`,
  `people`, or `import_batches` with `on delete cascade`, `on delete set null`,
  or self-references according to whether the review row should disappear with
  its parent or preserve historical evidence
- `workflow_runs.deferred_from_run_id` is a self-reference used only for
  scheduled-overlap evidence; a deferred row must not mutate the still-running
  active row
- `external_request_log` enforces provider idempotency with a unique index on
  `(provider, operation, idempotency_key)`
- `feature_flag_targets.target_type` is constrained to `persona` or `site`
- operational timestamps default to `now()` when they record row creation or
  last update; reviewers should compare `updated_at`, lease timestamps, and
  outbox timestamps with `global_tick` rather than replacing tick ordering with
  wall-clock ordering
- JSON fields default to empty objects or arrays where the application expects a
  structured-but-empty review value instead of `null`

## Table Groups

### Identity And Source Truth

| Table | Columns | Key constraints and indexes | Review notes |
| --- | --- | --- | --- |
| `people` | `uuid`, `person_state`, `first_name`, `last_name`, `email`, `created_at`, `updated_at` | Primary key on `uuid` | Canonical person row. `person_state` follows the lifecycle states documented in the implementation plan. Names and email can be sensitive personnel data. |
| `employees` | `people_uuid`, `employee_number` | Primary key and foreign key on `people_uuid`; unique `employee_number` | Full-time employee subtype keyed to `people`. Employee numbers must not be copied into public tickets or logs. |
| `contractors` | `people_uuid`, `generated_employee_number` | Primary key and foreign key on `people_uuid`; unique `generated_employee_number` | Contractor subtype for records absent from upstream HR truth. |
| `external_volunteers` | `people_uuid`, `generated_employee_number` | Primary key and foreign key on `people_uuid`; unique `generated_employee_number` | External volunteer subtype for manually tracked people. |
| `source_records` | `id`, `people_uuid`, `source_system`, `source_id`, `payload`, `created_at` | Primary key on `id`; foreign key to `people` | Stores inbound source payload fragments. `payload` must omit or redact secrets, credentials, raw service-account JSON, auth headers, passwords, private keys, and unnecessary sensitive provider fields. |
| `known_identifiers` | `people_uuid`, `source_system`, `source_id`, `last_seen_at` | Composite primary key on `people_uuid`, `source_system`, `source_id`; unique index `known_identifiers_source_unique` on `source_system`, `source_id` | Maps source-system ids to people. Use this before creating a new person. Source ids may be sensitive and should be minimized in logs. |

### Operator Visibility And Overrides

| Table | Columns | Key constraints and indexes | Review notes |
| --- | --- | --- | --- |
| `user_sync_status` | `user_id`, `user_type`, `school_year`, `people_uuid`, `display_name`, `site_code`, `current_phase`, `overall_status`, `queued_at`, `last_job_date`, `completion_date`, `completion_summary`, `errors_warnings`, `is_archived`, `archived_at` | Composite primary key on `user_type`, `user_id`, `school_year`; nullable foreign key to `people` | Review dashboard projection. `errors_warnings` should contain sanitized diagnostics only. |
| `room_mapping_overrides` | `id`, `school_year`, `source_room`, `normalized_room`, `incident_iq_room_id`, `incident_iq_room_name`, `actor_id`, `created_at`, `updated_at` | Primary key on `id`; unique `school_year`, `source_room` | Operator-approved room mapping override. Actor and room identifiers are internal operational data. |
| `manual_overrides` | `id`, `people_uuid`, `target_user_type`, `target_user_id`, `school_year`, `actor_id`, `reason`, `diff`, `created_at` | Primary key on `id`; nullable foreign key to `people` | Durable operator override record. `diff` must not include raw secrets, personal phone numbers, private notes beyond the required reason, or unredacted provider payloads. |
| `audit_log` | `id`, `actor_id`, `actor_type`, `request_id`, `target_entity`, `target_id`, `reason`, `diff`, `created_at` | Primary key on `id` | Audit trail. `diff` should be sanitized and limited to fields needed to understand the change. Breakglass audit rows use sanitized account, source, and outcome metadata only. |
| `record_backups` | `id`, `target_table`, `target_id`, `snapshot`, `created_at` | Primary key on `id` | Recovery snapshot store. `snapshot` must follow the same masking and omission rules as source payloads and audit diffs. |

### Workflow Runs, Jobs, Approvals, And Outbox

| Table | Columns | Key constraints and indexes | Review notes |
| --- | --- | --- | --- |
| `import_batches` | `id`, `source_system`, `source_fingerprint`, `status`, `row_count`, `failure_summary`, `created_at`, `updated_at` | Primary key on `id` | Tracks imported source batches. `failure_summary` should be actionable without embedding raw rows or credentials. |
| `workflow_runs` | `id`, `workflow_type`, `subject_kind`, `subject_id`, `trigger_type`, `status`, `job_family`, `scheduled_for`, `deferred_from_run_id`, `overlap_state`, `overlap_count`, `approval_state`, `desired_snapshot`, `source_batch_id`, `current_job_count`, `created_at`, `updated_at` | Primary key on `id`; self foreign key `deferred_from_run_id`; nullable foreign key to `import_batches`; partial indexes `workflow_runs_scheduled_family_active_idx` and `workflow_runs_scheduled_family_overlap_idx` | Durable workflow header. `status` lifecycle values include `planned`, `deferred`, `running`, `waiting_manual`, `blocked`, `recovering`, `succeeded`, `failed`, and `canceled`. `desired_snapshot` must avoid raw provider payloads and secret material. |
| `jobs` | `id`, `global_tick`, `workflow_run_id`, `people_uuid`, `job_state`, `job_type`, `provider`, `operation`, `step_key`, `depends_on_step_key`, `attempt_count`, `run_after`, `approval_required`, `reason_code`, `lease_owner`, `lease_expires_at`, `lease_heartbeat_at`, `created_at`, `updated_at` | Primary key on `id`; foreign keys to `workflow_runs` and `people`; partial indexes `jobs_claimable_global_tick_idx` and `jobs_expired_lease_global_tick_idx` | Durable work item. `job_state` lifecycle values include `queued`, `running`, `recovering`, `blocked`, `waiting_manual`, `succeeded`, `failed`, `skipped`, and `canceled`. `attempt_count` is the retry counter. Lease fields are the crash-recovery evidence. |
| `approval_requests` | `id`, `workflow_run_id`, `job_id`, `approval_state`, `reason_code`, `requested_at`, `decided_at`, `decided_by` | Primary key on `id`; foreign key to `workflow_runs`; nullable foreign key to `jobs` | Approval record for destructive or manual-gated work. `approval_state` values include `not_required`, `pending`, `approved`, `rejected`, and `expired`. |
| `event_outbox` | `id`, `global_tick`, `topic`, `event_type`, `payload`, `created_at` | Primary key on `id` | Local event outbox. Use `global_tick` for event order. `payload` should contain sanitized event facts and non-secret identifiers only. |

### Provider Safety, Allocation, Publishing, And Controls

| Table | Columns | Key constraints and indexes | Review notes |
| --- | --- | --- | --- |
| `external_request_log` | `id`, `job_id`, `provider`, `operation`, `idempotency_key`, `request_hash`, `provider_object_id`, `outcome`, `response_summary`, `created_at` | Primary key on `id`; nullable foreign key to `jobs`; unique index `external_request_log_idempotency_key_unique` on `provider`, `operation`, `idempotency_key`; index `external_request_log_job_outcome_idx` on `job_id`, `outcome` | Idempotency and provider-effect evidence. `request_hash` should be a one-way hash, not raw request content. `response_summary` must be sanitized. Do not store tokens, auth headers, passwords, private keys, raw service-account JSON, or full provider responses. |
| `provider_circuit_breakers` | `provider`, `operation_class`, `state`, `failure_count`, `opened_at`, `next_probe_at` | Composite primary key on `provider`, `operation_class` | Provider operation-class safety state. |
| `resource_registry` | `id`, `room_key`, `site_code`, `room_resource_state`, `provider_type`, `provider_object_id`, `assigned_people_uuid`, `created_at`, `updated_at` | Primary key on `id`; unique `room_key`; nullable foreign key to `people` | Room and provider resource registry. `room_resource_state` follows the implementation-plan room resource lifecycle. |
| `extension_inventory` | `extension`, `site_code`, `status`, `reserved_for_job_id`, `assigned_to_people_uuid`, `updated_at` | Primary key on `extension`; nullable foreign keys to `jobs` and `people` | Extension allocator. Status moves through the documented two-phase allocation model, including reservation by job before assignment to a person. |
| `sheet_publish_log` | `id`, `tab_name`, `staging_sheet`, `checksum`, `row_count`, `publish_version`, `sentinel_validated`, `pointer_applied`, `created_at` | Primary key on `id` | Google Sheets publishing evidence. `publish_version` is tied to the strict ordered publish flow. |
| `system_controls` | `control_name`, `enabled`, `reason`, `actor_id`, `updated_at` | Primary key on `control_name` | Global controls such as pause/cutoff state. Reasons should be concise and free of secrets. |
| `feature_flags` | `flag_key`, `label`, `description`, `feature_route`, `default_enabled`, `actor_id`, `created_at`, `updated_at` | Primary key on `flag_key` | DEV/admin feature metadata. |
| `feature_flag_targets` | `flag_key`, `target_type`, `target_id`, `enabled`, `actor_id`, `updated_at` | Composite primary key on `flag_key`, `target_type`, `target_id`; foreign key to `feature_flags`; check constraint limiting `target_type` to `persona` or `site` | Per-persona or per-site flag target state. |

## Lifecycle, Audit, And Retry Fields

- Lifecycle fields:
  - `people.person_state`
  - `user_sync_status.current_phase`
  - `user_sync_status.overall_status`
  - `workflow_runs.status`
  - `workflow_runs.approval_state`
  - `jobs.job_state`
  - `approval_requests.approval_state`
  - `provider_circuit_breakers.state`
  - `resource_registry.room_resource_state`
  - `extension_inventory.status`
  - `system_controls.enabled`
  - `feature_flags.default_enabled`
  - `feature_flag_targets.enabled`
- Audit fields:
  - actor fields: `actor_id`, `actor_type`, `decided_by`
  - reason fields: `reason`, `reason_code`, `failure_summary`, `completion_summary`
  - diff/snapshot fields: `diff`, `snapshot`, `desired_snapshot`, `payload`
  - timestamps: `created_at`, `updated_at`, `requested_at`, `decided_at`,
    `queued_at`, `archived_at`, `last_job_date`, `completion_date`
- Retry and lease fields:
  - `jobs.attempt_count`
  - `jobs.run_after`
  - `jobs.lease_owner`
  - `jobs.lease_expires_at`
  - `jobs.lease_heartbeat_at`
  - `provider_circuit_breakers.failure_count`
  - `provider_circuit_breakers.opened_at`
  - `provider_circuit_breakers.next_probe_at`
- Job and outbox ordering fields:
  - `jobs.global_tick`
  - `event_outbox.global_tick`
  - both use `global_tick_seq`
- Scheduled overlap fields:
  - `workflow_runs.job_family`
  - `workflow_runs.scheduled_for`
  - `workflow_runs.deferred_from_run_id`
  - `workflow_runs.overlap_state`
  - `workflow_runs.overlap_count`

## Direct SQL Review Queries

Run these queries against a non-production Phase 0 database. Do not paste raw
result rows containing names, employee ids, source ids, provider ids, payloads,
diffs, snapshots, request hashes, or response summaries into tickets or logs.
Use counts, ids already approved for sharing, or redacted excerpts instead.

### Confirm Core Tables Exist

```sql
select table_name
from information_schema.tables
where table_schema = 'public'
  and table_name in (
    'people', 'employees', 'contractors', 'external_volunteers',
    'source_records', 'known_identifiers', 'user_sync_status',
    'room_mapping_overrides', 'import_batches', 'workflow_runs',
    'jobs', 'approval_requests', 'manual_overrides', 'audit_log',
    'record_backups', 'external_request_log', 'provider_circuit_breakers',
    'resource_registry', 'extension_inventory', 'event_outbox',
    'sheet_publish_log', 'system_controls', 'feature_flags',
    'feature_flag_targets'
  )
order by table_name;
```

### Inspect Columns Without Reading Data

```sql
select table_name, ordinal_position, column_name, data_type, is_nullable, column_default
from information_schema.columns
where table_schema = 'public'
order by table_name, ordinal_position;
```

### Inspect Primary, Unique, Foreign-Key, And Check Constraints

```sql
select
  tc.table_name,
  tc.constraint_name,
  tc.constraint_type,
  kcu.column_name,
  ccu.table_name as referenced_table,
  ccu.column_name as referenced_column
from information_schema.table_constraints tc
left join information_schema.key_column_usage kcu
  on tc.constraint_catalog = kcu.constraint_catalog
 and tc.constraint_schema = kcu.constraint_schema
 and tc.constraint_name = kcu.constraint_name
left join information_schema.constraint_column_usage ccu
  on tc.constraint_catalog = ccu.constraint_catalog
 and tc.constraint_schema = ccu.constraint_schema
 and tc.constraint_name = ccu.constraint_name
where tc.table_schema = 'public'
order by tc.table_name, tc.constraint_type, tc.constraint_name, kcu.ordinal_position;
```

### Inspect Defaults And Nullability For Review-Critical Columns

```sql
select table_name, column_name, is_nullable, column_default
from information_schema.columns
where table_schema = 'public'
  and table_name in (
    'workflow_runs', 'jobs', 'event_outbox', 'external_request_log',
    'system_controls', 'feature_flags', 'feature_flag_targets'
  )
  and (
    column_name like '%state%'
    or column_name in (
      'global_tick', 'attempt_count', 'run_after', 'approval_required',
      'lease_owner', 'lease_expires_at', 'lease_heartbeat_at',
      'job_family', 'scheduled_for', 'deferred_from_run_id',
      'overlap_count', 'payload', 'desired_snapshot', 'response_summary',
      'enabled', 'default_enabled'
    )
  )
order by table_name, column_name;
```

### Inspect Review-Critical Indexes

```sql
select
  schemaname,
  tablename,
  indexname,
  indexdef
from pg_indexes
where schemaname = 'public'
  and indexname in (
    'known_identifiers_source_unique',
    'workflow_runs_scheduled_family_active_idx',
    'workflow_runs_scheduled_family_overlap_idx',
    'jobs_claimable_global_tick_idx',
    'jobs_expired_lease_global_tick_idx',
    'external_request_log_idempotency_key_unique',
    'external_request_log_job_outcome_idx'
  )
order by tablename, indexname;
```

### Summarize Workflow, Job, And Approval State Counts

```sql
select 'workflow_runs.status' as field, status as value, count(*) as row_count
from workflow_runs
group by status
union all
select 'workflow_runs.approval_state', approval_state, count(*)
from workflow_runs
group by approval_state
union all
select 'jobs.job_state', job_state, count(*)
from jobs
group by job_state
union all
select 'approval_requests.approval_state', approval_state, count(*)
from approval_requests
group by approval_state
order by field, value;
```

### Confirm Global Tick Ordering

```sql
select id, global_tick, workflow_run_id, provider, operation, job_state, created_at
from jobs
order by global_tick asc
limit 50;
```

```sql
select id, global_tick, topic, event_type, created_at
from event_outbox
order by global_tick asc
limit 50;
```

### Check Tick Interleaving Across Jobs And Outbox Events

```sql
select 'job' as row_kind, id::text as row_id, global_tick, job_state as state, created_at
from jobs
union all
select 'event', id::text, global_tick, event_type, created_at
from event_outbox
order by global_tick asc
limit 100;
```

### Find Claimable Jobs

```sql
select id, global_tick, workflow_run_id, provider, operation, attempt_count, run_after
from jobs
where job_state = 'queued'
  and approval_required = false
  and (run_after is null or run_after <= now())
order by global_tick asc
limit 20;
```

### Review Running Leases

```sql
select
  id,
  global_tick,
  workflow_run_id,
  provider,
  operation,
  attempt_count,
  lease_owner,
  lease_expires_at,
  lease_heartbeat_at,
  updated_at
from jobs
where job_state = 'running'
order by lease_expires_at nulls last, global_tick asc;
```

### Find Expired Or Recovering Jobs

```sql
select
  id,
  global_tick,
  workflow_run_id,
  provider,
  operation,
  job_state,
  attempt_count,
  lease_owner,
  lease_expires_at,
  lease_heartbeat_at
from jobs
where (job_state = 'running' and lease_expires_at < now())
   or job_state = 'recovering'
order by global_tick asc;
```

### Check Recovery Evidence Before Requeue

```sql
select
  j.id as job_id,
  j.global_tick,
  j.job_state,
  j.attempt_count,
  erl.provider,
  erl.operation,
  erl.outcome,
  erl.created_at as request_logged_at
from jobs j
left join external_request_log erl
  on erl.job_id = j.id
 and erl.outcome = 'succeeded'
where j.job_state = 'recovering'
order by j.global_tick asc, erl.created_at asc;
```

### Review Scheduled Family Overlap Evidence

```sql
select
  id,
  workflow_type,
  subject_kind,
  subject_id,
  status,
  job_family,
  scheduled_for,
  deferred_from_run_id,
  overlap_state,
  overlap_count,
  created_at
from workflow_runs
where trigger_type = 'scheduled'
order by job_family, created_at asc;
```

### Find Active Scheduled Runs By Family

```sql
select id, job_family, status, scheduled_for, created_at, updated_at
from workflow_runs
where trigger_type = 'scheduled'
  and status in ('planned', 'running', 'recovering', 'waiting_manual')
order by job_family, created_at asc;
```

### Inspect Idempotency Log Without Raw Payloads

```sql
select
  id,
  job_id,
  provider,
  operation,
  left(idempotency_key, 12) || '...' as idempotency_key_prefix,
  left(request_hash, 12) || '...' as request_hash_prefix,
  provider_object_id,
  outcome,
  created_at
from external_request_log
order by created_at desc
limit 50;
```

### Review Audit Activity Without Diff Payloads

```sql
select
  id,
  actor_type,
  actor_id,
  request_id,
  target_entity,
  target_id,
  reason,
  created_at
from audit_log
order by created_at desc
limit 50;
```

### Check Global Pause Or Other System Controls

```sql
select control_name, enabled, reason, actor_id, updated_at
from system_controls
order by control_name;
```

### Review Feature-Flag Targets Without Reading User Data

```sql
select
  ff.flag_key,
  ff.feature_route,
  ff.default_enabled,
  fft.target_type,
  fft.target_id,
  fft.enabled,
  fft.updated_at
from feature_flags ff
left join feature_flag_targets fft
  on fft.flag_key = ff.flag_key
order by ff.flag_key, fft.target_type, fft.target_id;
```

## Sensitive-Field Handling

Do not copy these values into GitHub issues, PR descriptions, logs, fixtures,
screenshots, generated artifacts, support tickets, or committed docs:

- secrets, tokens, auth headers, passwords, private keys, client secrets, raw
  service-account JSON, refresh tokens, bearer tokens, and provider credentials
- raw provider request or response bodies
- unredacted `source_records.payload`
- unredacted `manual_overrides.diff`, `audit_log.diff`, or
  `record_backups.snapshot`
- personal phone numbers from manual onboarding or provider payloads
- employee numbers, generated employee numbers, source ids, student-like ids, or
  provider object ids unless the evidence destination explicitly allows them
- raw breakglass tokens, token hashes, or CIDR-bypass context beyond sanitized
  account/source/outcome evidence

Allowed reviewer evidence should prefer:

- row counts and state counts
- table, column, index, and constraint names
- `global_tick` values when needed for ordering proof
- non-secret ids that are already approved for the evidence surface
- one-way hash prefixes or redacted id prefixes when uniqueness evidence is
  needed
- sanitized state transitions, timestamps, and outcome labels

## Schema Alignment Checklist

Use this checklist for every Phase 0 schema or migration PR:

- Update `internal/db/schema.sql` for new databases.
- Add an idempotent migration under `internal/db/migrations/` when existing dev
  or staging databases need the change.
- Update `internal/db/schema_test.go` when a new table, field, constraint, or
  index becomes part of the Phase 0 contract.
- Update this guide when reviewers need to understand new fields, sensitive data
  handling, lifecycle states, retry/lease semantics, or SQL evidence queries.
- Update `docs/planning/external-write-inventory.md` when a database write path
  changes or a new write path is added.
- Link the related Phase 0 issue or PR back to GitHub issue #282 and this guide
  when the change touches database schema or review queries.
- Run the repo-local documentation and schema checks that apply to the changed
  files, plus `git diff --check`.
