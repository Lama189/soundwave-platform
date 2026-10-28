package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

func TestNewWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		provider  domain.PaymentProvider
		eventID   string
		eventType string
		payload   []byte
		wantErr   error
	}{
		{
			name:      "success",
			provider:  domain.PaymentProviderStripe,
			eventID:   "evt_123",
			eventType: "payment_intent.succeeded",
			payload:   []byte(`{"id":"evt_123"}`),
			wantErr:   nil,
		},
		{
			name:      "empty event id",
			provider:  domain.PaymentProviderStripe,
			eventID:   "",
			eventType: "payment_intent.succeeded",
			payload:   []byte(`{"id":"evt_123"}`),
			wantErr:   domain.ErrInvalidInput,
		},
		{
			name:      "empty event type",
			provider:  domain.PaymentProviderStripe,
			eventID:   "evt_123",
			eventType: "",
			payload:   []byte(`{"id":"evt_123"}`),
			wantErr:   domain.ErrInvalidInput,
		},
		{
			name:      "empty payload",
			provider:  domain.PaymentProviderStripe,
			eventID:   "evt_123",
			eventType: "payment_intent.succeeded",
			payload:   nil,
			wantErr:   domain.ErrWebhookPayloadEmpty,
		},
		{
			name:      "unsupported provider",
			provider:  domain.PaymentProvider("unknown"),
			eventID:   "evt_123",
			eventType: "payment_intent.succeeded",
			payload:   []byte(`{}`),
			wantErr:   domain.ErrUnsupportedProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := domain.NewWebhookEvent(tt.provider, tt.eventID, tt.eventType, tt.payload)
			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ev.Status != domain.WebhookStatusReceived {
				t.Errorf("expected status %s, got %s", domain.WebhookStatusReceived, ev.Status)
			}
		})
	}
}

func TestWebhookEvent_Transitions(t *testing.T) {
	ev, err := domain.NewWebhookEvent(domain.PaymentProviderStripe, "evt_1", "payment_intent.succeeded", []byte(`{}`))
	if err != nil {
		t.Fatalf("failed to create webhook event: %v", err)
	}

	if ev.IsTerminal() {
		t.Errorf("initial state should not be terminal")
	}

	if err := ev.MarkProcessed(); err != nil {
		t.Fatalf("unexpected error marking processed: %v", err)
	}
	if ev.Status != domain.WebhookStatusProcessed {
		t.Errorf("expected status %s, got %s", domain.WebhookStatusProcessed, ev.Status)
	}
	if !ev.IsTerminal() {
		t.Errorf("expected terminal state")
	}
	if ev.ProcessedAt == nil {
		t.Errorf("processed_at should be set")
	}

	if err := ev.MarkProcessed(); !errors.Is(err, domain.ErrWebhookAlreadyProcessed) {
		t.Errorf("expected ErrWebhookAlreadyProcessed, got %v", err)
	}
	if err := ev.MarkFailed(errors.New("fail")); !errors.Is(err, domain.ErrWebhookAlreadyProcessed) {
		t.Errorf("expected ErrWebhookAlreadyProcessed, got %v", err)
	}
	if err := ev.MarkIgnored("ignore"); !errors.Is(err, domain.ErrWebhookAlreadyProcessed) {
		t.Errorf("expected ErrWebhookAlreadyProcessed, got %v", err)
	}
}

func TestRestoreWebhookEvent(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()

	ev, err := domain.RestoreWebhookEvent(
		id, domain.PaymentProviderStripe, "evt_restore", "payment_intent.succeeded",
		[]byte(`{}`), domain.WebhookStatusProcessed, nil, now, &now,
	)
	if err != nil {
		t.Fatalf("failed to restore webhook event: %v", err)
	}
	if ev.ID != id || ev.Status != domain.WebhookStatusProcessed {
		t.Errorf("restored webhook event fields mismatch")
	}
}
