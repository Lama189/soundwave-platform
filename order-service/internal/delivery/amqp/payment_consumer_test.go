package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mockPaymentConsumerOrderService struct {
	payOrderCalls  []order.PayOrderDTO
	failOrderCalls []order.FailOrderDTO
	payOrderErr    error
	failOrderErr   error
}

func (m *mockPaymentConsumerOrderService) PayOrder(ctx context.Context, input order.PayOrderDTO) (*order.OrderOutputDTO, error) {
	m.payOrderCalls = append(m.payOrderCalls, input)
	if m.payOrderErr != nil {
		return nil, m.payOrderErr
	}
	return &order.OrderOutputDTO{ID: input.OrderID}, nil
}

func (m *mockPaymentConsumerOrderService) FailOrder(ctx context.Context, input order.FailOrderDTO) (*order.OrderOutputDTO, error) {
	m.failOrderCalls = append(m.failOrderCalls, input)
	if m.failOrderErr != nil {
		return nil, m.failOrderErr
	}
	return &order.OrderOutputDTO{ID: input.OrderID}, nil
}

func (m *mockPaymentConsumerOrderService) OnCatalogEntitlementsGranted(ctx context.Context, orderID uuid.UUID) error {
	return nil
}

func (m *mockPaymentConsumerOrderService) OnCatalogEntitlementsFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	return nil
}

func (m *mockPaymentConsumerOrderService) OnBillingSettled(ctx context.Context, orderID uuid.UUID) error {
	return nil
}

func (m *mockPaymentConsumerOrderService) OnBillingFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	return nil
}

func (m *mockPaymentConsumerOrderService) OnCatalogEntitlementsRevoked(ctx context.Context, orderID uuid.UUID) error {
	return nil
}

func (m *mockPaymentConsumerOrderService) OnPaymentRefunded(ctx context.Context, orderID uuid.UUID) error {
	return nil
}

func TestPaymentConsumer_HandleMessage_PaymentSucceeded(t *testing.T) {
	svc := &mockPaymentConsumerOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewPaymentConsumer(nil, svc, logger)

	orderID := uuid.New()
	paymentID := uuid.New()
	event := PaymentSucceededEvent{
		OrderID:   orderID,
		PaymentID: paymentID,
		Amount:    1000,
	}
	body, _ := json.Marshal(event)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyPaymentSuccess,
		Body:         body,
	})

	if len(svc.payOrderCalls) != 1 || svc.payOrderCalls[0].OrderID != orderID {
		t.Fatalf("expected 1 PayOrder call for %s, got %v", orderID, svc.payOrderCalls)
	}
	if !ack.acked || ack.nacked {
		t.Errorf("expected ack=true, nack=false, got ack=%v, nack=%v", ack.acked, ack.nacked)
	}
}

func TestPaymentConsumer_HandleMessage_AlreadyPaid_Ack(t *testing.T) {
	svc := &mockPaymentConsumerOrderService{payOrderErr: domain.ErrOrderAlreadyPaid}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewPaymentConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := PaymentSucceededEvent{
		OrderID:   orderID,
		PaymentID: uuid.New(),
		Amount:    1000,
	}
	body, _ := json.Marshal(event)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyPaymentSuccess,
		Body:         body,
	})

	if !ack.acked || ack.nacked {
		t.Errorf("expected ack=true on duplicate already paid, got ack=%v, nack=%v", ack.acked, ack.nacked)
	}
}

func TestPaymentConsumer_HandleMessage_TransientError_Requeue(t *testing.T) {
	svc := &mockPaymentConsumerOrderService{payOrderErr: errors.New("db down")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewPaymentConsumer(nil, svc, logger)

	orderID := uuid.New()
	event := PaymentSucceededEvent{
		OrderID:   orderID,
		PaymentID: uuid.New(),
		Amount:    1000,
	}
	body, _ := json.Marshal(event)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyPaymentSuccess,
		Body:         body,
	})

	if !ack.nacked || !ack.requeue {
		t.Errorf("expected nack with requeue=true, got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}

func TestPaymentConsumer_HandleMessage_MalformedJSON_DLQ(t *testing.T) {
	svc := &mockPaymentConsumerOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewPaymentConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   RoutingKeyPaymentSuccess,
		Body:         []byte("invalid json"),
	})

	if len(svc.payOrderCalls) != 0 {
		t.Fatalf("expected 0 calls, got %d", len(svc.payOrderCalls))
	}
	if !ack.nacked || ack.requeue {
		t.Errorf("expected dlq nack (requeue=false), got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}

func TestPaymentConsumer_HandleMessage_UnhandledRoutingKey_DLQ(t *testing.T) {
	svc := &mockPaymentConsumerOrderService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewPaymentConsumer(nil, svc, logger)

	ack := &mockAcknowledger{}
	consumer.handleMessage(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		RoutingKey:   "payment.unknown",
		Body:         []byte(`{}`),
	})

	if !ack.nacked || ack.requeue {
		t.Errorf("expected dlq nack (requeue=false), got nacked=%v, requeue=%v", ack.nacked, ack.requeue)
	}
}
