create extension if not exists pgcrypto;

create sequence if not exists global_tick_seq;

create table if not exists people (
    uuid uuid primary key,
    person_state text not null,
    first_name text,
    last_name text,
    email text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists employees (
    people_uuid uuid primary key references people(uuid) on delete cascade,
    employee_number text not null unique
);

create table if not exists contractors (
    people_uuid uuid primary key references people(uuid) on delete cascade,
    generated_employee_number text not null unique
);

create table if not exists external_volunteers (
    people_uuid uuid primary key references people(uuid) on delete cascade,
    generated_employee_number text not null unique
);

create table if not exists source_records (
    id bigserial primary key,
    people_uuid uuid not null references people(uuid) on delete cascade,
    source_system text not null,
    source_id text not null,
    payload jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table if not exists known_identifiers (
    people_uuid uuid not null references people(uuid) on delete cascade,
    source_system text not null,
    source_id text not null,
    last_seen_at timestamptz not null,
    primary key (people_uuid, source_system, source_id)
);

create unique index if not exists known_identifiers_source_unique
    on known_identifiers (source_system, source_id);

create table if not exists user_sync_status (
    user_id text not null,
    user_type text not null,
    school_year text not null,
    people_uuid uuid references people(uuid) on delete set null,
    display_name text not null,
    site_code text,
    current_phase text not null,
    overall_status text not null,
    queued_at timestamptz not null default now(),
    last_job_date timestamptz,
    completion_date timestamptz,
    completion_summary text,
    errors_warnings jsonb not null default '[]'::jsonb,
    is_archived boolean not null default false,
    archived_at timestamptz,
    primary key (user_type, user_id, school_year)
);

create table if not exists room_mapping_overrides (
    id bigserial primary key,
    school_year text not null,
    source_room text not null,
    normalized_room text not null,
    incident_iq_room_id text not null,
    incident_iq_room_name text,
    actor_id text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (school_year, source_room)
);

create table if not exists import_batches (
    id bigserial primary key,
    source_system text not null,
    source_fingerprint text not null,
    status text not null,
    row_count integer not null default 0,
    failure_summary text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists workflow_runs (
    id bigserial primary key,
    workflow_type text not null,
    subject_kind text not null,
    subject_id text not null,
    trigger_type text not null,
    status text not null,
    job_family text not null default 'unclassified',
    scheduled_for timestamptz,
    deferred_from_run_id bigint references workflow_runs(id) on delete set null,
    overlap_state text not null default 'none',
    overlap_count integer not null default 0,
    approval_state text not null default 'not_required',
    desired_snapshot jsonb not null default '{}'::jsonb,
    source_batch_id bigint references import_batches(id) on delete set null,
    current_job_count integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists workflow_runs_scheduled_family_active_idx
    on workflow_runs (job_family, created_at)
    where trigger_type = 'scheduled'
      and status in ('planned', 'running', 'recovering', 'waiting_manual');

create index if not exists workflow_runs_scheduled_family_overlap_idx
    on workflow_runs (job_family, overlap_count, created_at)
    where trigger_type = 'scheduled'
      and overlap_state <> 'none';

create table if not exists jobs (
    id bigserial primary key,
    global_tick bigint not null default nextval('global_tick_seq'),
    workflow_run_id bigint references workflow_runs(id) on delete cascade,
    people_uuid uuid references people(uuid) on delete cascade,
    job_state text not null,
    job_type text not null,
    provider text not null default 'internal',
    operation text not null default 'internal.noop',
    step_key text,
    depends_on_step_key text,
    attempt_count integer not null default 0,
    run_after timestamptz,
    approval_required boolean not null default false,
    reason_code text,
    lease_owner text,
    lease_expires_at timestamptz,
    lease_heartbeat_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists jobs_claimable_global_tick_idx
    on jobs (global_tick)
    where job_state = 'queued' and approval_required = false;

create index if not exists jobs_expired_lease_global_tick_idx
    on jobs (lease_expires_at, global_tick)
    where job_state = 'running' and lease_expires_at is not null;

create table if not exists approval_requests (
    id bigserial primary key,
    workflow_run_id bigint not null references workflow_runs(id) on delete cascade,
    job_id bigint references jobs(id) on delete set null,
    approval_state text not null,
    reason_code text,
    requested_at timestamptz not null default now(),
    decided_at timestamptz,
    decided_by text
);

create table if not exists manual_overrides (
    id bigserial primary key,
    people_uuid uuid references people(uuid) on delete cascade,
    target_user_type text,
    target_user_id text,
    school_year text,
    actor_id text not null,
    reason text not null,
    diff jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table if not exists audit_log (
    id bigserial primary key,
    actor_id text not null,
    actor_type text not null,
    request_id text,
    target_entity text not null,
    target_id text not null,
    reason text not null,
    diff jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table if not exists auth_site_scope_mappings (
    id bigserial primary key,
    source_type text not null check (source_type in ('group', 'ou', 'attribute')),
    source_value text not null,
    attribute_values jsonb not null default '[]'::jsonb,
    site_codes jsonb not null default '[]'::jsonb,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (source_type, source_value)
);

create index if not exists auth_site_scope_mappings_source_idx
    on auth_site_scope_mappings (source_type, source_value);

create table if not exists auth_role_mappings (
    id bigserial primary key,
    source_type text not null check (source_type in ('group', 'ou', 'attribute')),
    source_value text not null,
    attribute_values jsonb not null default '[]'::jsonb,
    role_keys jsonb not null default '[]'::jsonb,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (source_type, source_value)
);

create index if not exists auth_role_mappings_source_idx
    on auth_role_mappings (source_type, source_value);

create table if not exists external_data_sources (
    provider_key text primary key,
    provider_label text not null,
    sync_enabled boolean not null default false,
    last_test_status text,
    last_test_summary text,
    last_test_at timestamptz,
    actor_id text not null default 'system',
    reason text not null default 'registry_default_off',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists external_provider_credentials (
    id bigserial primary key,
    provider_key text not null references external_data_sources(provider_key) on delete cascade,
    field_key text not null,
    encrypted_value text not null,
    key_id text not null,
    fingerprint text not null,
    label text not null,
    actor_id text not null,
    reason text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (provider_key, field_key)
);

create index if not exists external_provider_credentials_provider_idx
    on external_provider_credentials (provider_key);

create table if not exists record_backups (
    id bigserial primary key,
    target_table text not null,
    target_id text not null,
    snapshot jsonb not null,
    created_at timestamptz not null default now()
);

create table if not exists external_request_log (
    id bigserial primary key,
    job_id bigint references jobs(id) on delete set null,
    provider text not null,
    operation text not null,
    idempotency_key text not null,
    request_hash text not null,
    provider_object_id text,
    outcome text not null,
    response_summary jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create unique index if not exists external_request_log_idempotency_key_unique
    on external_request_log (provider, operation, idempotency_key);

create index if not exists external_request_log_job_outcome_idx
    on external_request_log (job_id, outcome);

create table if not exists provider_circuit_breakers (
    provider text not null,
    operation_class text not null,
    state text not null,
    failure_count integer not null default 0,
    opened_at timestamptz,
    next_probe_at timestamptz,
    primary key (provider, operation_class)
);

create table if not exists resource_registry (
    id bigserial primary key,
    room_key text not null unique,
    site_code text not null,
    room_resource_state text not null,
    provider_type text not null,
    provider_object_id text,
    assigned_people_uuid uuid references people(uuid) on delete set null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists extension_inventory (
    extension text primary key,
    site_code text not null,
    status text not null,
    reserved_for_job_id bigint references jobs(id) on delete set null,
    assigned_to_people_uuid uuid references people(uuid) on delete set null,
    updated_at timestamptz not null default now()
);

create table if not exists event_outbox (
    id bigserial primary key,
    global_tick bigint not null default nextval('global_tick_seq'),
    topic text not null,
    event_type text not null,
    payload jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists event_outbox_global_tick_idx
    on event_outbox (global_tick);

create table if not exists sheet_publish_log (
    id bigserial primary key,
    tab_name text not null,
    staging_sheet text not null,
    checksum text not null,
    row_count integer not null,
    publish_version bigint not null,
    sentinel_validated boolean not null default false,
    pointer_applied boolean not null default false,
    created_at timestamptz not null default now()
);

create table if not exists system_controls (
    control_name text primary key,
    enabled boolean not null default false,
    reason text,
    actor_id text,
    updated_at timestamptz not null default now()
);

create table if not exists feature_flags (
    flag_key text primary key,
    label text not null,
    description text not null,
    feature_route text not null,
    default_enabled boolean not null default true,
    actor_id text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists feature_flag_targets (
    flag_key text not null references feature_flags(flag_key) on delete cascade,
    target_type text not null check (target_type in ('persona', 'site')),
    target_id text not null,
    enabled boolean not null,
    actor_id text,
    updated_at timestamptz not null default now(),
    primary key (flag_key, target_type, target_id)
);
