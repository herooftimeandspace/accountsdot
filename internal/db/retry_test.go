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

func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { return f, nil }
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
func (f *fakeTx) Rollback(context.Context) error { f.rollbacks++; return nil }
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (f *fakeTx) Conn() *pgx.Conn                                         { return nil }

type serializableErr struct{}

func (serializableErr) Error() string    { return "serialization failure" }
func (serializableErr) SQLState() string { return "40001" }

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
