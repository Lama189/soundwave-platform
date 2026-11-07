package order

import (
	"context"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/uow"
	"github.com/google/uuid"
)

type mockOrderRepo struct {
	orders map[uuid.UUID]*domain.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[uuid.UUID]*domain.Order),
	}
}

func (m *mockOrderRepo) Create(ctx context.Context, order *domain.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return m.GetByID(ctx, id)
}

func (m *mockOrderRepo) GetByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	var res []*domain.Order
	for _, o := range m.orders {
		if o.BuyerUserID == buyerID {
			res = append(res, o)
		}
	}
	return res, nil
}

func (m *mockOrderRepo) GetTimedOut(ctx context.Context, now time.Time, limit int) ([]*domain.Order, error) {
	var res []*domain.Order
	for _, o := range m.orders {
		if o.IsTimedOut(now) {
			res = append(res, o)
		}
	}
	return res, nil
}

func (m *mockOrderRepo) Update(ctx context.Context, order *domain.Order) error {
	m.orders[order.ID] = order
	return nil
}

type mockOutboxRepo struct {
	events []*domain.Event
}

func (m *mockOutboxRepo) Record(ctx context.Context, event *domain.Event) error {
	m.events = append(m.events, event)
	return nil
}

type mockUnitOfWork struct {
	orderRepo  *mockOrderRepo
	outboxRepo *mockOutboxRepo
}

func (m *mockUnitOfWork) Orders() uow.OrderRepository {
	return m.orderRepo
}

func (m *mockUnitOfWork) Outbox() uow.OutboxRepository {
	return m.outboxRepo
}

type mockUoWManager struct {
	work *mockUnitOfWork
}

func (m *mockUoWManager) Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error {
	return fn(m.work)
}

func createTestOrder(t *testing.T) *domain.Order {
	t.Helper()
	buyerID := uuid.New()
	item, err := domain.NewOrderItem(uuid.Nil, domain.ItemTypeTrack, uuid.New(), uuid.New(), 1000, domain.CurrencyUSD)
	if err != nil {
		t.Fatalf("unexpected error creating item: %v", err)
	}
	o, err := domain.NewOrder(buyerID, []domain.OrderItem{*item})
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}
	return o
}

func setupService(t *testing.T, order *domain.Order) (*Service, *mockOrderRepo, *mockOutboxRepo) {
	t.Helper()
	orderRepo := newMockOrderRepo()
	if order != nil {
		orderRepo.orders[order.ID] = order
	}
	outboxRepo := &mockOutboxRepo{}
	uowWork := &mockUnitOfWork{
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
	}
	uowManager := &mockUoWManager{work: uowWork}

	svc := NewService(uowManager, orderRepo, nil, 10*time.Second, 2)
	return svc, orderRepo, outboxRepo
}

func TestService_PayOrder_StartsSaga(t *testing.T) {
	order := createTestOrder(t)
	svc, orderRepo, outboxRepo := setupService(t, order)

	paymentID := uuid.New()
	out, err := svc.PayOrder(context.Background(), PayOrderDTO{
		OrderID:   order.ID,
		PaymentID: paymentID,
	})
	if err != nil {
		t.Fatalf("unexpected error on PayOrder: %v", err)
	}
	if out.Status != domain.OrderStatusProcessing {
		t.Fatalf("expected order status %s, got %s", domain.OrderStatusProcessing, out.Status)
	}

	updated := orderRepo.orders[order.ID]
	if updated.Saga.Step != domain.StepGrantEntitlements {
		t.Fatalf("expected saga step %s, got %s", domain.StepGrantEntitlements, updated.Saga.Step)
	}
	if len(outboxRepo.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[0].EventType != domain.EventTypeCmdCatalogGrantEntitlements {
		t.Fatalf("expected outbox event %s, got %s", domain.EventTypeCmdCatalogGrantEntitlements, outboxRepo.events[0].EventType)
	}
}

func TestService_SagaHappyPath(t *testing.T) {
	order := createTestOrder(t)
	svc, orderRepo, outboxRepo := setupService(t, order)

	paymentID := uuid.New()
	_, _ = svc.PayOrder(context.Background(), PayOrderDTO{
		OrderID:   order.ID,
		PaymentID: paymentID,
	})

	err := svc.OnCatalogEntitlementsGranted(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on OnCatalogEntitlementsGranted: %v", err)
	}

	updated := orderRepo.orders[order.ID]
	if updated.Saga.Step != domain.StepSettleBilling {
		t.Fatalf("expected saga step %s, got %s", domain.StepSettleBilling, updated.Saga.Step)
	}
	if len(outboxRepo.events) != 2 {
		t.Fatalf("expected 2 outbox events, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[1].EventType != domain.EventTypeCmdBillingSettleOrder {
		t.Fatalf("expected outbox event %s, got %s", domain.EventTypeCmdBillingSettleOrder, outboxRepo.events[1].EventType)
	}

	err = svc.OnBillingSettled(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on OnBillingSettled: %v", err)
	}

	updated = orderRepo.orders[order.ID]
	if updated.Status != domain.OrderStatusPaid {
		t.Fatalf("expected order status %s, got %s", domain.OrderStatusPaid, updated.Status)
	}
	if updated.Saga.Status != domain.SagaStatusCompleted {
		t.Fatalf("expected saga status %s, got %s", domain.SagaStatusCompleted, updated.Saga.Status)
	}
}

func TestService_SagaBillingFailed_LIFOCompensation(t *testing.T) {
	order := createTestOrder(t)
	svc, orderRepo, outboxRepo := setupService(t, order)

	paymentID := uuid.New()
	_, _ = svc.PayOrder(context.Background(), PayOrderDTO{
		OrderID:   order.ID,
		PaymentID: paymentID,
	})
	_ = svc.OnCatalogEntitlementsGranted(context.Background(), order.ID)

	err := svc.OnBillingFailed(context.Background(), order.ID, "insufficient funds")
	if err != nil {
		t.Fatalf("unexpected error on OnBillingFailed: %v", err)
	}

	updated := orderRepo.orders[order.ID]
	if updated.Saga.Step != domain.StepRevokeEntitlements {
		t.Fatalf("expected saga step %s, got %s", domain.StepRevokeEntitlements, updated.Saga.Step)
	}
	if len(outboxRepo.events) != 3 {
		t.Fatalf("expected 3 outbox events, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[2].EventType != domain.EventTypeCmdCatalogRevokeEntitlements {
		t.Fatalf("expected outbox event %s, got %s", domain.EventTypeCmdCatalogRevokeEntitlements, outboxRepo.events[2].EventType)
	}

	err = svc.OnCatalogEntitlementsRevoked(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on OnCatalogEntitlementsRevoked: %v", err)
	}

	updated = orderRepo.orders[order.ID]
	if updated.Saga.Step != domain.StepRefundPayment {
		t.Fatalf("expected saga step %s, got %s", domain.StepRefundPayment, updated.Saga.Step)
	}
	if len(outboxRepo.events) != 4 {
		t.Fatalf("expected 4 outbox events, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[3].EventType != domain.EventTypeCmdPaymentRefund {
		t.Fatalf("expected outbox event %s, got %s", domain.EventTypeCmdPaymentRefund, outboxRepo.events[3].EventType)
	}

	err = svc.OnPaymentRefunded(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on OnPaymentRefunded: %v", err)
	}

	updated = orderRepo.orders[order.ID]
	if updated.Status != domain.OrderStatusCancelled {
		t.Fatalf("expected order status %s, got %s", domain.OrderStatusCancelled, updated.Status)
	}
	if updated.Saga.Status != domain.SagaStatusFailed {
		t.Fatalf("expected saga status %s, got %s", domain.SagaStatusFailed, updated.Saga.Status)
	}
}

func TestService_HandleTimeout_RetryThenCompensate(t *testing.T) {
	order := createTestOrder(t)
	svc, orderRepo, outboxRepo := setupService(t, order)

	paymentID := uuid.New()
	_, _ = svc.PayOrder(context.Background(), PayOrderDTO{
		OrderID:   order.ID,
		PaymentID: paymentID,
	})

	past := time.Now().UTC().Add(-20 * time.Second)
	order.Saga.StepTimeoutAt = &past

	err := svc.HandleTimeout(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on timeout retry 1: %v", err)
	}
	if len(outboxRepo.events) != 2 {
		t.Fatalf("expected 2 outbox events after retry 1, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[1].EventType != domain.EventTypeCmdCatalogGrantEntitlements {
		t.Fatalf("expected retry event %s, got %s", domain.EventTypeCmdCatalogGrantEntitlements, outboxRepo.events[1].EventType)
	}

	order = orderRepo.orders[order.ID]
	if order.Saga.StepRetries != 1 {
		t.Fatalf("expected step retries 1, got %d", order.Saga.StepRetries)
	}

	order.Saga.StepTimeoutAt = &past
	err = svc.HandleTimeout(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on timeout retry 2: %v", err)
	}
	if len(outboxRepo.events) != 3 {
		t.Fatalf("expected 3 outbox events after retry 2, got %d", len(outboxRepo.events))
	}
	if order.Saga.StepRetries != 2 {
		t.Fatalf("expected step retries 2, got %d", order.Saga.StepRetries)
	}

	order.Saga.StepTimeoutAt = &past
	err = svc.HandleTimeout(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error on timeout compensation: %v", err)
	}
	if len(outboxRepo.events) != 4 {
		t.Fatalf("expected 4 outbox events after exhaustion, got %d", len(outboxRepo.events))
	}
	if outboxRepo.events[3].EventType != domain.EventTypeCmdPaymentRefund {
		t.Fatalf("expected compensation event %s, got %s", domain.EventTypeCmdPaymentRefund, outboxRepo.events[3].EventType)
	}

	order = orderRepo.orders[order.ID]
	if order.Saga.Status != domain.SagaStatusCompensating {
		t.Fatalf("expected saga status %s, got %s", domain.SagaStatusCompensating, order.Saga.Status)
	}
	if order.Saga.Step != domain.StepRefundPayment {
		t.Fatalf("expected saga step %s, got %s", domain.StepRefundPayment, order.Saga.Step)
	}
}
