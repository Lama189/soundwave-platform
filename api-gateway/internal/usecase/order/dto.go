package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/pkg/money"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid order input")

type CreateOrderItemInput struct {
	ItemType string
	ItemID   uuid.UUID
}

type CreateOrderInput struct {
	BuyerUserID uuid.UUID
	Items       []CreateOrderItemInput
}

func (in CreateOrderInput) Validate() error {
	if in.BuyerUserID == uuid.Nil {
		return fmt.Errorf("%w: buyer user id is required", ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return fmt.Errorf("%w: order must contain at least one item", ErrInvalidInput)
	}

	for _, it := range in.Items {
		if it.ItemID == uuid.Nil {
			return fmt.Errorf("%w: item id cannot be empty", ErrInvalidInput)
		}
		if it.ItemType != "track" && it.ItemType != "album" {
			return fmt.Errorf("%w: invalid item type", ErrInvalidInput)
		}
	}

	return nil
}

type GetOrderInput struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
}

type ListBuyerOrdersInput struct {
	BuyerUserID uuid.UUID
	Limit       int32
	Offset      int32
}

type CancelOrderInput struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
}

type OrderItemOutput struct {
	ID       uuid.UUID      `json:"id"`
	ItemType string         `json:"item_type"`
	ItemID   uuid.UUID      `json:"item_id"`
	ArtistID uuid.UUID      `json:"artist_id"`
	Price    int64          `json:"price"`
	Currency money.Currency `json:"currency"`
}

type OrderOutput struct {
	ID            uuid.UUID         `json:"id"`
	BuyerUserID   uuid.UUID         `json:"buyer_user_id"`
	Status        string            `json:"status"`
	TotalAmount   int64             `json:"total_amount"`
	Currency      money.Currency    `json:"currency"`
	PaymentID     *uuid.UUID        `json:"payment_id,omitempty"`
	FailureReason *string           `json:"failure_reason,omitempty"`
	Items         []OrderItemOutput `json:"items"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func ToOrderOutput(res *grpcclient.OrderResponse) *OrderOutput {
	if res == nil {
		return nil
	}

	items := make([]OrderItemOutput, len(res.Items))
	for i, it := range res.Items {
		items[i] = OrderItemOutput{
			ID:       it.ID,
			ItemType: it.ItemType,
			ItemID:   it.ItemID,
			ArtistID: it.ArtistID,
			Price:    it.Price,
			Currency: it.Currency,
		}
	}

	return &OrderOutput{
		ID:            res.ID,
		BuyerUserID:   res.BuyerUserID,
		Status:        res.Status,
		TotalAmount:   res.TotalAmount,
		Currency:      res.Currency,
		PaymentID:     res.PaymentID,
		FailureReason: res.FailureReason,
		Items:         items,
		CreatedAt:     res.CreatedAt,
		UpdatedAt:     res.UpdatedAt,
	}
}
