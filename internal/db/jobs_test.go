package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/core"
)

type jobQuery struct {
	sql  string
	args []any
}

type fakeJobExecutor struct {
	queryRows []pgx.Rows
	rowScans  [][]any
	rowErrs   []error
	queries   []jobQuery
}

// Query records job-recovery SQL issued by tests so they can prove the
// implementation uses the recovery and reconciliation tables instead of an
// in-memory shortcut.
func (f *fakeJobExecutor) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queries = append(f.queries, jobQuery{sql: sql, args: args})
	if len(f.queryRows) == 0 {
		return nil, errors.New("unexpected Query call")
	}
	rows := f.queryRows[0]
	f.queryRows = f.queryRows[1:]
	return rows, nil
}

// QueryRow records claim and reconciliation updates, then returns the next
// scripted row result for the test scenario.
func (f *fakeJobExecutor) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.queries = append(f.queries, jobQuery{sql: sql, args: args})
	if len(f.rowErrs) > 0 {
		err := f.rowErrs[0]
		f.rowErrs = f.rowErrs[1:]
		return fakeJobRow{err: err}
	}
	if len(f.rowScans) == 0 {
		return fakeJobRow{err: errors.New("unexpected QueryRow call")}
	}
	values := f.rowScans[0]
	f.rowScans = f.rowScans[1:]
	return fakeJobRow{values: values}
}

type fakeJobRow struct {
	values []any
	err    error
}

// Scan copies scripted row values into destination pointers, matching the pgx
// Row contract closely enough for the lease-recovery unit tests.
func (r fakeJobRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan destination count mismatch")
	}
	for i := range dest {
		assignFakeScanValue(dest[i], r.values[i])
	}
	return nil
}

type fakeJobRows struct {
	values [][]any
	pos    int
	err    error
	closed bool
}

// Close marks the fake row set closed so RecoverExpiredJobLeases follows the
// same cleanup path as it would with a live pgx.Rows value.
func (r *fakeJobRows) Close() { r.closed = true }

// Err returns the scripted iteration error after all rows have been consumed.
func (r *fakeJobRows) Err() error { return r.err }

// CommandTag satisfies pgx.Rows for tests that only need row scanning.
func (r *fakeJobRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

// FieldDescriptions satisfies pgx.Rows for tests that only need row scanning.
func (r *fakeJobRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

// Next advances through scripted recovered jobs.
func (r *fakeJobRows) Next() bool {
	if r.pos >= len(r.values) {
		r.closed = true
		return false
	}
	r.pos++
	return true
}

// Scan copies the current scripted recovered-job row into destination pointers.
func (r *fakeJobRows) Scan(dest ...any) error {
	if r.pos == 0 || r.pos > len(r.values) {
		return errors.New("scan without current row")
	}
	values := r.values[r.pos-1]
	if len(dest) != len(values) {
		return errors.New("scan destination count mismatch")
	}
	for i := range dest {
		assignFakeScanValue(dest[i], values[i])
	}
	return nil
}

// Values exposes the current scripted row for pgx.Rows compatibility.
func (r *fakeJobRows) Values() ([]any, error) {
	if r.pos == 0 || r.pos > len(r.values) {
		return nil, errors.New("values without current row")
	}
	return r.values[r.pos-1], nil
}

// RawValues is unused by the tests but required by pgx.Rows.
func (r *fakeJobRows) RawValues() [][]byte { return nil }

// Conn is unused by the tests but required by pgx.Rows.
func (r *fakeJobRows) Conn() *pgx.Conn { return nil }

func assignFakeScanValue(dest any, value any) {
	switch d := dest.(type) {
	case *int64:
		*d = value.(int64)
	case *int:
		*d = value.(int)
	case *string:
		*d = value.(string)
	case *bool:
		*d = value.(bool)
	case *time.Time:
		*d = value.(time.Time)
	case *sql.NullTime:
		if value == nil {
			*d = sql.NullTime{}
		} else {
			*d = sql.NullTime{Time: value.(time.Time), Valid: true}
		}
	case *core.ProviderKind:
		*d = core.ProviderKind(value.(string))
	default:
		panic("unsupported fake scan destination")
	}
}

// TestClaimNextJobWritesLeaseFields exercises the claim half of
// P0-0B-001. It proves a worker claim moves a queued job to running and stores
// the lease owner, expiry, and heartbeat fields needed by crash recovery.
func TestClaimNextJobWritesLeaseFields(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	expires := now.Add(30 * time.Second)
	exec := &fakeJobExecutor{rowScans: [][]any{{
		int64(42),
		int64(1001),
		int64(7),
		"zoom",
		"zoom.create_user",
		"create-user",
		"",
		0,
		"worker-a",
		expires,
		now,
	}}}

	job, err := ClaimNextJob(context.Background(), exec, "worker-a", expires, now)
	if err != nil {
		t.Fatalf("ClaimNextJob returned error: %v", err)
	}
	if job.ID != 42 || job.GlobalTick != 1001 || job.LeaseOwner != "worker-a" {
		t.Fatalf("unexpected leased job: %#v", job)
	}
	if got := exec.queries[0]; !strings.Contains(got.sql, "for update skip locked") || got.args[0] != string(core.JobStateRunning) {
		t.Fatalf("claim query did not lock and mark running: %#v", got)
	}
}

// TestClaimNextJobReturnsNoJobAvailable verifies idle workers receive a stable
// sentinel instead of treating an empty queue as a database failure.
func TestClaimNextJobReturnsNoJobAvailable(t *testing.T) {
	exec := &fakeJobExecutor{rowErrs: []error{pgx.ErrNoRows}}
	_, err := ClaimNextJob(context.Background(), exec, "worker-a", time.Now(), time.Now())
	if !errors.Is(err, ErrNoJobAvailable) {
		t.Fatalf("expected ErrNoJobAvailable, got %v", err)
	}
}

// TestRecoverExpiredJobLeasesMovesRunningJobsToRecovering simulates the worker
// crash after claim required by P0-0B-001. The expired running lease is moved to
// recovering with the previous owner retained in the returned evidence row.
func TestRecoverExpiredJobLeasesMovesRunningJobsToRecovering(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 1, 0, 0, time.UTC)
	expired := now.Add(-time.Second)
	heartbeat := expired.Add(-10 * time.Second)
	rows := &fakeJobRows{values: [][]any{{
		int64(42),
		int64(1001),
		int64(7),
		"worker-a",
		expired,
		1,
		"zoom",
		"zoom.create_user",
		"create-user",
		heartbeat,
	}}}
	exec := &fakeJobExecutor{queryRows: []pgx.Rows{rows}}

	recovered, err := RecoverExpiredJobLeases(context.Background(), exec, now, 10)
	if err != nil {
		t.Fatalf("RecoverExpiredJobLeases returned error: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("expected one recovered job, got %d", len(recovered))
	}
	if recovered[0].PreviousOwner != "worker-a" || recovered[0].ReconcileState != JobRecoveryStateRecoveredForRetry {
		t.Fatalf("unexpected recovered job: %#v", recovered[0])
	}
	if recovered[0].ExpiredAt == nil || !recovered[0].ExpiredAt.Equal(expired) {
		t.Fatalf("expected expired lease evidence to preserve %s, got %#v", expired, recovered[0].ExpiredAt)
	}
	if recovered[0].LastHeartbeat == nil || !recovered[0].LastHeartbeat.Equal(heartbeat) {
		t.Fatalf("expected heartbeat evidence to preserve %s, got %#v", heartbeat, recovered[0].LastHeartbeat)
	}
	query := exec.queries[0]
	if !strings.Contains(query.sql, "or job_state = $4") || query.args[3] != string(core.JobStateRecovering) {
		t.Fatalf("recovery query did not include interrupted recovering rows: %#v", query)
	}
	if !rows.closed {
		t.Fatal("expected recovered rows to be closed")
	}
}

// TestRecoverExpiredJobLeasesPreservesMissingHeartbeatEvidence covers old or
// interrupted recovery rows where heartbeat evidence is genuinely absent. The
// recovery loop must surface that absence instead of replacing it with now.
func TestRecoverExpiredJobLeasesPreservesMissingHeartbeatEvidence(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 1, 30, 0, time.UTC)
	rows := &fakeJobRows{values: [][]any{{
		int64(43),
		int64(1002),
		int64(8),
		"",
		nil,
		2,
		"internal",
		"internal.noop",
		"noop",
		nil,
	}}}
	exec := &fakeJobExecutor{queryRows: []pgx.Rows{rows}}

	recovered, err := RecoverExpiredJobLeases(context.Background(), exec, now, 10)
	if err != nil {
		t.Fatalf("RecoverExpiredJobLeases returned error: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("expected one recovered job, got %d", len(recovered))
	}
	if recovered[0].ExpiredAt != nil {
		t.Fatalf("expected nil expired-at evidence, got %#v", recovered[0].ExpiredAt)
	}
	if recovered[0].LastHeartbeat != nil {
		t.Fatalf("expected nil heartbeat evidence, got %#v", recovered[0].LastHeartbeat)
	}
	if recovered[0].NeedsProvider {
		t.Fatalf("expected internal recovery row not to need provider reconciliation: %#v", recovered[0])
	}
}

// TestReconcileRecoveredJobMarksAlreadySucceededWithoutRetry proves the
// duplicate-execution guard: when an external_request_log success exists for
// the crashed job, recovery marks the job succeeded instead of making it
// claimable again.
func TestReconcileRecoveredJobMarksAlreadySucceededWithoutRetry(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 2, 0, 0, time.UTC)
	exec := &fakeJobExecutor{rowScans: [][]any{
		{true},
		{int64(42)},
	}}

	state, err := ReconcileRecoveredJob(context.Background(), exec, 42, now)
	if err != nil {
		t.Fatalf("ReconcileRecoveredJob returned error: %v", err)
	}
	if state != JobRecoveryStateSucceeded {
		t.Fatalf("expected succeeded reconciliation, got %s", state)
	}
	update := exec.queries[1]
	if update.args[0] != string(core.JobStateSucceeded) || update.args[1] != true {
		t.Fatalf("expected succeeded update without attempt bump: %#v", update)
	}
}

// TestReconcileRecoveredJobQueuesRetryWhenNoExternalSuccess proves the safe
// retry path for a crash that happened before any provider success was logged.
func TestReconcileRecoveredJobQueuesRetryWhenNoExternalSuccess(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 3, 0, 0, time.UTC)
	exec := &fakeJobExecutor{rowScans: [][]any{
		{false},
		{int64(42)},
	}}

	state, err := ReconcileRecoveredJob(context.Background(), exec, 42, now)
	if err != nil {
		t.Fatalf("ReconcileRecoveredJob returned error: %v", err)
	}
	if state != JobRecoveryStateRecoveredForRetry {
		t.Fatalf("expected recovered_for_retry, got %s", state)
	}
	update := exec.queries[1]
	if update.args[0] != string(core.JobStateQueued) || update.args[1] != false {
		t.Fatalf("expected queued retry update with attempt bump: %#v", update)
	}
}
