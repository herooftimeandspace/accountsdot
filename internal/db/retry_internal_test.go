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

func (t *testTx) Begin(context.Context) (pgx.Tx, error) { return t, nil }
func (t *testTx) Commit(context.Context) error          { return t.commitErr }
func (t *testTx) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}
func (t *testTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *testTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *testTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *testTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *testTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *testTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *testTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *testTx) Conn() *pgx.Conn                                         { return nil }

type deadlockErr struct{}

func (deadlockErr) Error() string    { return "deadlock detected" }
func (deadlockErr) SQLState() string { return "40P01" }

type serializableTxErr struct{}

func (serializableTxErr) Error() string    { return "serialization failure" }
func (serializableTxErr) SQLState() string { return "40001" }

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

func TestWithRetryReturnsNonRetryableCommitError(t *testing.T) {
	beginner := &testBeginner{
		tx: &testTx{commitErr: errors.New("commit failed")},
	}

	err := WithRetry(context.Background(), beginner, func(pgx.Tx) error { return nil })
	if err == nil || err.Error() != "commit failed" {
		t.Fatalf("expected direct commit error, got %v", err)
	}
}

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

func TestDefaultSleepHookReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	defaultSleepHook(ctx, 0)
}

func TestSetSleepHookNilUsesDefaultSleepHook(t *testing.T) {
	restore := SetSleepHook(func(context.Context, int) {})
	defer restore()

	restoreDefault := SetSleepHook(nil)
	defer restoreDefault()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sleepHook(ctx, 0)
}
