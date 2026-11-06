package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
)

func TestOrder_MarkProcessing(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()

	// 1. Invalid input: nil paymentID
	err := order.MarkProcessing(uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for nil payment id, got %v", err)
	}

	// 2. Success: pending -> processing
	err = order.MarkProcessing(paymentID)
	if err != nil {
		t.Fatalf("unexpected error on MarkProcessing: %v", err)
	}
	if order.Status != domain.OrderStatusProcessing {
		t.Errorf("expected status %s, got %s", domain.OrderStatusProcessing, order.Status)
	}
	if order.PaymentID == nil || *order.PaymentID != paymentID {
		t.Errorf("expected payment id %s, got %v", paymentID, order.PaymentID)
	}

	// 3. Idempotent: processing -> processing with same paymentID
	err = order.MarkProcessing(paymentID)
	if err != nil {
		t.Fatalf("expected idempotent success on MarkProcessing, got %v", err)
	}
}

func TestOrder_Transitions(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()

	// Pending -> Processing
	if !order.CanTransitionTo(domain.OrderStatusProcessing) {
		t.Errorf("expected CanTransitionTo(processing) to be true from pending")
	}

	_ = order.MarkProcessing(paymentID)

	// Processing -> Paid
	if !order.CanTransitionTo(domain.OrderStatusPaid) {
		t.Errorf("expected CanTransitionTo(paid) to be true from processing")
	}

	// Processing -> Cancelled
	if !order.CanTransitionTo(domain.OrderStatusCancelled) {
		t.Errorf("expected CanTransitionTo(cancelled) to be true from processing")
	}

	// Processing -> Refunded
	if !order.CanTransitionTo(domain.OrderStatusRefunded) {
		t.Errorf("expected CanTransitionTo(refunded) to be true from processing")
	}

	// Mark Paid
	err := order.MarkPaid(paymentID)
	if err != nil {
		t.Fatalf("unexpected error on MarkPaid from processing: %v", err)
	}
	if order.Status != domain.OrderStatusPaid {
		t.Errorf("expected status %s, got %s", domain.OrderStatusPaid, order.Status)
	}

	// Paid -> Refunded
	err = order.MarkRefunded()
	if err != nil {
		t.Fatalf("unexpected error on MarkRefunded from paid: %v", err)
	}
	if order.Status != domain.OrderStatusRefunded {
		t.Errorf("expected status %s, got %s", domain.OrderStatusRefunded, order.Status)
	}

	// Terminal check
	if !order.IsTerminal() {
		t.Errorf("expected refunded to be terminal")
	}
}

func TestOrder_ValidateSagaState(t *testing.T) {
	order := createTestOrder(t)

	// Valid initial state
	if err := order.Validate(); err != nil {
		t.Fatalf("expected valid order, got %v", err)
	}

	// Invalid SagaStatus
	order.Saga.Status = "INVALID_STATUS"
	if err := order.Validate(); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for invalid saga status, got %v", err)
	}

	// Restore valid SagaStatus, make SagaStep invalid
	order.Saga.Status = domain.SagaStatusProcessing
	order.Saga.Step = "INVALID_STEP"
	if err := order.Validate(); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for invalid saga step, got %v", err)
	}
}
