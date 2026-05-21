-- Adds the Phase 0 scheduled job-family overlap fields to existing
-- workflow_runs tables. New databases receive the same columns and indexes from
-- internal/db/schema.sql; this migration preserves already-created dev or
-- staging databases that predate P0-0B-003.

alter table if exists workflow_runs
    add column if not exists job_family text not null default 'unclassified';

alter table if exists workflow_runs
    add column if not exists scheduled_for timestamptz;

alter table if exists workflow_runs
    add column if not exists deferred_from_run_id bigint references workflow_runs(id) on delete set null;

alter table if exists workflow_runs
    add column if not exists overlap_state text not null default 'none';

alter table if exists workflow_runs
    add column if not exists overlap_count integer not null default 0;

create index if not exists workflow_runs_scheduled_family_active_idx
    on workflow_runs (job_family, created_at)
    where trigger_type = 'scheduled'
      and status in ('planned', 'running', 'recovering', 'waiting_manual');

create index if not exists workflow_runs_scheduled_family_overlap_idx
    on workflow_runs (job_family, overlap_count, created_at)
    where trigger_type = 'scheduled'
      and overlap_state <> 'none';
