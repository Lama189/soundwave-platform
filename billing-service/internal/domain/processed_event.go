package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type InboxEventType string

const (
	InboxEventTypeOrderPaid InboxEventType = "order.paid"
)

func (t InboxEventType) IsValid() bool {
	switch t {
	case InboxEventTypeOrderPaid:
		return true
	default:
		return false
	}
}

type ProcessedEvent struct {
	EventID     uuid.UUID
	EventType   InboxEventType
	ProcessedAt time.Time
}

func NewProcessedEvent(eventID uuid.UUID, eventType InboxEventType) (*ProcessedEvent, error) {
	pe := &ProcessedEvent{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now().UTC(),
	}

	if err := pe.Validate(); err != nil {
		return nil, err
	}

	return pe, nil
}

func RestoreProcessedEvent(eventID uuid.UUID, eventType InboxEventType, processedAt time.Time) *ProcessedEvent {
	return &ProcessedEvent{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: processedAt,
	}
}

func (pe *ProcessedEvent) Validate() error {
	if pe.EventID == uuid.Nil {
		return fmt.Errorf("%w: event id is required", ErrInvalidInput)
	}
	if !pe.EventType.IsValid() {
		return fmt.Errorf("%w: invalid inbox event type %q", ErrInvalidInput, pe.EventType)
	}
	return nil
}
