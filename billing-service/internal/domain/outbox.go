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
	AggregateTypePayout AggregateType = "payout"
	AggregateTypeWallet AggregateType = "wallet"
)

const (
	EventTypePayoutRequested EventType = "payout.requested"
	EventTypePayoutCompleted EventType = "payout.completed"
	EventTypePayoutFailed    EventType = "payout.failed"
)

func (t AggregateType) IsValid() bool {
	switch t {
	case AggregateTypePayout, AggregateTypeWallet:
		return true
	default:
		return false
	}
}

func (t EventType) IsValid() bool {
	switch t {
	case EventTypePayoutRequested, EventTypePayoutCompleted, EventTypePayoutFailed:
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
