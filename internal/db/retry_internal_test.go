package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type testBeginner struct {
	beginErrs []error
	tx        *testTx
	begins    int
}

// BeginTx documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (b *testBeginner) BeginTx(_ context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	b.begins++
	if len(b.beginErrs) > 0 {
		err := b.beginErrs[0]
		b.beginErrs = b.beginErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	tx := *b.tx
	return &tx, nil
}

type testTx struct {
	commitErr error
	rollbacks int
}

// Begin documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }

// Commit documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (t *testTx) Commit(context.Context) error { return t.commitErr }

// Rollback documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (t *testTx) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

// CopyFrom documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

// SendBatch documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }

// LargeObjects documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

// Prepare documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

// Exec documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func (t *testTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// Query documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

// QueryRow documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

// Conn documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (t *testTx) Conn() *pgx.Conn { return nil }

type deadlockErr struct{}

// Error documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (deadlockErr) Error() string { return "deadlock detected" }

// SQLState documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (deadlockErr) SQLState() string { return "40P01" }

type serializableTxErr struct{}

// Error documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (serializableTxErr) Error() string { return "serialization failure" }

// SQLState documents the data flow for internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func (serializableTxErr) SQLState() string { return "40001" }

// TestIsRetryableTxError exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestIsRetryableTxError(t *testing.T) {
	if !IsRetryableTxError(serializableTxErr{}) {
		t.Fatal("expected serializable error to be retryable")
	}
	if !IsRetryableTxError(deadlockErr{}) {
		t.Fatal("expected deadlock error to be retryable")
	}
	if IsRetryableTxError(errors.New("boom")) {
		t.Fatal("expected plain error to be non-retryable")
	}
}

// TestWithRetryRetriesRetryableBeginErrors exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryRetriesRetryableBeginErrors(t *testing.T) {
	beginner := &testBeginner{
		beginErrs: []error{serializableTxErr{}, serializableTxErr{}, nil},
		tx:        &testTx{},
	}

	var sleeps []int
	restore := SetSleepHook(func(_ context.Context, attempt int) {
		sleeps = append(sleeps, attempt)
	})
	defer restore()

	if err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil }); err != nil {
		t.Fatalf("WithRetry returned error: %v", err)
	}
	if beginner.begins != 3 {
		t.Fatalf("expected 3 begins, got %d", beginner.begins)
	}
	if len(sleeps) != 2 {
		t.Fatalf("expected 2 sleeps, got %d", len(sleeps))
	}
}

// TestWithRetryRetriesRetryableFunctionErrors exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryRetriesRetryableFunctionErrors(t *testing.T) {
	beginner := &testBeginner{tx: &testTx{}}
	calls := 0

	var sleeps []int
	restore := SetSleepHook(func(_ context.Context, attempt int) {
		sleeps = append(sleeps, attempt)
	})
	defer restore()

	err := WithRetry(context.Background(), beginner, func(tx pgx.Tx) error {
		calls++
		if calls == 1 {
			return serializableTxErr{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithRetry returned error: %v", err)
	}
	if beginner.begins != 2 {
		t.Fatalf("expected 2 transaction attempts, got %d", beginner.begins)
	}
	if len(sleeps) != 1 {
		t.Fatalf("expected 1 sleep, got %d", len(sleeps))
	}
}

// TestWithRetryReturnsWrappedRetryableFunctionErrorAfterMaxAttempts exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryReturnsWrappedRetryableFunctionErrorAfterMaxAttempts(t *testing.T) {
	beginner := &testBeginner{tx: &testTx{}}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error {
		return serializableTxErr{}
	})
	if err == nil {
		t.Fatal("expected a wrapped max retry error")
	}
	if !strings.Contains(err.Error(), "transaction failed after 5 retries") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// TestWithRetryReturnsWrappedRetryableCommitErrorAfterMaxAttempts exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryReturnsWrappedRetryableCommitErrorAfterMaxAttempts(t *testing.T) {
	beginner := &testBeginner{
		tx: &testTx{commitErr: serializableTxErr{}},
	}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil })
	if err == nil {
		t.Fatal("expected a wrapped max retry error")
	}
	if !strings.Contains(err.Error(), "transaction failed after 5 retries") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// TestWithRetryReturnsNonRetryableBeginError exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryReturnsNonRetryableBeginError(t *testing.T) {
	beginner := &testBeginner{
		beginErrs: []error{errors.New("nope")},
		tx:        &testTx{},
	}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil })
	if err == nil || err.Error() != "nope" {
		t.Fatalf("expected direct begin error, got %v", err)
	}
}

// TestWithRetryReturnsNonRetryableCommitError exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryReturnsNonRetryableCommitError(t *testing.T) {
	beginner := &testBeginner{
		tx: &testTx{commitErr: errors.New("commit failed")},
	}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil })
	if err == nil || err.Error() != "commit failed" {
		t.Fatalf("expected direct commit error, got %v", err)
	}
}

// TestWithRetryReturnsMaxRetryError exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestWithRetryReturnsMaxRetryError(t *testing.T) {
	beginner := &testBeginner{
		beginErrs: []error{
			serializableTxErr{},
			serializableTxErr{},
			serializableTxErr{},
			serializableTxErr{},
			serializableTxErr{},
		},
		tx: &testTx{},
	}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil })
	if err == nil {
		t.Fatal("expected an error after max retries")
	}
	if !strings.Contains(err.Error(), "transaction failed after 5 retries") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

// TestDefaultSleepHookReturnsOnCanceledContext exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/external-write-inventory.md.
func TestDefaultSleepHookReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	defaultSleepHook(ctx, 0)
}

// TestSetSleepHookNilUsesDefaultSleepHook exercises and documents internal/db/retry_internal_test.go. Repo tests call this function to lock down the behavior described here; use failing assertions and breakpoints in this test path to debug regressions. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func TestSetSleepHookNilUsesDefaultSleepHook(t *testing.T) {
	restore := SetSleepHook(func(context.Context, int) {})
	defer restore()

	restoreDefault := SetSleepHook(nil)
	defer restoreDefault()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sleepHook(ctx, 0)
}
