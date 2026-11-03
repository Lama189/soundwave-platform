package order_test

import (
	"errors"
	"testing"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/order"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

func TestCreateOrderInput_Validate(t *testing.T) {
	buyerID := uuid.New()
	validItemID := uuid.New()

	tests := []struct {
		name    string
		input   order.CreateOrderInput
		wantErr bool
	}{
		{
			name: "valid track item",
			input: order.CreateOrderInput{
				BuyerUserID: buyerID,
				Items: []order.CreateOrderItemInput{
					{
						ItemType: itemtype.Track,
						ItemID:   validItemID,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid album item",
			input: order.CreateOrderInput{
				BuyerUserID: buyerID,
				Items: []order.CreateOrderItemInput{
					{
						ItemType: itemtype.Album,
						ItemID:   validItemID,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing buyer user id",
			input: order.CreateOrderInput{
				BuyerUserID: uuid.Nil,
				Items: []order.CreateOrderItemInput{
					{
						ItemType: itemtype.Track,
						ItemID:   validItemID,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty items",
			input: order.CreateOrderInput{
				BuyerUserID: buyerID,
				Items:       []order.CreateOrderItemInput{},
			},
			wantErr: true,
		},
		{
			name: "nil item id",
			input: order.CreateOrderInput{
				BuyerUserID: buyerID,
				Items: []order.CreateOrderItemInput{
					{
						ItemType: itemtype.Track,
						ItemID:   uuid.Nil,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid item type",
			input: order.CreateOrderInput{
				BuyerUserID: buyerID,
				Items: []order.CreateOrderItemInput{
					{
						ItemType: "invalid_type",
						ItemID:   validItemID,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, order.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}
