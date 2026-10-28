package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WebhookEventRepository struct {
	db DB
}

func NewWebhookEventRepository(db DB) *WebhookEventRepository {
	return &WebhookEventRepository{db: db}
}

func (r *WebhookEventRepository) Create(ctx context.Context, event *domain.WebhookEvent) error {
	const query = `
		INSERT INTO webhook_events (
			id,
			provider,
			external_event_id,
			event_type,
			payload,
			status,
			error_message,
			created_at,
			processed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		event.ID,
		string(event.Provider),
		event.ExternalEventID,
		event.EventType,
		event.Payload,
		string(event.Status),
		event.ErrorMessage,
		event.CreatedAt,
		event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("create webhook event %s: %w", event.ID, mapError(err))
	}

	return nil
}

const baseWebhookEventSelectQuery = `
	SELECT
		id,
		provider,
		external_event_id,
		event_type,
		payload,
		status,
		error_message,
		created_at,
		processed_at
	FROM webhook_events
`

func (r *WebhookEventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEvent, error) {
	const query = baseWebhookEventSelectQuery + "\nWHERE id = $1;"

	event, err := r.scanWebhookEvent(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get webhook event by id %s: %w", id, err)
	}

	return event, nil
}

func (r *WebhookEventRepository) GetByProviderAndExternalID(ctx context.Context, provider domain.PaymentProvider, externalEventID string) (*domain.WebhookEvent, error) {
	const query = baseWebhookEventSelectQuery + "\nWHERE provider = $1 AND external_event_id = $2;"

	event, err := r.scanWebhookEvent(r.db.QueryRow(ctx, query, string(provider), externalEventID))
	if err != nil {
		return nil, fmt.Errorf("get webhook event by provider %s and external_id %s: %w", provider, externalEventID, err)
	}

	return event, nil
}

func (r *WebhookEventRepository) GetByProviderAndExternalIDForUpdate(ctx context.Context, provider domain.PaymentProvider, externalEventID string) (*domain.WebhookEvent, error) {
	const query = baseWebhookEventSelectQuery + "\nWHERE provider = $1 AND external_event_id = $2\nFOR UPDATE;"

	event, err := r.scanWebhookEvent(r.db.QueryRow(ctx, query, string(provider), externalEventID))
	if err != nil {
		return nil, fmt.Errorf("get webhook event by provider %s and external_id %s for update: %w", provider, externalEventID, err)
	}

	return event, nil
}

func (r *WebhookEventRepository) Update(ctx context.Context, event *domain.WebhookEvent) error {
	const query = `
		UPDATE webhook_events
		SET
			status = $2,
			error_message = $3,
			processed_at = $4
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		event.ID,
		string(event.Status),
		event.ErrorMessage,
		event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("update webhook event %s: %w", event.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("update webhook event %s: %w", event.ID, domain.ErrNotFound)
	}

	return nil
}

func (r *WebhookEventRepository) scanWebhookEvent(row pgx.Row) (*domain.WebhookEvent, error) {
	var m webhookEventModel
	err := row.Scan(
		&m.ID,
		&m.Provider,
		&m.ExternalEventID,
		&m.EventType,
		&m.Payload,
		&m.Status,
		&m.ErrorMessage,
		&m.CreatedAt,
		&m.ProcessedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	event, err := m.toDomain()
	if err != nil {
		return nil, fmt.Errorf("restore webhook event domain: %w", err)
	}

	return event, nil
}
