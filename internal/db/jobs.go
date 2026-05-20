package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

// JobExecutor is the narrow transaction surface used by the Phase 0 job lease
// primitives. Production callers should pass a pgx.Tx from WithRetry so claims,
// lease recovery, and reconciliation share the repo's SERIALIZABLE retry
// behavior.
type JobExecutor interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// LeasedJob is the database row a worker receives after a successful claim.
// Workers use ID for execution and idempotency-key construction, while
// GlobalTick is the ordering signal that keeps work sequencing independent of
// UUID or timestamp ordering.
type LeasedJob struct {
	ID               int64
	GlobalTick       int64
	WorkflowRunID    int64
	Provider         core.ProviderKind
	Operation        string
	StepKey          string
	DependsOnStepKey string
	AttemptCount     int
	LeaseOwner       string
	LeaseExpiresAt   time.Time
	LeaseHeartbeatAt time.Time
}

// RecoveredJob identifies a running job whose lease expired and was moved into
// recovering. The recovery loop records these rows as runtime evidence and then
// calls ReconcileRecoveredJob before allowing any provider operation to run
// again.
type RecoveredJob struct {
	ID             int64
	GlobalTick     int64
	WorkflowRunID  int64
	PreviousOwner  string
	ExpiredAt      time.Time
	AttemptCount   int
	Provider       core.ProviderKind
	Operation      string
	StepKey        string
	LastHeartbeat  time.Time
	RecoveredAt    time.Time
	NeedsProvider  bool
	ReconcileState JobRecoveryState
}

type JobRecoveryState string

const (
	JobRecoveryStateRecoveredForRetry JobRecoveryState = "recovered_for_retry"
	JobRecoveryStateSucceeded         JobRecoveryState = "succeeded"
)

var ErrNoJobAvailable = errors.New("no job available for claim")

// ClaimNextJob atomically claims the oldest runnable queued job and writes the
// worker lease fields that a later crash-recovery pass can inspect. The query
// uses FOR UPDATE SKIP LOCKED so multiple workers can ask for work without
// double-claiming the same global_tick-ordered job.
func ClaimNextJob(ctx context.Context, tx JobExecutor, owner string, leaseExpiresAt time.Time, now time.Time) (LeasedJob, error) {
	var job LeasedJob
	err := tx.QueryRow(ctx, `
update jobs
set job_state = $1,
    lease_owner = $2,
    lease_expires_at = $3,
    lease_heartbeat_at = $4,
    updated_at = $4
where id = (
    select id
    from jobs
    where job_state = $5
      and approval_required = false
      and (run_after is null or run_after <= $4)
    order by global_tick asc
    for update skip locked
    limit 1
)
returning id, global_tick, coalesce(workflow_run_id, 0), provider, operation,
          coalesce(step_key, ''), coalesce(depends_on_step_key, ''),
          attempt_count, lease_owner, lease_expires_at, lease_heartbeat_at
`, string(core.JobStateRunning), owner, leaseExpiresAt, now, string(core.JobStateQueued)).Scan(
		&job.ID,
		&job.GlobalTick,
		&job.WorkflowRunID,
		&job.Provider,
		&job.Operation,
		&job.StepKey,
		&job.DependsOnStepKey,
		&job.AttemptCount,
		&job.LeaseOwner,
		&job.LeaseExpiresAt,
		&job.LeaseHeartbeatAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LeasedJob{}, ErrNoJobAvailable
	}
	if err != nil {
		return LeasedJob{}, fmt.Errorf("claim next job: %w", err)
	}
	return job, nil
}

// RecoverExpiredJobLeases moves expired running leases to recovering and clears
// ownership before the job can be claimed again. It returns the affected rows so
// the recovery loop can reconcile external effects first and prove that a worker
// crash did not create duplicate execution.
func RecoverExpiredJobLeases(ctx context.Context, tx JobExecutor, now time.Time, limit int) ([]RecoveredJob, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
with expired as (
    select id, lease_owner, lease_expires_at, lease_heartbeat_at
    from jobs
    where job_state = $1
      and lease_expires_at is not null
      and lease_expires_at <= $2
    order by global_tick asc
    for update skip locked
    limit $3
)
update jobs
set job_state = $4,
    lease_owner = null,
    lease_expires_at = null,
    lease_heartbeat_at = null,
    updated_at = $2
from expired
where jobs.id = expired.id
returning jobs.id, jobs.global_tick, coalesce(jobs.workflow_run_id, 0),
          coalesce(expired.lease_owner, ''), expired.lease_expires_at,
          jobs.attempt_count,
          jobs.provider, jobs.operation, coalesce(jobs.step_key, ''),
          coalesce(expired.lease_heartbeat_at, $2::timestamptz)
`, string(core.JobStateRunning), now, limit, string(core.JobStateRecovering))
	if err != nil {
		return nil, fmt.Errorf("recover expired job leases: %w", err)
	}
	defer rows.Close()

	recovered := []RecoveredJob{}
	for rows.Next() {
		var job RecoveredJob
		if err := rows.Scan(
			&job.ID,
			&job.GlobalTick,
			&job.WorkflowRunID,
			&job.PreviousOwner,
			&job.ExpiredAt,
			&job.AttemptCount,
			&job.Provider,
			&job.Operation,
			&job.StepKey,
			&job.LastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("scan recovered job: %w", err)
		}
		job.RecoveredAt = now
		job.NeedsProvider = job.Provider != core.ProviderKindInternal
		job.ReconcileState = JobRecoveryStateRecoveredForRetry
		recovered = append(recovered, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recovered jobs: %w", err)
	}
	return recovered, nil
}

// ReconcileRecoveredJob finishes a recovering job without re-execution when an
// existing external_request_log row proves the provider effect already
// succeeded. If no success log exists, the job is made queued again with a
// bumped attempt count so the next worker claim follows the normal idempotent
// execution path.
func ReconcileRecoveredJob(ctx context.Context, tx JobExecutor, jobID int64, now time.Time) (JobRecoveryState, error) {
	var completed bool
	err := tx.QueryRow(ctx, `
select exists (
    select 1
    from external_request_log
    where job_id = $1
      and outcome = 'succeeded'
)
`, jobID).Scan(&completed)
	if err != nil {
		return "", fmt.Errorf("check recovered job external request log: %w", err)
	}

	nextState := core.JobStateQueued
	outcome := JobRecoveryStateRecoveredForRetry
	if completed {
		nextState = core.JobStateSucceeded
		outcome = JobRecoveryStateSucceeded
	}

	err = tx.QueryRow(ctx, `
update jobs
set job_state = $1,
    attempt_count = case when $2 then attempt_count else attempt_count + 1 end,
    lease_owner = null,
    lease_expires_at = null,
    lease_heartbeat_at = null,
    updated_at = $3
where id = $4
  and job_state = $5
returning id
`, string(nextState), completed, now, jobID, string(core.JobStateRecovering)).Scan(&jobID)
	if err != nil {
		return "", fmt.Errorf("update recovered job state: %w", err)
	}
	return outcome, nil
}
