package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
)

type ProcessedEventsRepository struct {
	db DB
}

func NewProcessedEventsRepository(db DB) *ProcessedEventsRepository {
	return &ProcessedEventsRepository{db: db}
}

func (r *ProcessedEventsRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1);`

	var exists bool
	err := r.db.QueryRow(ctx, query, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check processed event %s: %w", eventID, mapError(err))
	}

	return exists, nil
}

func (r *ProcessedEventsRepository) MarkProcessed(ctx context.Context, event *domain.ProcessedEvent) error {
	const query = `
		INSERT INTO processed_events (
			event_id,
			event_type,
			processed_at
		) VALUES (
			$1, $2, $3
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		event.EventID,
		string(event.EventType),
		event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("mark event %s as processed: %w", event.EventID, mapError(err))
	}

	return nil
}

func (r *ProcessedEventsRepository) GetByID(ctx context.Context, eventID uuid.UUID) (*domain.ProcessedEvent, error) {
	const query = `
		SELECT 
			event_id,
			event_type,
			processed_at
		FROM processed_events
		WHERE event_id = $1;
	`

	var m processedEventModel
	err := r.db.QueryRow(ctx, query, eventID).Scan(&m.EventID, &m.EventType, &m.ProcessedAt)
	if err != nil {
		return nil, fmt.Errorf("get processed event %s: %w", eventID, mapError(err))
	}

	return m.toDomain(), nil
}
