package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxSerializableAttempts = 5

type TxBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type sqlStateError interface {
	SQLState() string
}

var sleepHook = defaultSleepHook

// WithRetry documents the data flow for internal/db/retry.go. Database tests and future transaction code reach this function; debug it by checking transaction boundaries, retryable SQLSTATEs, and rollback behavior. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers. Pay special attention to side effects: this path may mutate response state, DEV mock state, cookies, database transactions, or planned provider work and must stay aligned with docs/planning/external-write-inventory.md.
func WithRetry(ctx context.Context, db TxBeginner, fn func(pgx.Tx) error) error {
	lastErr := errors.New("serializable transaction failed")

	for attempt := 0; attempt < maxSerializableAttempts; attempt++ {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			if IsRetryableTxError(err) && attempt < maxSerializableAttempts-1 {
				lastErr = err
				sleepHook(ctx, attempt)
				continue
			}
			if IsRetryableTxError(err) {
				lastErr = err
				break
			}
			return err
		}

		err = fn(tx)
		if err != nil {
			_ = tx.Rollback(ctx)
			if IsRetryableTxError(err) && attempt < maxSerializableAttempts-1 {
				lastErr = err
				sleepHook(ctx, attempt)
				continue
			}
			if IsRetryableTxError(err) {
				lastErr = err
				break
			}
			return err
		}

		err = tx.Commit(ctx)
		if err == nil {
			return nil
		}
		if IsRetryableTxError(err) && attempt < maxSerializableAttempts-1 {
			lastErr = err
			sleepHook(ctx, attempt)
			continue
		}
		if IsRetryableTxError(err) {
			lastErr = err
			break
		}
		return err
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxSerializableAttempts, lastErr)
}

// IsRetryableTxError resolves decision data for internal/db/retry.go. Database tests and future transaction code reach this function; debug it by checking transaction boundaries, retryable SQLSTATEs, and rollback behavior. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func IsRetryableTxError(err error) bool {
	var stateErr sqlStateError
	if errors.As(err, &stateErr) {
		switch stateErr.SQLState() {
		case "40001", "40P01":
			return true
		}
	}
	return false
}

// SetSleepHook documents the data flow for internal/db/retry.go. Database tests and future transaction code reach this function; debug it by checking transaction boundaries, retryable SQLSTATEs, and rollback behavior. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func SetSleepHook(hook func(context.Context, int)) func() {
	previous := sleepHook
	if hook == nil {
		sleepHook = defaultSleepHook
	} else {
		sleepHook = hook
	}
	return func() {
		sleepHook = previous
	}
}

// defaultSleepHook builds the value used by internal/db/retry.go. Database tests and future transaction code reach this function; debug it by checking transaction boundaries, retryable SQLSTATEs, and rollback behavior. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func defaultSleepHook(ctx context.Context, attempt int) {
	base := 10 * time.Millisecond
	delay := base * time.Duration(1<<attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
