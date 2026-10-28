package domain_test

import (
	"testing"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

func TestNewEvent(t *testing.T) {
	paymentID := uuid.New()
	orderID := uuid.New()
	userID := uuid.New()

	payload := domain.PaymentSucceededPayload{
		PaymentID:      paymentID,
		OrderID:        orderID,
		PayerUserID:    userID,
		Amount:         2500,
		Currency:       domain.CurrencyUSD,
		Provider:       domain.PaymentProviderMock,
		ExternalID:     "mock_pi_1",
		IdempotencyKey: "idem_1",
	}

	event, err := domain.NewEvent(
		domain.AggregateTypePayment,
		paymentID,
		domain.EventTypePaymentSucceeded,
		payload,
	)
	if err != nil {
		t.Fatalf("unexpected error creating event: %v", err)
	}

	if event.AggregateType != domain.AggregateTypePayment {
		t.Errorf("expected aggregate type %s, got %s", domain.AggregateTypePayment, event.AggregateType)
	}
	if event.EventType != domain.EventTypePaymentSucceeded {
		t.Errorf("expected event type %s, got %s", domain.EventTypePaymentSucceeded, event.EventType)
	}
	if len(event.Payload) == 0 {
		t.Errorf("payload should not be empty")
	}
}
