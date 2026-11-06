package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
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

type mockSettlementService struct {
	processSettleOrderFn func(ctx context.Context, input settlement.SettleOrderInput) error
}

func (m *mockSettlementService) ProcessSettleOrder(ctx context.Context, input settlement.SettleOrderInput) error {
	if m.processSettleOrderFn != nil {
		return m.processSettleOrderFn(ctx, input)
	}
	return nil
}

func TestOrderPaidConsumer_HandleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()
	paymentID := uuid.New()
	artistID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()

	validCmd := SettleOrderCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 1000,
		Currency:    domain.CurrencyRUB,
		PaymentID:   paymentID,
		PaidAt:      now,
		Items: []SettleOrderCommandItem{
			{
				ItemID:   itemID,
				ItemType: "track",
				ArtistID: artistID,
				Price:    1000,
				Currency: domain.CurrencyRUB,
			},
		},
	}
	validBody, _ := json.Marshal(validCmd)

	tests := []struct {
		name          string
		routingKey    string
		body          []byte
		serviceFn     func(ctx context.Context, input settlement.SettleOrderInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name:       "success processing",
			routingKey: RoutingKeySettleOrder,
			body:       validBody,
			serviceFn: func(ctx context.Context, input settlement.SettleOrderInput) error {
				if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID {
					t.Errorf("unexpected identifiers: %+v", input)
				}
				if len(input.Items) != 1 || input.Items[0].ItemType != itemtype.Track {
					t.Errorf("unexpected items: %+v", input.Items)
				}
				return nil
			},
			expectAck:  true,
			expectNack: false,
		},
		{
			name:       "invalid json triggers ack to discard corrupt message",
			routingKey: RoutingKeySettleOrder,
			body:       []byte("invalid-json{"),
			expectAck:  true,
			expectNack: false,
		},
		{
			name:       "service transient error triggers nack with requeue",
			routingKey: RoutingKeySettleOrder,
			body:       validBody,
			serviceFn: func(ctx context.Context, input settlement.SettleOrderInput) error {
				return errors.New("db connection failure")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name:       "unhandled routing key triggers ack",
			routingKey: "other.routing.key",
			body:       validBody,
			expectAck:  true,
			expectNack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  42,
				RoutingKey:   tt.routingKey,
				Body:         tt.body,
			}

			svc := &mockSettlementService{processSettleOrderFn: tt.serviceFn}
			consumer := &OrderPaidConsumer{
				settlementService: svc,
				logger:            logger,
			}

			consumer.handleMessage(context.Background(), msg)

			if tt.expectAck && !ack.acked {
				t.Error("expected message to be Acked, but it was not")
			}
			if tt.expectNack && !ack.nacked {
				t.Error("expected message to be Nacked, but it was not")
			}
			if tt.expectRequeue != ack.requeue {
				t.Errorf("expected requeue=%v, got %v", tt.expectRequeue, ack.requeue)
			}
		})
	}
}

func TestSettleOrderCommand_ToServiceInput(t *testing.T) {
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()
	paymentID := uuid.New()
	artistID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()

	cmd := SettleOrderCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 2500,
		Currency:    domain.CurrencyRUB,
		PaymentID:   paymentID,
		PaidAt:      now,
		Items: []SettleOrderCommandItem{
			{
				ItemID:   itemID,
				ItemType: "album",
				ArtistID: artistID,
				Price:    2500,
				Currency: domain.CurrencyRUB,
			},
		},
	}

	input := cmd.ToServiceInput()
	if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID {
		t.Errorf("unexpected identifiers: %+v", input)
	}
	if input.BuyerUserID != buyerID {
		t.Errorf("expected buyer user id %s, got %s", buyerID, input.BuyerUserID)
	}
	if input.TotalAmount != 2500 {
		t.Errorf("expected total amount 2500, got %d", input.TotalAmount)
	}
	if len(input.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(input.Items))
	}
	if input.Items[0].ItemType != itemtype.Album {
		t.Errorf("expected item type %s, got %s", itemtype.Album, input.Items[0].ItemType)
	}
}
