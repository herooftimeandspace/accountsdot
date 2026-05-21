# Global Tick Ordering

This walkthrough explains the Phase 0 ordering primitive for `P0-0B-002`. It covers local database sequencing for jobs and outbox events. It does not approve provider writeback, production writes, or manual database repair.

## Tables

- `global_tick_seq` is the shared sequence that assigns durable order across workflow tables.
- `jobs.global_tick` records job order independently of `jobs.id`, UUID-backed entities, timestamps, or provider identifiers.
- `event_outbox.global_tick` records event order independently of `event_outbox.id`, payload contents, timestamps, or future publisher-specific identifiers.
- `jobs_claimable_global_tick_idx` supports worker claims that scan queued jobs by `global_tick`.
- `event_outbox_global_tick_idx` supports outbox readers that scan events by `global_tick`.

## Job Claim Ordering

`internal/db.ClaimNextJob` is the worker claim boundary.

1. The caller starts a transaction through `internal/db.WithRetry`.
2. `ClaimNextJob` finds eligible queued jobs with `job_state = queued`, `approval_required = false`, and a due `run_after`.
3. The claim statement orders candidates by `global_tick asc`.
4. The statement locks the selected row with `FOR UPDATE SKIP LOCKED`, moves it to `running`, writes lease fields, and returns the claimed row.

The returned `LeasedJob.ID` identifies the row to execute, but it is not an ordering signal. Future worker loops must keep ordering decisions on `LeasedJob.GlobalTick`.

## Recovery Ordering

`internal/db.RecoverExpiredJobLeases` is the recovery-loop boundary.

1. The caller starts a transaction through `internal/db.WithRetry`.
2. The function finds expired `running` jobs and already-`recovering` jobs.
3. Candidate rows are ordered by `global_tick asc`.
4. The function locks those rows with `FOR UPDATE SKIP LOCKED`, moves expired running rows to `recovering`, and returns evidence rows for reconciliation.

This keeps concurrent recovery workers from using UUIDs, ids, or timestamps to choose work. Recovery order follows the same shared sequence as worker claims.

## Event Outbox Ordering

`internal/db.ListOutboxEvents` is the current outbox read boundary.

1. The caller passes a database executor and a positive limit.
2. The function reads `event_outbox` rows ordered by `global_tick asc`.
3. The returned `OutboxEvent.ID` identifies the row, while `OutboxEvent.GlobalTick` is the strict sequencing value.
4. The function is read-only. It does not publish, acknowledge, delete, or mutate provider-facing state.

Future publisher loops that add acknowledgement or delivery state must keep selection order on `global_tick` and must document any new database write path in `docs/planning/external-write-inventory.md`.

## Verification

The focused dev mock evidence is in `internal/db/jobs_test.go`:

- `TestClaimNextJobWritesLeaseFields` verifies worker claim SQL uses `order by global_tick asc` and rejects identifier or timestamp ordering.
- `TestRecoverExpiredJobLeasesMovesRunningJobsToRecovering` verifies recovery SQL uses `order by global_tick asc` and rejects identifier or timestamp ordering.
- `TestListOutboxEventsUsesGlobalTickOrdering` verifies outbox read SQL uses `order by global_tick asc` while scripted event ids intentionally do not match tick order.

Run the focused check with:

```bash
go test ./internal/db
```

Staging evidence for `P0-0B-002` should run the same job and outbox ordering checks against staging infrastructure with concurrent inserts. Record the inserted `jobs.global_tick` and `event_outbox.global_tick` values, the selected order, and the checked-out revision in the external IncidentIQ evidence ticket. Do not store staging credentials, raw provider responses, or production-derived snapshots in this repository.
