package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

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
	processGrantEntitlementsFn func(ctx context.Context, input user_purchase.GrantEntitlementsInput) error
	revokePurchasesFn          func(ctx context.Context, input user_purchase.RevokePurchasesInput) error
}

func (m *mockPurchaseService) ProcessGrantEntitlements(ctx context.Context, input user_purchase.GrantEntitlementsInput) error {
	if m.processGrantEntitlementsFn != nil {
		return m.processGrantEntitlementsFn(ctx, input)
	}
	return nil
}

func (m *mockPurchaseService) RevokePurchases(ctx context.Context, input user_purchase.RevokePurchasesInput) error {
	if m.revokePurchasesFn != nil {
		return m.revokePurchasesFn(ctx, input)
	}
	return nil
}

func TestOrderPaidConsumer_HandleGrantEntitlements(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()
	trackID := uuid.New()

	validCmd := GrantEntitlementsCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []GrantEntitlementsCommandItem{
			{
				ItemID:   trackID,
				ItemType: itemtype.Track,
			},
		},
	}
	validBody, _ := json.Marshal(validCmd)

	tests := []struct {
		name          string
		body          []byte
		serviceFn     func(ctx context.Context, input user_purchase.GrantEntitlementsInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name: "success processing",
			body: validBody,
			serviceFn: func(ctx context.Context, input user_purchase.GrantEntitlementsInput) error {
				if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID {
					t.Errorf("unexpected input identifiers: %+v", input)
				}
				if input.BuyerUserID != buyerID {
					t.Errorf("expected buyer user id %s, got %s", buyerID, input.BuyerUserID)
				}
				if len(input.Items) != 1 || input.Items[0].ItemID != trackID {
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
			serviceFn: func(ctx context.Context, input user_purchase.GrantEntitlementsInput) error {
				return errors.New("database connection failed")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name:          "invalid json triggers dlq rejection",
			body:          []byte("{invalid json"),
			serviceFn:     nil,
			expectAck:     false,
			expectNack:    true,
			expectRequeue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPurchaseService{processGrantEntitlementsFn: tt.serviceFn}
			consumer := NewOrderPaidConsumer(nil, svc, logger)

			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  123,
				RoutingKey:   RoutingKeyGrantEntitlements,
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

func TestOrderPaidConsumer_HandleRevokeEntitlements(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	validCmd := RevokeEntitlementsCommand{
		CommandID: commandID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "settlement failed",
	}
	validBody, _ := json.Marshal(validCmd)

	tests := []struct {
		name          string
		body          []byte
		serviceFn     func(ctx context.Context, input user_purchase.RevokePurchasesInput) error
		expectAck     bool
		expectNack    bool
		expectRequeue bool
	}{
		{
			name: "success revoking",
			body: validBody,
			serviceFn: func(ctx context.Context, input user_purchase.RevokePurchasesInput) error {
				if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID {
					t.Errorf("unexpected identifiers: %+v", input)
				}
				if input.Reason != "settlement failed" {
					t.Errorf("unexpected reason: %s", input.Reason)
				}
				return nil
			},
			expectAck:  true,
			expectNack: false,
		},
		{
			name: "service error triggers nack with requeue",
			body: validBody,
			serviceFn: func(ctx context.Context, input user_purchase.RevokePurchasesInput) error {
				return errors.New("db error")
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name:          "invalid json triggers dlq rejection",
			body:          []byte("{invalid json"),
			serviceFn:     nil,
			expectAck:     false,
			expectNack:    true,
			expectRequeue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockPurchaseService{revokePurchasesFn: tt.serviceFn}
			consumer := NewOrderPaidConsumer(nil, svc, logger)

			ack := &mockAcknowledger{}
			msg := amqp.Delivery{
				Acknowledger: ack,
				DeliveryTag:  456,
				RoutingKey:   RoutingKeyRevokeEntitlements,
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

func TestOrderPaidConsumer_UnhandledRoutingKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	consumer := NewOrderPaidConsumer(nil, &mockPurchaseService{}, logger)

	ack := &mockAcknowledger{}
	msg := amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  789,
		RoutingKey:   "unknown.routing.key",
		Body:         []byte("{}"),
	}

	consumer.handleMessage(context.Background(), msg)

	if ack.acked {
		t.Error("expected unhandled routing key not to be acknowledged")
	}
	if !ack.nacked {
		t.Error("expected unhandled routing key to be nacked for dlq")
	}
	if ack.requeue {
		t.Error("expected unhandled routing key not to be requeued")
	}
}

func TestGrantEntitlementsCommand_ToServiceInput(t *testing.T) {
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()
	buyerID := uuid.New()
	trackID := uuid.New()
	albumID := uuid.New()

	cmd := GrantEntitlementsCommand{
		CommandID:   commandID,
		SagaID:      sagaID,
		OrderID:     orderID,
		BuyerUserID: buyerID,
		Items: []GrantEntitlementsCommandItem{
			{ItemID: trackID, ItemType: itemtype.Track},
			{ItemID: albumID, ItemType: itemtype.Album},
		},
	}

	input := cmd.ToServiceInput()

	if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID || input.BuyerUserID != buyerID {
		t.Errorf("unexpected converted fields: %+v", input)
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

func TestRevokeEntitlementsCommand_ToServiceInput(t *testing.T) {
	commandID := uuid.New()
	sagaID := uuid.New()
	orderID := uuid.New()

	cmd := RevokeEntitlementsCommand{
		CommandID: commandID,
		SagaID:    sagaID,
		OrderID:   orderID,
		Reason:    "failed",
	}

	input := cmd.ToServiceInput()

	if input.CommandID != commandID || input.SagaID != sagaID || input.OrderID != orderID || input.Reason != "failed" {
		t.Errorf("unexpected converted fields: %+v", input)
	}
}
