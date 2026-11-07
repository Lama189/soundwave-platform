package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mockAcknowledger struct {
	acked       bool
	nacked      bool
	requeue     bool
	deliveryTag uint64
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.acked = true
	m.deliveryTag = tag
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	m.nacked = true
	m.requeue = requeue
	m.deliveryTag = tag
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	m.nacked = true
	m.requeue = requeue
	return nil
}

type mockOrderService struct {
	grantedCalls     []uuid.UUID
	grantFailedCalls []struct {
		ID     uuid.UUID
		Reason string
	}
	settledCalls       []uuid.UUID
	billingFailedCalls []struct {
		ID     uuid.UUID
		Reason string
	}
	revokedCalls  []uuid.UUID
	refundedCalls []uuid.UUID
	err           error
}

func (m *mockOrderService) PayOrder(ctx context.Context, input order.PayOrderDTO) (*order.OrderOutputDTO, error) {
	return nil, nil
}

func (m *mockOrderService) FailOrder(ctx context.Context, input order.FailOrderDTO) (*order.OrderOutputDTO, error) {
	return nil, nil
}

func (m *mockOrderService) OnCatalogEntitlementsGranted(ctx context.Context, orderID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.grantedCalls = append(m.grantedCalls, orderID)
	return nil
}

func (m *mockOrderService) OnCatalogEntitlementsFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	if m.err != nil {
		return m.err
	}
	m.grantFailedCalls = append(m.grantFailedCalls, struct {
		ID     uuid.UUID
		Reason string
	}{ID: orderID, Reason: reason})
	return nil
}

func (m *mockOrderService) OnBillingSettled(ctx context.Context, orderID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.settledCalls = append(m.settledCalls, orderID)
	return nil
}

func (m *mockOrderService) OnBillingFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	if m.err != nil {
		return m.err
	}
	m.billingFailedCalls = append(m.billingFailedCalls, struct {
		ID     uuid.UUID
		Reason string
	}{ID: orderID, Reason: reason})
	return nil
}

func (m *mockOrderService) OnCatalogEntitlementsRevoked(ctx context.Context, orderID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.revokedCalls = append(m.revokedCalls, orderID)
	return nil
}

func (m *mockOrderService) OnPaymentRefunded(ctx context.Context, orderID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.refundedCalls = append(m.refundedCalls, orderID)
	return nil
}

func TestSagaReplyConsumer_HandleMessage_CatalogGranted(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   true,
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyCatalogGranted,
		Body:       body,
	})

	if len(svc.grantedCalls) != 1 || svc.grantedCalls[0] != orderID {
		t.Fatalf("expected 1 OnCatalogEntitlementsGranted call for %s, got %v", orderID, svc.grantedCalls)
	}
}

func TestSagaReplyConsumer_HandleMessage_CatalogGrantFailed(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   false,
		Error:     "track unavailable",
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyCatalogGrantFailed,
		Body:       body,
	})

	if len(svc.grantFailedCalls) != 1 {
		t.Fatalf("expected 1 OnCatalogEntitlementsFailed call, got %d", len(svc.grantFailedCalls))
	}
	if svc.grantFailedCalls[0].ID != orderID || svc.grantFailedCalls[0].Reason != "track unavailable" {
		t.Fatalf("unexpected call params: %+v", svc.grantFailedCalls[0])
	}
}

func TestSagaReplyConsumer_HandleMessage_BillingSettled(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   true,
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyBillingSettled,
		Body:       body,
	})

	if len(svc.settledCalls) != 1 || svc.settledCalls[0] != orderID {
		t.Fatalf("expected 1 OnBillingSettled call for %s, got %v", orderID, svc.settledCalls)
	}
}

func TestSagaReplyConsumer_HandleMessage_BillingFailed(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   false,
		Error:     "insufficient funds",
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyBillingFailed,
		Body:       body,
	})

	if len(svc.billingFailedCalls) != 1 {
		t.Fatalf("expected 1 OnBillingFailed call, got %d", len(svc.billingFailedCalls))
	}
	if svc.billingFailedCalls[0].ID != orderID || svc.billingFailedCalls[0].Reason != "insufficient funds" {
		t.Fatalf("unexpected call params: %+v", svc.billingFailedCalls[0])
	}
}

func TestSagaReplyConsumer_HandleMessage_CatalogRevoked(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   true,
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyCatalogRevoked,
		Body:       body,
	})

	if len(svc.revokedCalls) != 1 || svc.revokedCalls[0] != orderID {
		t.Fatalf("expected 1 OnCatalogEntitlementsRevoked call for %s, got %v", orderID, svc.revokedCalls)
	}
}

func TestSagaReplyConsumer_HandleMessage_PaymentRefunded(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   true,
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		RoutingKey: RoutingKeyPaymentRefunded,
		Body:       body,
	})

	if len(svc.refundedCalls) != 1 || svc.refundedCalls[0] != orderID {
		t.Fatalf("expected 1 OnPaymentRefunded call for %s, got %v", orderID, svc.refundedCalls)
	}
}

func TestSagaReplyConsumer_HandleMessage_MalformedJSON(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyCatalogGranted,
		Body:         []byte("invalid json"),
	})

	if len(svc.grantedCalls) != 0 {
		t.Fatalf("expected 0 calls for malformed JSON, got %d", len(svc.grantedCalls))
	}
	if !ack.nacked || ack.requeue {
		t.Errorf("expected dlq nack (requeue=false), got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}

func TestSagaReplyConsumer_HandleMessage_UnhandledRoutingKey(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "unknown.routing.key",
		Body:         []byte(`{"order_id":"00000000-0000-0000-0000-000000000000"}`),
	})

	if !ack.nacked || ack.requeue {
		t.Errorf("expected dlq nack (requeue=false), got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}

func TestSagaReplyConsumer_HandleMessage_PaymentRefundFailed_DLQ(t *testing.T) {
	svc := &mockOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   false,
		Error:     "payment gateway error",
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyPaymentRefundFailed,
		Body:         body,
	})

	if !ack.nacked || ack.requeue {
		t.Errorf("expected dlq nack (requeue=false), got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}

func TestSagaReplyConsumer_HandleMessage_TransientError_Requeue(t *testing.T) {
	svc := &mockOrderService{err: errors.New("temporary db error")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewSagaReplyConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	orderID := uuid.New()
	event := SagaReplyEvent{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Success:   true,
	}
	body, _ := json.Marshal(event)

	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyCatalogGranted,
		Body:         body,
	})

	if !ack.nacked || !ack.requeue {
		t.Errorf("expected nack with requeue=true, got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}
