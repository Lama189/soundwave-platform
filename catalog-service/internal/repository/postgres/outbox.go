package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
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
		)
		VALUES ($1, $2, $3, $4, $5, $6);
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

func (r *OutboxRepository) ClaimUnprocessed(ctx context.Context, limit int, lockDuration time.Duration, workerID string) ([]*domain.Event, error) {
	const query = `
		WITH available AS (
			SELECT id
			FROM outbox_events
			WHERE processed_at IS NULL
			  AND (locked_until IS NULL OR locked_until < $1)
			ORDER BY created_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events
		SET 
			locked_until = $3,
			locked_by = $4
		FROM available
		WHERE outbox_events.id = available.id
		RETURNING 
			outbox_events.id, 
			outbox_events.aggregate_type, 
			outbox_events.aggregate_id, 
			outbox_events.event_type, 
			outbox_events.payload, 
			outbox_events.created_at, 
			outbox_events.processed_at;
	`

	now := time.Now().UTC()
	lockedUntil := now.Add(lockDuration)

	rows, err := r.db.Query(ctx, query, now, limit, lockedUntil, workerID)
	if err != nil {
		return nil, fmt.Errorf("claim unprocessed outbox events: %w", mapError(err))
	}
	defer rows.Close()

	events := make([]*domain.Event, 0, limit)
	for rows.Next() {
		var m EventModel
		if err := rows.Scan(
			&m.ID,
			&m.AggregateType,
			&m.AggregateID,
			&m.EventType,
			&m.Payload,
			&m.CreatedAt,
			&m.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}

		events = append(events, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox rows: %w", err)
	}

	return events, nil
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
		var m EventModel
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
		SET processed_at = $1,
		    locked_until = NULL
		WHERE id = ANY($2);
	`

	_, err := r.db.Exec(ctx, query, time.Now().UTC(), ids)
	if err != nil {
		return fmt.Errorf("mark outbox events as processed: %w", mapError(err))
	}

	return nil
}

func (r *OutboxRepository) RecordFailure(ctx context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time, maxRetries int) error {
	const query = `
		UPDATE outbox_events
		SET 
			retry_count = retry_count + 1,
			last_error = $2,
			locked_until = CASE 
				WHEN retry_count + 1 >= $4 THEN NULL
				ELSE $3
			END,
			processed_at = CASE 
				WHEN retry_count + 1 >= $4 THEN NOW()
				ELSE NULL
			END
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id, errMsg, nextRetryAt, maxRetries)
	if err != nil {
		return fmt.Errorf("record outbox event failure: %w", mapError(err))
	}

	return nil
}

func (r *OutboxRepository) CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	const query = `
		DELETE FROM outbox_events
		WHERE processed_at IS NOT NULL
		  AND processed_at < $1;
	`

	threshold := time.Now().UTC().Add(-olderThan)
	tag, err := r.db.Exec(ctx, query, threshold)
	if err != nil {
		return 0, fmt.Errorf("cleanup processed outbox events: %w", mapError(err))
	}

	return tag.RowsAffected(), nil
}

func (r *OutboxRepository) FetchAndClaim(ctx context.Context, limit int) ([]*domain.Event, error) {
	if limit <= 0 {
		return nil, nil
	}

	const query = `
		UPDATE outbox_events
		SET processed_at = NOW()
		WHERE id IN (
			SELECT id
			FROM outbox_events
			WHERE processed_at IS NULL
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at;
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch and claim outbox events: %w", mapError(err))
	}
	defer rows.Close()

	events := make([]*domain.Event, 0, limit)
	for rows.Next() {
		var m EventModel
		if err := rows.Scan(
			&m.ID,
			&m.AggregateType,
			&m.AggregateID,
			&m.EventType,
			&m.Payload,
			&m.CreatedAt,
			&m.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}

		events = append(events, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}

	return events, nil
}
