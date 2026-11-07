package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
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

type mockPaymentService struct {
	processRefundCommandFn func(ctx context.Context, input payment.RefundCommandInput) error
}

func (m *mockPaymentService) ProcessRefundCommand(ctx context.Context, input payment.RefundCommandInput) error {
	if m.processRefundCommandFn != nil {
		return m.processRefundCommandFn(ctx, input)
	}
	return nil
}

func TestOrderCommandConsumer_HandleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	validCmd := RefundCommand{
		CommandID: commandID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "settlement failed",
	}
	validBody, _ := json.Marshal(validCmd)

	tests := []struct {
		name          string
		routingKey    string
		body          []byte
		serviceFn     func(ctx context.Context, input payment.RefundCommandInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name:       "success processing",
			routingKey: RoutingKeyRefundPayment,
			body:       validBody,
			serviceFn: func(ctx context.Context, input payment.RefundCommandInput) error {
				if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID {
					t.Errorf("unexpected input identifiers: %+v", input)
				}
				if input.Reason != "settlement failed" {
					t.Errorf("expected reason 'settlement failed', got %s", input.Reason)
				}
				return nil
			},
			expectAck:  true,
			expectNack: false,
		},
		{
			name:       "service error triggers nack with requeue",
			routingKey: RoutingKeyRefundPayment,
			body:       validBody,
			serviceFn: func(ctx context.Context, input payment.RefundCommandInput) error {
				return errors.New("db error")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name:          "invalid json triggers dlq rejection",
			routingKey:    RoutingKeyRefundPayment,
			body:          []byte("invalid-json{"),
			expectAck:     false,
			expectNack:    true,
			expectRequeue: false,
		},
		{
			name:          "unhandled routing key triggers dlq rejection",
			routingKey:    "other.key",
			body:          validBody,
			expectAck:     false,
			expectNack:    true,
			expectRequeue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  99,
				RoutingKey:   tt.routingKey,
				Body:         tt.body,
			}

			svc := &mockPaymentService{processRefundCommandFn: tt.serviceFn}
			consumer := &OrderCommandConsumer{
				paymentService: svc,
				logger:         logger,
			}

			consumer.handleMessage(context.Background(), msg)

			if ack.acked != tt.expectAck {
				t.Errorf("expected ack = %v, got %v", tt.expectAck, ack.acked)
			}
			if ack.nacked != tt.expectNack {
				t.Errorf("expected nack = %v, got %v", tt.expectNack, ack.nacked)
			}
			if ack.requeue != tt.expectRequeue {
				t.Errorf("expected requeue = %v, got %v", tt.expectRequeue, ack.requeue)
			}
		})
	}
}

func TestRefundCommand_ToServiceInput(t *testing.T) {
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	cmd := RefundCommand{
		CommandID: commandID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "timeout",
	}

	input := cmd.ToServiceInput()
	if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID || input.Reason != "timeout" {
		t.Errorf("unexpected mapped fields: %+v", input)
	}
}
