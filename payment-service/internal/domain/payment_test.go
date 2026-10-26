package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/pkg/money"
	"github.com/google/uuid"
)

func TestNewPayment(t *testing.T) {
	orderID := uuid.New()
	buyerID := uuid.New()

	tests := []struct {
		name           string
		orderID        uuid.UUID
		payerUserID    uuid.UUID
		amount         int64
		currency       domain.Currency
		provider       domain.PaymentProvider
		idempotencyKey string
		wantErr        error
	}{
		{
			name:           "success",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         1000,
			currency:       domain.CurrencyUSD,
			provider:       domain.PaymentProviderStripe,
			idempotencyKey: "idem-key-1",
			wantErr:        nil,
		},
		{
			name:           "zero amount",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         0,
			currency:       domain.CurrencyUSD,
			provider:       domain.PaymentProviderStripe,
			idempotencyKey: "idem-key-1",
			wantErr:        domain.ErrZeroAmount,
		},
		{
			name:           "negative amount",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         -100,
			currency:       domain.CurrencyUSD,
			provider:       domain.PaymentProviderStripe,
			idempotencyKey: "idem-key-1",
			wantErr:        domain.ErrNegativeAmount,
		},
		{
			name:           "unsupported currency",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         1000,
			currency:       money.Currency("BTC"),
			provider:       domain.PaymentProviderStripe,
			idempotencyKey: "idem-key-1",
			wantErr:        domain.ErrUnsupportedCurrency,
		},
		{
			name:           "unsupported provider",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         1000,
			currency:       domain.CurrencyUSD,
			provider:       domain.PaymentProvider("unknown"),
			idempotencyKey: "idem-key-1",
			wantErr:        domain.ErrUnsupportedProvider,
		},
		{
			name:           "empty idempotency key",
			orderID:        orderID,
			payerUserID:    buyerID,
			amount:         1000,
			currency:       domain.CurrencyUSD,
			provider:       domain.PaymentProviderStripe,
			idempotencyKey: "",
			wantErr:        domain.ErrIdempotencyKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := domain.NewPayment(tt.orderID, tt.payerUserID, tt.amount, tt.currency, tt.provider, tt.idempotencyKey)
			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Status != domain.PaymentStatusPending {
				t.Errorf("expected status %s, got %s", domain.PaymentStatusPending, p.Status)
			}
		})
	}
}

func TestPayment_FSM_Transitions(t *testing.T) {
	p, err := domain.NewPayment(uuid.New(), uuid.New(), 2500, domain.CurrencyUSD, domain.PaymentProviderStripe, "idem-1")
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// 1. Pending -> Processing
	if err := p.MarkProcessing("pi_123", "secret_123"); err != nil {
		t.Fatalf("unexpected error on MarkProcessing: %v", err)
	}
	if p.Status != domain.PaymentStatusProcessing {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusProcessing, p.Status)
	}
	if p.ExternalID != "pi_123" || p.ClientSecret != "secret_123" {
		t.Errorf("unexpected external credentials: %s, %s", p.ExternalID, p.ClientSecret)
	}

	// 2. Processing -> Succeeded
	if err := p.MarkSucceeded("pi_123"); err != nil {
		t.Fatalf("unexpected error on MarkSucceeded: %v", err)
	}
	if p.Status != domain.PaymentStatusSucceeded {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusSucceeded, p.Status)
	}
	if !p.IsTerminal() {
		t.Errorf("expected Succeeded to be terminal")
	}

	// 3. Terminal payment cannot transition to Failed or Processing
	if err := p.MarkFailed("some reason", "pi_123"); !errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
		t.Errorf("expected ErrPaymentAlreadyCompleted, got %v", err)
	}
	if err := p.MarkProcessing("pi_123", "secret"); !errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
		t.Errorf("expected ErrPaymentAlreadyCompleted, got %v", err)
	}

	// 4. Succeeded can transition to Refunded
	if err := p.MarkRefunded(); err != nil {
		t.Fatalf("unexpected error on MarkRefunded: %v", err)
	}
	if p.Status != domain.PaymentStatusRefunded {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusRefunded, p.Status)
	}
	if !p.IsTerminal() {
		t.Errorf("expected Refunded to be terminal")
	}

	// 5. Refunded cannot transition anywhere
	if err := p.MarkSucceeded("pi_123"); !errors.Is(err, domain.ErrPaymentAlreadyCompleted) {
		t.Errorf("expected ErrPaymentAlreadyCompleted, got %v", err)
	}
}

func TestPayment_MarkFailed(t *testing.T) {
	p, err := domain.NewPayment(uuid.New(), uuid.New(), 2500, domain.CurrencyUSD, domain.PaymentProviderStripe, "idem-2")
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	if err := p.MarkFailed("card declined", "pi_999"); err != nil {
		t.Fatalf("unexpected error on MarkFailed: %v", err)
	}
	if p.Status != domain.PaymentStatusFailed {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusFailed, p.Status)
	}
	if p.FailureReason == nil || *p.FailureReason != "card declined" {
		t.Errorf("unexpected failure reason: %v", p.FailureReason)
	}
	if !p.IsTerminal() {
		t.Errorf("expected Failed to be terminal")
	}
}

func TestPayment_MarkCancelled(t *testing.T) {
	p, err := domain.NewPayment(uuid.New(), uuid.New(), 2500, domain.CurrencyUSD, domain.PaymentProviderStripe, "idem-3")
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	if err := p.MarkCancelled(); err != nil {
		t.Fatalf("unexpected error on MarkCancelled: %v", err)
	}
	if p.Status != domain.PaymentStatusCancelled {
		t.Errorf("expected status %s, got %s", domain.PaymentStatusCancelled, p.Status)
	}
	if !p.IsTerminal() {
		t.Errorf("expected Cancelled to be terminal")
	}
}

func TestRestorePayment(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()

	p, err := domain.RestorePayment(
		id, orderID, buyerID, 5000, domain.CurrencyUSD,
		domain.PaymentStatusSucceeded, domain.PaymentProviderStripe,
		"pi_restore", "secret", "idem-restore", nil, now, now,
	)
	if err != nil {
		t.Fatalf("unexpected error on restore: %v", err)
	}
	if p.ID != id || p.Status != domain.PaymentStatusSucceeded {
		t.Errorf("restored payment fields mismatch")
	}
}
