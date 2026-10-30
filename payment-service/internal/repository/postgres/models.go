package postgres

import (
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

type paymentModel struct {
	ID             uuid.UUID `db:"id"`
	OrderID        uuid.UUID `db:"order_id"`
	PayerUserID    uuid.UUID `db:"payer_user_id"`
	Amount         int64     `db:"amount"`
	Currency       string    `db:"currency"`
	Status         string    `db:"status"`
	Provider       string    `db:"provider"`
	ExternalID     string    `db:"external_id"`
	ClientSecret   string    `db:"client_secret"`
	FailureReason  *string   `db:"failure_reason"`
	IdempotencyKey string    `db:"idempotency_key"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (m *paymentModel) toDomain() (*domain.Payment, error) {
	return domain.RestorePayment(
		m.ID,
		m.OrderID,
		m.PayerUserID,
		m.Amount,
		domain.Currency(m.Currency),
		domain.PaymentStatus(m.Status),
		domain.PaymentProvider(m.Provider),
		m.ExternalID,
		m.ClientSecret,
		m.IdempotencyKey,
		m.FailureReason,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

func toPaymentModel(p *domain.Payment) paymentModel {
	return paymentModel{
		ID:             p.ID,
		OrderID:        p.OrderID,
		PayerUserID:    p.PayerUserID,
		Amount:         p.Amount,
		Currency:       string(p.Currency),
		Status:         string(p.Status),
		Provider:       string(p.Provider),
		ExternalID:     p.ExternalID,
		ClientSecret:   p.ClientSecret,
		FailureReason:  p.FailureReason,
		IdempotencyKey: p.IdempotencyKey,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

type webhookEventModel struct {
	ID              uuid.UUID  `db:"id"`
	Provider        string     `db:"provider"`
	ExternalEventID string     `db:"external_event_id"`
	EventType       string     `db:"event_type"`
	Payload         []byte     `db:"payload"`
	Status          string     `db:"status"`
	ErrorMessage    *string    `db:"error_message"`
	CreatedAt       time.Time  `db:"created_at"`
	ProcessedAt     *time.Time `db:"processed_at"`
}

func (m *webhookEventModel) toDomain() (*domain.WebhookEvent, error) {
	return domain.RestoreWebhookEvent(
		m.ID,
		domain.PaymentProvider(m.Provider),
		m.ExternalEventID,
		m.EventType,
		m.Payload,
		domain.WebhookStatus(m.Status),
		m.ErrorMessage,
		m.CreatedAt,
		m.ProcessedAt,
	)
}

func toWebhookEventModel(e *domain.WebhookEvent) webhookEventModel {
	return webhookEventModel{
		ID:              e.ID,
		Provider:        string(e.Provider),
		ExternalEventID: e.ExternalEventID,
		EventType:       e.EventType,
		Payload:         e.Payload,
		Status:          string(e.Status),
		ErrorMessage:    e.ErrorMessage,
		CreatedAt:       e.CreatedAt,
		ProcessedAt:     e.ProcessedAt,
	}
}

type eventModel struct {
	ID            uuid.UUID  `db:"id"`
	AggregateType string     `db:"aggregate_type"`
	AggregateID   uuid.UUID  `db:"aggregate_id"`
	EventType     string     `db:"event_type"`
	Payload       []byte     `db:"payload"`
	CreatedAt     time.Time  `db:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at"`
	LockedUntil   *time.Time `db:"locked_until"`
	LockedBy      *string    `db:"locked_by"`
	RetryCount    int        `db:"retry_count"`
	LastError     *string    `db:"last_error"`
}

func (m *eventModel) toDomain() *domain.Event {
	return domain.RestoreEvent(
		m.ID,
		domain.AggregateType(m.AggregateType),
		m.AggregateID,
		domain.EventType(m.EventType),
		m.Payload,
		m.CreatedAt,
		m.ProcessedAt,
	)
}
