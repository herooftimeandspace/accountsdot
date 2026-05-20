package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

// JobExecutor is the narrow transaction surface used by the Phase 0 job lease
// primitives. Production callers should pass a pgx.Tx from WithRetry so claims,
// lease recovery, and reconciliation share the repo's SERIALIZABLE retry
// behavior.
type JobExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
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

// RecoveredJob identifies a job that needs recovery-loop reconciliation. Fresh
// rows come from expired running leases, while already-recovering rows can be
// returned after a prior loop was interrupted before ReconcileRecoveredJob ran.
type RecoveredJob struct {
	ID             int64
	GlobalTick     int64
	WorkflowRunID  int64
	PreviousOwner  string
	ExpiredAt      *time.Time
	AttemptCount   int
	Provider       core.ProviderKind
	Operation      string
	StepKey        string
	LastHeartbeat  *time.Time
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

const (
	scheduledTriggerType     = "scheduled"
	overlapStateNone         = "none"
	overlapStateDeferred     = "deferred_due_to_active_run"
	defaultDesiredSnapshot   = "{}"
	defaultScheduledApproval = string(core.ApprovalStateNotRequired)
)

// ScheduledWorkflowRunRequest is the scheduler input for one provider or sync
// job-family cadence tick. The scheduler supplies the durable family key and
// desired snapshot JSON; StartScheduledWorkflowRun stores either an active run
// or an overlap-deferred run in workflow_runs without touching provider state.
type ScheduledWorkflowRunRequest struct {
	WorkflowType        core.WorkflowType
	SubjectKind         core.SubjectKind
	SubjectID           string
	JobFamily           string
	ScheduledFor        time.Time
	DesiredSnapshotJSON string
}

// ScheduledWorkflowRunStart is the durable database outcome for a scheduled
// cadence tick. Deferred outcomes point back to the active run that blocked the
// launch and include the family overlap count used by later cadence-tuning
// ticket work.
type ScheduledWorkflowRunStart struct {
	RunID          int64
	ActiveRunID    int64
	JobFamily      string
	Status         core.WorkflowRunState
	OverlapState   string
	OverlapCount   int
	Deferred       bool
	ScheduledFor   time.Time
	WorkflowType   core.WorkflowType
	SubjectKind    core.SubjectKind
	SubjectID      string
	TriggerType    string
	DesiredPayload string
}

// StartScheduledWorkflowRun serializes a scheduled provider/job-family launch
// with a PostgreSQL transaction-level advisory lock, then records exactly one
// active scheduled run or a deferred overlap row. Callers must invoke it inside
// db.WithRetry so the advisory lock, active-run check, and insert commit or
// retry together when PostgreSQL reports serialization or deadlock failures.
func StartScheduledWorkflowRun(ctx context.Context, tx JobExecutor, req ScheduledWorkflowRunRequest, now time.Time) (ScheduledWorkflowRunStart, error) {
	if req.DesiredSnapshotJSON == "" {
		req.DesiredSnapshotJSON = defaultDesiredSnapshot
	}

	if _, err := tx.Exec(ctx, `
select pg_advisory_xact_lock(hashtextextended($1, 0))
`, req.JobFamily); err != nil {
		return ScheduledWorkflowRunStart{}, fmt.Errorf("lock scheduled workflow family: %w", err)
	}

	activeID, err := activeScheduledWorkflowRunID(ctx, tx, req.JobFamily)
	if err != nil {
		return ScheduledWorkflowRunStart{}, err
	}
	if activeID != 0 {
		return insertDeferredScheduledWorkflowRun(ctx, tx, req, activeID, now)
	}
	return insertActiveScheduledWorkflowRun(ctx, tx, req, now)
}

// activeScheduledWorkflowRunID locks and returns the current active scheduled
// run for a family after StartScheduledWorkflowRun has acquired the family
// advisory lock. A missing row is normal and lets the caller create the first
// active run for that cadence window.
func activeScheduledWorkflowRunID(ctx context.Context, tx JobExecutor, jobFamily string) (int64, error) {
	var activeID int64
	err := tx.QueryRow(ctx, `
select id
from workflow_runs
where job_family = $1
  and trigger_type = $2
  and status in ($3, $4, $5, $6)
order by created_at asc
for update
limit 1
`, jobFamily, scheduledTriggerType,
		string(core.WorkflowRunStatePlanned),
		string(core.WorkflowRunStateRunning),
		string(core.WorkflowRunStateRecovering),
		string(core.WorkflowRunStateWaitingManual),
	).Scan(&activeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find active scheduled workflow run: %w", err)
	}
	return activeID, nil
}

// insertActiveScheduledWorkflowRun creates the row a worker family may execute
// now. The inserted row starts in running state so a later scheduled tick can
// see it as active and defer instead of creating duplicate provider work.
func insertActiveScheduledWorkflowRun(ctx context.Context, tx JobExecutor, req ScheduledWorkflowRunRequest, now time.Time) (ScheduledWorkflowRunStart, error) {
	start := ScheduledWorkflowRunStart{
		JobFamily:      req.JobFamily,
		Status:         core.WorkflowRunStateRunning,
		OverlapState:   overlapStateNone,
		ScheduledFor:   req.ScheduledFor,
		WorkflowType:   req.WorkflowType,
		SubjectKind:    req.SubjectKind,
		SubjectID:      req.SubjectID,
		TriggerType:    scheduledTriggerType,
		DesiredPayload: req.DesiredSnapshotJSON,
	}
	err := tx.QueryRow(ctx, `
insert into workflow_runs (
    workflow_type,
    subject_kind,
    subject_id,
    trigger_type,
    status,
    job_family,
    scheduled_for,
    overlap_state,
    overlap_count,
    approval_state,
    desired_snapshot,
    created_at,
    updated_at
) values ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, $10::jsonb, $11, $11)
returning id
`, req.WorkflowType, req.SubjectKind, req.SubjectID, scheduledTriggerType,
		string(core.WorkflowRunStateRunning), req.JobFamily, req.ScheduledFor,
		overlapStateNone, defaultScheduledApproval, req.DesiredSnapshotJSON, now,
	).Scan(&start.RunID)
	if err != nil {
		return ScheduledWorkflowRunStart{}, fmt.Errorf("insert active scheduled workflow run: %w", err)
	}
	start.ActiveRunID = start.RunID
	return start, nil
}

// insertDeferredScheduledWorkflowRun records an overlap without clobbering the
// active family run. It stores the blocking run id and a family-scoped overlap
// count so Phase 4 cadence-ticketing work can build on durable history rather
// than log scraping.
func insertDeferredScheduledWorkflowRun(ctx context.Context, tx JobExecutor, req ScheduledWorkflowRunRequest, activeID int64, now time.Time) (ScheduledWorkflowRunStart, error) {
	start := ScheduledWorkflowRunStart{
		ActiveRunID:    activeID,
		JobFamily:      req.JobFamily,
		Status:         core.WorkflowRunStateDeferred,
		OverlapState:   overlapStateDeferred,
		Deferred:       true,
		ScheduledFor:   req.ScheduledFor,
		WorkflowType:   req.WorkflowType,
		SubjectKind:    req.SubjectKind,
		SubjectID:      req.SubjectID,
		TriggerType:    scheduledTriggerType,
		DesiredPayload: req.DesiredSnapshotJSON,
	}
	err := tx.QueryRow(ctx, `
insert into workflow_runs (
    workflow_type,
    subject_kind,
    subject_id,
    trigger_type,
    status,
    job_family,
    scheduled_for,
    deferred_from_run_id,
    overlap_state,
    overlap_count,
    approval_state,
    desired_snapshot,
    created_at,
    updated_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    coalesce((select max(overlap_count) from workflow_runs where job_family = $6), 0) + 1,
    $10, $11::jsonb, $12, $12
)
returning id, overlap_count
`, req.WorkflowType, req.SubjectKind, req.SubjectID, scheduledTriggerType,
		string(core.WorkflowRunStateDeferred), req.JobFamily, req.ScheduledFor,
		activeID, overlapStateDeferred, defaultScheduledApproval, req.DesiredSnapshotJSON, now,
	).Scan(&start.RunID, &start.OverlapCount)
	if err != nil {
		return ScheduledWorkflowRunStart{}, fmt.Errorf("insert deferred scheduled workflow run: %w", err)
	}
	return start, nil
}

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

// RecoverExpiredJobLeases moves expired running leases to recovering and also
// returns already-recovering rows left behind by an interrupted recovery loop.
// The caller records the evidence rows, then calls ReconcileRecoveredJob before
// the job can be claimed again or treated as already completed.
func RecoverExpiredJobLeases(ctx context.Context, tx JobExecutor, now time.Time, limit int) ([]RecoveredJob, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
	with candidates as (
	    select id, lease_owner, lease_expires_at, lease_heartbeat_at
	    from jobs
	    where (job_state = $1
	           and lease_expires_at is not null
	           and lease_expires_at <= $2)
	       or job_state = $4
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
	from candidates
	where jobs.id = candidates.id
	returning jobs.id, jobs.global_tick, coalesce(jobs.workflow_run_id, 0),
	          coalesce(candidates.lease_owner, ''), candidates.lease_expires_at,
	          jobs.attempt_count,
	          jobs.provider, jobs.operation, coalesce(jobs.step_key, ''),
	          candidates.lease_heartbeat_at
	`, string(core.JobStateRunning), now, limit, string(core.JobStateRecovering))
	if err != nil {
		return nil, fmt.Errorf("recover expired job leases: %w", err)
	}
	defer rows.Close()

	recovered := []RecoveredJob{}
	for rows.Next() {
		var job RecoveredJob
		var expiredAt sql.NullTime
		var lastHeartbeat sql.NullTime
		if err := rows.Scan(
			&job.ID,
			&job.GlobalTick,
			&job.WorkflowRunID,
			&job.PreviousOwner,
			&expiredAt,
			&job.AttemptCount,
			&job.Provider,
			&job.Operation,
			&job.StepKey,
			&lastHeartbeat,
		); err != nil {
			return nil, fmt.Errorf("scan recovered job: %w", err)
		}
		if expiredAt.Valid {
			job.ExpiredAt = &expiredAt.Time
		}
		if lastHeartbeat.Valid {
			job.LastHeartbeat = &lastHeartbeat.Time
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
