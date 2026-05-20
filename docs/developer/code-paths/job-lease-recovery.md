# Job Lease Recovery And Scheduled Overlap Protection

This walkthrough explains the Phase 0 worker-crash recovery primitive for `P0-0B-001` and the scheduled job-family overlap primitive for `P0-0B-003`. It covers local database state only. Provider read-before-write checks remain required before any later live provider worker mutates an upstream system.

## Tables

- `jobs` stores the durable work item, ordered by `global_tick`.
- `jobs.lease_owner`, `jobs.lease_expires_at`, and `jobs.lease_heartbeat_at` record the worker lease.
- `external_request_log` stores idempotency and provider-effect evidence. During Phase 0 tests, a `succeeded` row stands in for a provider effect that was already completed before the worker crashed.
- `workflow_runs.job_family` names the scheduled provider or sync family that must not overlap itself.
- `workflow_runs.scheduled_for` records the cadence window that caused the run.
- `workflow_runs.deferred_from_run_id` points from an overlap-deferred run to the active run that blocked it.
- `workflow_runs.overlap_state` records `none` for active scheduled runs and `deferred_due_to_active_run` for suppressed duplicate starts.
- `workflow_runs.overlap_count` records the family-scoped overlap sequence used by later cadence-adjustment ticket work.

## Claim Path

`internal/db.ClaimNextJob` is the worker claim boundary.

1. The caller starts a transaction through `internal/db.WithRetry`.
2. `ClaimNextJob` selects the oldest eligible `queued` job by `global_tick` with `FOR UPDATE SKIP LOCKED`.
3. The same statement moves the row to `running`, writes the lease owner, expiry, heartbeat, and `updated_at`, then returns the claimed job.
4. If no row is available, the function returns `ErrNoJobAvailable` so an idle worker can sleep without treating the empty queue as a failure.

This keeps duplicate claims out of normal worker execution. Concurrent workers lock different rows, and ordering depends on `global_tick` rather than UUIDs or timestamps.

## Crash Recovery Path

`internal/db.RecoverExpiredJobLeases` is the recovery-loop boundary.

1. The caller starts a transaction through `internal/db.WithRetry`.
2. The function selects expired `running` jobs with `FOR UPDATE SKIP LOCKED`.
3. It moves those rows to `recovering`, clears claim ownership, and returns evidence fields including previous owner, nullable expired lease time, nullable heartbeat time, provider, operation, and step key.
4. The job is not immediately executable again. The recovery loop must reconcile first.
5. If a previous recovery loop stopped after moving a row to `recovering` but before reconciliation, the next call returns that already-`recovering` row again with missing lease evidence preserved as null.

## Reconciliation Path

`internal/db.ReconcileRecoveredJob` decides whether recovery can finish without duplicate execution.

1. It checks `external_request_log` for a `succeeded` row tied to the job id.
2. If success evidence exists, it marks the job `succeeded`, clears lease fields, and does not increment `attempt_count`.
3. If no success evidence exists, it marks the job `queued`, clears lease fields, and increments `attempt_count`.
4. A later worker claim may retry the queued job through the normal idempotent execution path.

This behavior covers both crash-after-claim cases:

- crash before a provider effect was recorded: the job is safely reclaimed for retry
- crash after a provider effect was recorded: the job is reconciled as succeeded and does not execute again

## Scheduled Job-Family Overlap Path

`internal/db.StartScheduledWorkflowRun` is the scheduled cadence boundary for `P0-0B-003`.

1. The caller starts a transaction through `internal/db.WithRetry`.
2. The function acquires `pg_advisory_xact_lock(hashtextextended(job_family, 0))`, which serializes schedule-start decisions for the same family even when no active row exists yet.
3. It looks for an active scheduled `workflow_runs` row for that family with status `planned`, `running`, `recovering`, or `waiting_manual`.
4. If no active row exists, it inserts a new `workflow_runs` row with `trigger_type = scheduled`, `status = running`, `overlap_state = none`, and `overlap_count = 0`.
5. If an active row exists, it inserts a separate `workflow_runs` row with `status = deferred`, `deferred_from_run_id` set to the active row id, `overlap_state = deferred_due_to_active_run`, and the next family `overlap_count`.
6. The deferred insert does not update the active row, lease fields, job rows, or provider state. The in-flight run keeps ownership until a later completion or recovery path handles it.

This behavior covers the duplicate scheduled-start case:

- first scheduled tick: records one active family owner
- second scheduled tick while the first is still active: records deferred overlap evidence and leaves the first run untouched

## Verification

The focused tests are in `internal/db/jobs_test.go`:

- idle scheduled family creates one running workflow run
- overlapping scheduled family creates a deferred workflow run tied to the active run
- claim writes the running lease fields
- an empty queue returns `ErrNoJobAvailable`
- expired running jobs move to `recovering`
- interrupted `recovering` rows are returned for reconciliation on the next pass
- recovered jobs with `external_request_log.outcome = 'succeeded'` complete without retry
- recovered jobs without a success log return to `queued` with an attempt bump

Staging evidence for `P0-0B-001` should run the same state transitions against the staging database and record before/after `jobs` and `external_request_log` rows in the external IncidentIQ evidence ticket.

Staging evidence for `P0-0B-003` should run two schedule-start attempts for the same `job_family` against staging. The evidence should show the first row as active, the second row as `deferred`, `deferred_from_run_id` pointing to the first row, `overlap_state = deferred_due_to_active_run`, and no provider write or active-row clobber. The repository must not store staging credentials, raw provider responses, or production-derived snapshots.
