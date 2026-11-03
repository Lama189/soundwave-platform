package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/user_purchase"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
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

type mockPurchaseService struct {
	processPaidOrderFn func(ctx context.Context, input user_purchase.ProcessPaidOrderInput) error
}

func (m *mockPurchaseService) ProcessPaidOrder(ctx context.Context, input user_purchase.ProcessPaidOrderInput) error {
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
	trackID := uuid.New()

	validEvent := OrderPaidEvent{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		TotalAmount: 1000,
		Currency:    "USD",
		PaymentID:   paymentID,
		PaidAt:      time.Now().UTC(),
		Items: []OrderPaidItemEvent{
			{
				ItemID:   trackID,
				ItemType: itemtype.Track,
				ArtistID: uuid.New(),
				Price:    1000,
				Currency: "USD",
			},
		},
	}
	validBody, _ := json.Marshal(validEvent)

	tests := []struct {
		name          string
		body          []byte
		serviceFn     func(ctx context.Context, input user_purchase.ProcessPaidOrderInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name: "success processing",
			body: validBody,
			serviceFn: func(ctx context.Context, input user_purchase.ProcessPaidOrderInput) error {
				if input.OrderID != orderID {
					t.Errorf("expected order id %s, got %s", orderID, input.OrderID)
				}
				if input.BuyerUserID != buyerID {
					t.Errorf("expected buyer user id %s, got %s", buyerID, input.BuyerUserID)
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
			name: "service error triggers nack with requeue",
			body: validBody,
			serviceFn: func(ctx context.Context, input user_purchase.ProcessPaidOrderInput) error {
				return errors.New("db failure")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name:      "invalid json triggers ack to discard corrupt message",
			body:      []byte("{invalid json"),
			serviceFn: nil,
			expectAck:  true,
			expectNack: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPurchaseService{processPaidOrderFn: tt.serviceFn}
			consumer := NewOrderPaidConsumer(nil, svc, logger)

			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  123,
				Body:         tt.body,
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

func TestOrderPaidEvent_ToServiceInput(t *testing.T) {
	orderID := uuid.New()
	buyerID := uuid.New()
	trackID := uuid.New()
	albumID := uuid.New()

	event := OrderPaidEvent{
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []OrderPaidItemEvent{
			{
				ItemID:   trackID,
				ItemType: itemtype.Track,
			},
			{
				ItemID:   albumID,
				ItemType: itemtype.Album,
			},
		},
	}

	input := event.toServiceInput()

	if input.OrderID != orderID {
		t.Errorf("expected orderID %s, got %s", orderID, input.OrderID)
	}
	if input.BuyerUserID != buyerID {
		t.Errorf("expected buyerUserID %s, got %s", buyerID, input.BuyerUserID)
	}
	if len(input.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(input.Items))
	}
	if input.Items[0].ItemID != trackID || input.Items[0].ItemType != itemtype.Track {
		t.Errorf("unexpected item 0: %+v", input.Items[0])
	}
	if input.Items[1].ItemID != albumID || input.Items[1].ItemType != itemtype.Album {
		t.Errorf("unexpected item 1: %+v", input.Items[1])
	}
}
