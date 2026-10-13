package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AggregateType string
type EventType string

const (
	AggregateTypeUser  AggregateType = "user"
	AggregateTypeTrack AggregateType = "track"
)

const (
	EventTypeTrackUploaded EventType = "track.uploaded"
	EventTypeTrackReady    EventType = "track.ready"
)

func (t AggregateType) IsValid() bool {
	switch t {
	case AggregateTypeUser, AggregateTypeTrack:
		return true
	default:
		return false
	}
}

func (t EventType) IsValid() bool {
	switch t {
	case EventTypeTrackUploaded, EventTypeTrackReady:
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

func NewEvent(id uuid.UUID, aggregateType AggregateType, aggregateID uuid.UUID, eventType EventType, payload []byte) (*Event, error) {
	e := &Event{
		ID:            id,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
		ProcessedAt:   nil,
	}

	if err := e.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	return e, nil
}

func RestoreEvent(id uuid.UUID, aggregateType AggregateType, aggregateID uuid.UUID, eventType EventType, payload []byte, createdAt time.Time, processedAt *time.Time) *Event {
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
		return errors.New("event ID cannot be nil")
	}

	if !e.AggregateType.IsValid() {
		return errors.New("invalid aggregate type")
	}

	if e.AggregateID == uuid.Nil {
		return errors.New("aggregate ID cannot be nil")
	}

	if !e.EventType.IsValid() {
		return errors.New("invalid event type")
	}

	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return errors.New("payload must be valid JSON")
	}

	return nil
}
