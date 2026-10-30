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
	processPaidOrderFn func(ctx context.Context, input settlement.ProcessPaidOrderInput) error
}

func (m *mockSettlementService) ProcessPaidOrder(ctx context.Context, input settlement.ProcessPaidOrderInput) error {
	if m.processPaidOrderFn != nil {
		return m.processPaidOrderFn(ctx, input)
	}
	return nil
}

func TestOrderPaidConsumer_HandleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	orderID := uuid.New()
	buyerID := uuid.New()
	paymentID := uuid.New()
	artistID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()

	validEvent := OrderPaidEvent{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 1000,
		Currency:    domain.CurrencyRUB,
		PaymentID:   paymentID,
		PaidAt:      now,
		Items: []OrderPaidItemEvent{
			{
				ItemID:   itemID,
				ItemType: "track",
				ArtistID: artistID,
				Price:    1000,
				Currency: domain.CurrencyRUB,
			},
		},
	}
	validBody, _ := json.Marshal(validEvent)

	tests := []struct {
		name          string
		body          []byte
		serviceFn     func(ctx context.Context, input settlement.ProcessPaidOrderInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name: "success processing",
			body: validBody,
			serviceFn: func(ctx context.Context, input settlement.ProcessPaidOrderInput) error {
				if input.OrderID != orderID {
					t.Errorf("expected order id %s, got %s", orderID, input.OrderID)
				}
				if len(input.Items) != 1 || input.Items[0].ItemType != settlement.ItemTypeTrack {
					t.Errorf("unexpected items: %+v", input.Items)
				}
				return nil
			},
			expectAck:  true,
			expectNack: false,
		},
		{
			name: "already processed event (idempotency)",
			body: validBody,
			serviceFn: func(ctx context.Context, input settlement.ProcessPaidOrderInput) error {
				return domain.ErrOrderAlreadyProcessed
			},
			expectAck:  true,
			expectNack: false,
		},
		{
			name:       "invalid json (poison pill)",
			body:       []byte("invalid-json{"),
			expectAck:  true,
			expectNack: false,
		},
		{
			name: "service transient error (should Nack and requeue)",
			body: validBody,
			serviceFn: func(ctx context.Context, input settlement.ProcessPaidOrderInput) error {
				return errors.New("db connection failure")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  42,
				Body:         tt.body,
			}

			svc := &mockSettlementService{processPaidOrderFn: tt.serviceFn}
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

func TestOrderPaidEvent_ToSettlementInput(t *testing.T) {
	orderID := uuid.New()
	buyerID := uuid.New()
	paymentID := uuid.New()
	artistID := uuid.New()
	itemID := uuid.New()
	now := time.Now().UTC()

	event := OrderPaidEvent{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 2500,
		Currency:    domain.CurrencyRUB,
		PaymentID:   paymentID,
		PaidAt:      now,
		Items: []OrderPaidItemEvent{
			{
				ItemID:   itemID,
				ItemType: "album",
				ArtistID: artistID,
				Price:    2500,
				Currency: domain.CurrencyRUB,
			},
		},
	}

	input := event.toSettlementInput()
	if input.OrderID != orderID {
		t.Errorf("expected order id %s, got %s", orderID, input.OrderID)
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
	if input.Items[0].ItemType != settlement.ItemTypeAlbum {
		t.Errorf("expected item type %s, got %s", settlement.ItemTypeAlbum, input.Items[0].ItemType)
	}
}
