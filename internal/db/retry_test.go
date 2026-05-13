package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/herooftimeandspace/go-employee-provisioner/internal/db"
)

type fakePool struct {
	tx                      *fakeTx
	begins                  int
	options                 []pgx.TxOptions
	remainingCommitFailures int
}

// BeginTx documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakePool) BeginTx(_ context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	f.begins++
	f.options = append(f.options, opts)
	tx := *f.tx
	tx.pool = f
	return &tx, nil
}

type fakeTx struct {
	pool           *fakePool
	commitFailures int
	commits        int
	rollbacks      int
}

// Begin documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { return f, nil }

// Commit documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (f *fakeTx) Commit(context.Context) error {
	f.commits++
	if f.pool != nil && f.pool.remainingCommitFailures > 0 {
		f.pool.remainingCommitFailures--
		return serializableErr{}
	}
	if f.commits <= f.commitFailures {
		return serializableErr{}
	}
	return nil
}

// Rollback documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (f *fakeTx) Rollback(context.Context) error { f.rollbacks++; return nil }

// CopyFrom documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

// SendBatch documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }

// LargeObjects documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

// Prepare documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

// Exec documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// Query documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

// QueryRow documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

// Conn documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (f *fakeTx) Conn() *pgx.Conn { return nil }

type serializableErr struct{}

// Error documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (serializableErr) Error() string { return "serialization failure" }

// SQLState documents the data flow for internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (serializableErr) SQLState() string { return "40001" }

// TestWithRetrySerializableRetries exercises and documents internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetrySerializableRetries(t *testing.T) {
	ctx := context.Background()
	pool := &fakePool{
		tx:                      &fakeTx{},
		remainingCommitFailures: 2,
	}

	var sleeps []int
	restore := db.SetSleepHook(func(_ context.Context, attempt int) {
		sleeps = append(sleeps, attempt)
	})
	defer restore()

	err := db.WithRetry(ctx, pool, func(pgx.Tx) error { return nil })
	if err != nil {
		t.Fatalf("WithRetry returned error: %v", err)
	}
	if pool.begins != 3 {
		t.Fatalf("expected 3 transaction attempts, got %d", pool.begins)
	}
	if len(pool.options) == 0 || pool.options[0].IsoLevel != pgx.Serializable {
		t.Fatalf("expected serializable isolation, got %+v", pool.options)
	}
	if len(sleeps) != 2 {
		t.Fatalf("expected 2 backoff sleeps, got %d", len(sleeps))
	}
}

// TestWithRetryStopsOnNonRetryableError exercises and documents internal/db/retry_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryStopsOnNonRetryableError(t *testing.T) {
	ctx := context.Background()
	pool := &fakePool{tx: &fakeTx{}}
	want := errors.New("boom")

	err := db.WithRetry(ctx, pool, func(pgx.Tx) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
	if pool.begins != 1 {
		t.Fatalf("expected 1 transaction attempt, got %d", pool.begins)
	}
}
