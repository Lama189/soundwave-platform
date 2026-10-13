package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Record(ctx context.Context, tx pgx.Tx, event *domain.Event) error {
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

	if tx != nil {
		_, err := tx.Exec(
			ctx,
			query,
			event.ID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			event.Payload,
			event.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("record outbox event in tx: %w", err)
		}

		return nil
	}

	_, err := r.pool.Exec(
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
		return fmt.Errorf("record outbox event: %w", err)
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

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unprocessed outbox events: %w", err)
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
	if len(ids) <= 0 {
		return nil
	}

	const query = `
        UPDATE outbox_events
        SET processed_at = $1
        WHERE id = ANY($2);
    `

	_, err := r.pool.Exec(ctx, query, time.Now().UTC(), ids)
	if err != nil {
		return fmt.Errorf("mark outbox events as processed: %w", mapError(err))
	}

	return nil
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

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch and claim outbox events: %w", err)
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
