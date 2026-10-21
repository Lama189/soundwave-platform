package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository struct {
	db DB
}

func NewOutboxRepository(db DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Record(ctx context.Context, event *domain.Event) error {
	const query = `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		);
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(
		ctx,
		query,
		event.ID,
		string(event.AggregateType),
		event.AggregateID,
		string(event.EventType),
		event.Payload,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record outbox event: %w", mapError(err))
	}

	return nil
}

func (r *OutboxRepository) FetchUnprocessed(ctx context.Context, limit int) ([]*domain.Event, error) {
	const query = `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at
		FROM outbox_events
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1;
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unprocessed outbox events: %w", mapError(err))
	}
	defer rows.Close()

	events := make([]*domain.Event, 0, limit)
	for rows.Next() {
		var m eventModel
		if err := rows.Scan(
			&m.ID,
			&m.AggregateType,
			&m.AggregateID,
			&m.EventType,
			&m.Payload,
			&m.CreatedAt,
			&m.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}

		events = append(events, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}

	return events, nil
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	const query = `
		UPDATE outbox_events
		SET processed_at = $1
		WHERE id = ANY($2);
	`

	_, err := r.db.Exec(ctx, query, time.Now().UTC(), ids)
	if err != nil {
		return fmt.Errorf("mark outbox events processed: %w", mapError(err))
	}

	return nil
}
