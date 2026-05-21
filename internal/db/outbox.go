package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// OutboxExecutor is the narrow database surface used by event-outbox readers.
// Worker and publisher callers should pass a pgx.Tx from WithRetry when the
// read is paired with state changes so strict event sequencing stays inside the
// repo's SERIALIZABLE transaction boundary.
type OutboxExecutor interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// OutboxEvent is the durable event row returned to future outbox publishers.
// GlobalTick is the only ordering signal callers should use when comparing
// events with jobs or other outbox rows, because UUIDs and table-local ids do
// not represent cross-table workflow order.
type OutboxEvent struct {
	ID         int64
	GlobalTick int64
	Topic      string
	EventType  string
	Payload    string
	CreatedAt  time.Time
}

// ListOutboxEvents returns the oldest event_outbox rows by global_tick for
// Phase 0 ordering evidence and future publisher loops. The function only
// reads local database rows; it does not publish, acknowledge, or mutate
// provider-facing state.
func ListOutboxEvents(ctx context.Context, tx OutboxExecutor, limit int) ([]OutboxEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `
select id, global_tick, topic, event_type, payload::text, created_at
from event_outbox
order by global_tick asc
limit $1
`, limit)
	if err != nil {
		return nil, fmt.Errorf("list outbox events: %w", err)
	}
	defer rows.Close()

	events := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.GlobalTick,
			&event.Topic,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	return events, nil
}
