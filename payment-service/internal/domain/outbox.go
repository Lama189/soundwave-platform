package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AggregateType string
type EventType string

const (
	AggregateTypePayment AggregateType = "payment"
)

const (
	EventTypePaymentCreated   EventType = "payment.created"
	EventTypePaymentSucceeded EventType = "payment.succeeded"
	EventTypePaymentFailed    EventType = "payment.failed"
	EventTypePaymentCancelled EventType = "payment.cancelled"
	EventTypePaymentRefunded  EventType = "payment.refunded"
)

func (t AggregateType) IsValid() bool {
	switch t {
	case AggregateTypePayment:
		return true
	default:
		return false
	}
}

func (t EventType) IsValid() bool {
	switch t {
	case EventTypePaymentCreated,
		EventTypePaymentSucceeded,
		EventTypePaymentFailed,
		EventTypePaymentCancelled,
		EventTypePaymentRefunded:
		return true
	default:
		return false
	}
}

type Event struct {
	ID            uuid.UUID
	AggregateType AggregateType
	AggregateID   uuid.UUID
	EventType     EventType
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

func NewEvent(aggregateType AggregateType, aggregateID uuid.UUID, eventType EventType, payload any) (*Event, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox event payload: %w", err)
	}

	e := &Event{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       bytes,
		CreatedAt:     time.Now().UTC(),
	}

	if err := e.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create outbox event: %w", err)
	}

	return e, nil
}

func RestoreEvent(
	id uuid.UUID,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
	eventType EventType,
	payload []byte,
	createdAt time.Time,
	processedAt *time.Time,
) *Event {
	return &Event{
		ID:            id,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     createdAt,
		ProcessedAt:   processedAt,
	}
}

func (e *Event) Validate() error {
	if e.ID == uuid.Nil {
		return fmt.Errorf("%w: event id is required", ErrInvalidInput)
	}
	if e.AggregateID == uuid.Nil {
		return fmt.Errorf("%w: aggregate id is required", ErrInvalidInput)
	}
	if !e.AggregateType.IsValid() {
		return fmt.Errorf("%w: invalid aggregate type %q", ErrInvalidInput, e.AggregateType)
	}
	if !e.EventType.IsValid() {
		return fmt.Errorf("%w: invalid event type %q", ErrInvalidInput, e.EventType)
	}
	if len(e.Payload) == 0 {
		return fmt.Errorf("%w: event payload cannot be empty", ErrInvalidInput)
	}
	return nil
}

// Event payloads

type PaymentSucceededPayload struct {
	PaymentID      uuid.UUID       `json:"payment_id"`
	OrderID        uuid.UUID       `json:"order_id"`
	PayerUserID    uuid.UUID       `json:"payer_user_id"`
	Amount         int64           `json:"amount"`
	Currency       Currency        `json:"currency"`
	Provider       PaymentProvider `json:"provider"`
	ExternalID     string          `json:"external_id"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type PaymentFailedPayload struct {
	PaymentID     uuid.UUID `json:"payment_id"`
	OrderID       uuid.UUID `json:"order_id"`
	FailureReason string    `json:"failure_reason"`
}

type PaymentRefundedPayload struct {
	PaymentID  uuid.UUID `json:"payment_id"`
	OrderID    uuid.UUID `json:"order_id"`
	ExternalID string    `json:"external_id"`
}
