package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
)

func createTestOrder(t *testing.T) *domain.Order {
	t.Helper()
	buyerID := uuid.New()
	item, err := domain.NewOrderItem(uuid.Nil, domain.ItemTypeTrack, uuid.New(), uuid.New(), 1000, domain.CurrencyUSD)
	if err != nil {
		t.Fatalf("unexpected error creating order item: %v", err)
	}

	order, err := domain.NewOrder(buyerID, []domain.OrderItem{*item})
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}
	return order
}

func TestNewOrder_InitialSagaState(t *testing.T) {
	order := createTestOrder(t)

	if order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("expected order status %s, got %s", domain.OrderStatusPendingPayment, order.Status)
	}
	if order.Saga.Status != domain.SagaStatusIdle {
		t.Fatalf("expected saga status %s, got %s", domain.SagaStatusIdle, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepNone {
		t.Fatalf("expected saga step %s, got %s", domain.StepNone, order.Saga.Step)
	}
	if order.Saga.StepTimeoutAt != nil {
		t.Fatalf("expected step timeout to be nil")
	}
	if order.IsSagaActive() {
		t.Fatalf("expected saga to not be active initially")
	}
	if order.IsSagaTerminal() {
		t.Fatalf("expected saga to not be terminal initially")
	}
}

func TestSaga_HappyPath(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()
	stepTimeout := 30 * time.Second

	// 1. Start Fulfillment
	err := order.StartFulfillment(paymentID, stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on StartFulfillment: %v", err)
	}
	if order.Status != domain.OrderStatusProcessing {
		t.Errorf("expected order status %s, got %s", domain.OrderStatusProcessing, order.Status)
	}
	if order.Saga.Status != domain.SagaStatusProcessing {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusProcessing, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepGrantEntitlements {
		t.Errorf("expected saga step %s, got %s", domain.StepGrantEntitlements, order.Saga.Step)
	}
	if order.PaymentID == nil || *order.PaymentID != paymentID {
		t.Errorf("expected payment id %s, got %v", paymentID, order.PaymentID)
	}
	if !order.IsSagaActive() {
		t.Errorf("expected saga to be active")
	}

	// 2. Grant Entitlements Succeeded -> Move to Settle Billing
	err = order.OnEntitlementsGranted(stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on OnEntitlementsGranted: %v", err)
	}
	if order.Saga.Status != domain.SagaStatusProcessing {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusProcessing, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepSettleBilling {
		t.Errorf("expected saga step %s, got %s", domain.StepSettleBilling, order.Saga.Step)
	}

	// 3. Billing Settled -> Complete Fulfillment (Pivot Step)
	err = order.CompleteFulfillment()
	if err != nil {
		t.Fatalf("unexpected error on CompleteFulfillment: %v", err)
	}
	if order.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status %s, got %s", domain.OrderStatusPaid, order.Status)
	}
	if order.Saga.Status != domain.SagaStatusCompleted {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusCompleted, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepDone {
		t.Errorf("expected saga step %s, got %s", domain.StepDone, order.Saga.Step)
	}
	if order.Saga.StepTimeoutAt != nil {
		t.Errorf("expected step timeout to be cleared on completion")
	}
	if !order.IsSagaTerminal() {
		t.Errorf("expected saga to be terminal")
	}
}

func TestSaga_CatalogFailure_DirectRefund(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()
	stepTimeout := 30 * time.Second

	_ = order.StartFulfillment(paymentID, stepTimeout)

	// Step 1 fails -> should skip revoke entitlements and go directly to refund
	failureReason := "catalog db down"
	err := order.OnEntitlementsGrantFailed(failureReason, stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on OnEntitlementsGrantFailed: %v", err)
	}
	if order.Saga.Status != domain.SagaStatusCompensating {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusCompensating, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepRefundPayment {
		t.Errorf("expected saga step %s, got %s", domain.StepRefundPayment, order.Saga.Step)
	}
	if order.Saga.FailureReason == nil || *order.Saga.FailureReason != failureReason {
		t.Errorf("expected failure reason %q, got %v", failureReason, order.Saga.FailureReason)
	}

	// Refund completed
	err = order.CompleteCompensation()
	if err != nil {
		t.Fatalf("unexpected error on CompleteCompensation: %v", err)
	}
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("expected order status %s, got %s", domain.OrderStatusCancelled, order.Status)
	}
	if order.Saga.Status != domain.SagaStatusFailed {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusFailed, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepFailed {
		t.Errorf("expected saga step %s, got %s", domain.StepFailed, order.Saga.Step)
	}
	if order.Saga.StepTimeoutAt != nil {
		t.Errorf("expected step timeout to be nil after compensation")
	}
}

func TestSaga_BillingFailure_LIFOCompensation(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()
	stepTimeout := 30 * time.Second

	_ = order.StartFulfillment(paymentID, stepTimeout)
	_ = order.OnEntitlementsGranted(stepTimeout)

	// Step 2 fails -> must revoke entitlements first (LIFO)
	reason := "insufficient platform balance"
	err := order.OnSettlementFailed(reason, stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on OnSettlementFailed: %v", err)
	}
	if order.Saga.Status != domain.SagaStatusCompensating {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusCompensating, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepRevokeEntitlements {
		t.Errorf("expected saga step %s, got %s", domain.StepRevokeEntitlements, order.Saga.Step)
	}

	// Entitlements revoked -> move to refund payment
	err = order.OnEntitlementsRevoked(stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on OnEntitlementsRevoked: %v", err)
	}
	if order.Saga.Step != domain.StepRefundPayment {
		t.Errorf("expected saga step %s, got %s", domain.StepRefundPayment, order.Saga.Step)
	}

	// Payment refunded -> complete compensation
	err = order.CompleteCompensation()
	if err != nil {
		t.Fatalf("unexpected error on CompleteCompensation: %v", err)
	}
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("expected order status %s, got %s", domain.OrderStatusCancelled, order.Status)
	}
	if order.Saga.Status != domain.SagaStatusFailed {
		t.Errorf("expected saga status %s, got %s", domain.SagaStatusFailed, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepFailed {
		t.Errorf("expected saga step %s, got %s", domain.StepFailed, order.Saga.Step)
	}
}

func TestSaga_HandleTimeout(t *testing.T) {
	order := createTestOrder(t)
	paymentID := uuid.New()
	stepTimeout := 10 * time.Second

	_ = order.StartFulfillment(paymentID, stepTimeout)

	now := time.Now().UTC()
	// Not timed out yet
	if order.IsTimedOut(now) {
		t.Errorf("order should not be timed out yet")
	}
	_, err := order.HandleTimeout(now, stepTimeout)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict on non-timed-out order, got %v", err)
	}

	// Fast-forward past timeout
	future := now.Add(15 * time.Second)
	if !order.IsTimedOut(future) {
		t.Errorf("order should be timed out")
	}

	// Timeout on StepGrantEntitlements -> should trigger failure and move to StepRefundPayment
	nextStep, err := order.HandleTimeout(future, stepTimeout)
	if err != nil {
		t.Fatalf("unexpected error on HandleTimeout: %v", err)
	}
	if nextStep != domain.StepRefundPayment {
		t.Errorf("expected next step %s, got %s", domain.StepRefundPayment, nextStep)
	}
	if order.Saga.Status != domain.SagaStatusCompensating {
		t.Errorf("expected compensating status, got %s", order.Saga.Status)
	}
}

func TestSaga_InvalidTransitions(t *testing.T) {
	order := createTestOrder(t)

	// Cannot transition before start
	err := order.OnEntitlementsGranted(10 * time.Second)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}

	// Invalid input to StartFulfillment
	err = order.StartFulfillment(uuid.Nil, 10*time.Second)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for nil payment id, got %v", err)
	}
	err = order.StartFulfillment(uuid.New(), 0)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for non-positive timeout, got %v", err)
	}

	// Start properly
	_ = order.StartFulfillment(uuid.New(), 10*time.Second)

	// Cannot start again
	err = order.StartFulfillment(uuid.New(), 10*time.Second)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict when starting already running saga, got %v", err)
	}

	// Cannot complete before settling
	err = order.CompleteFulfillment()
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict when completing from GRANT_ENTITLEMENTS, got %v", err)
	}
}
