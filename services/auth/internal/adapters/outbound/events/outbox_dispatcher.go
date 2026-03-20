package events

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	oglpguow "github.com/ovya/ogl/pg/uow"
	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
	"github.com/rotisserie/eris"
)

// OutboxDispatcher writes domain events to the auth.event outbox table.
// It uses the active transaction from context (via ogl UoW) so events are
// written atomically with the domain record.
type OutboxDispatcher struct {
	pool *pgxpool.Pool
}

// NewOutboxDispatcher creates a new OutboxDispatcher.
func NewOutboxDispatcher(pool *pgxpool.Pool) *OutboxDispatcher {
	return &OutboxDispatcher{pool: pool}
}

// Dispatch inserts all events into auth.event in a single batch.
func (d *OutboxDispatcher) Dispatch(ctx context.Context, events []user.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	const query = `INSERT INTO auth.event (event_type, payload, occurred_at) VALUES ($1, $2::jsonb, $3)`

	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return eris.Wrapf(err, "marshal event %s", evt.EventType())
		}
		batch.Queue(query, evt.EventType(), string(payload), evt.OccurredAt())
	}

	exec := oglpguow.GetExecutor(ctx, d.pool)
	br := exec.SendBatch(ctx, batch)
	defer br.Close()

	for i := range events {
		if _, err := br.Exec(); err != nil {
			return eris.Wrapf(err, "insert outbox event at index %d", i)
		}
	}

	return nil
}
