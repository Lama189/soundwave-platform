package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProcessedEvent(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name        string
		eventID     uuid.UUID
		eventType   InboxEventType
		expectError bool
	}{
		{
			name:        "valid order.paid event",
			eventID:     validID,
			eventType:   InboxEventTypeOrderPaid,
			expectError: false,
		},
		{
			name:        "nil event id",
			eventID:     uuid.Nil,
			eventType:   InboxEventTypeOrderPaid,
			expectError: true,
		},
		{
			name:        "invalid event type",
			eventID:     validID,
			eventType:   "unknown.event",
			expectError: true,
		},
		{
			name:        "empty event type",
			eventID:     validID,
			eventType:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe, err := NewProcessedEvent(tt.eventID, tt.eventType)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pe.EventID != tt.eventID {
				t.Errorf("expected EventID %s, got %s", tt.eventID, pe.EventID)
			}
			if pe.EventType != tt.eventType {
				t.Errorf("expected EventType %s, got %s", tt.eventType, pe.EventType)
			}
			if pe.ProcessedAt.IsZero() {
				t.Error("expected non-zero ProcessedAt")
			}
		})
	}
}

func TestRestoreProcessedEvent(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()
	pe := RestoreProcessedEvent(id, InboxEventTypeOrderPaid, now)

	if pe.EventID != id {
		t.Errorf("expected EventID %s, got %s", id, pe.EventID)
	}
	if pe.EventType != InboxEventTypeOrderPaid {
		t.Errorf("expected EventType %s, got %s", InboxEventTypeOrderPaid, pe.EventType)
	}
	if !pe.ProcessedAt.Equal(now) {
		t.Errorf("expected ProcessedAt %v, got %v", now, pe.ProcessedAt)
	}
}
