package grpcclient

import (
	"time"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

type GetOrderRequest struct {
	OrderID     uuid.UUID
	BuyerUserID *uuid.UUID
}

type OrderItemResponse struct {
	ID       uuid.UUID
	ItemType string
	ItemID   uuid.UUID
	ArtistID uuid.UUID
	Price    int64
	Currency domain.Currency
}

type OrderResponse struct {
	ID            uuid.UUID
	BuyerUserID   uuid.UUID
	Status        string
	TotalAmount   int64
	Currency      domain.Currency
	PaymentID     *uuid.UUID
	FailureReason *string
	Items         []OrderItemResponse
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PayOrderRequest struct {
	OrderID   uuid.UUID
	PaymentID uuid.UUID
}

func toOrderResponse(pb *orderpb.Order) *OrderResponse {
	orderID, _ := uuid.Parse(pb.GetId())
	buyerID, _ := uuid.Parse(pb.GetBuyerUserId())

	var paymentID *uuid.UUID
	if pb.PaymentId != nil && *pb.PaymentId != "" {
		if parsed, err := uuid.Parse(*pb.PaymentId); err == nil {
			paymentID = &parsed
		}
	}

	var createdAt time.Time
	if pb.GetCreatedAt() != nil {
		createdAt = pb.GetCreatedAt().AsTime()
	}

	var updatedAt time.Time
	if pb.GetUpdatedAt() != nil {
		updatedAt = pb.GetUpdatedAt().AsTime()
	}

	items := make([]OrderItemResponse, len(pb.GetItems()))
	for i, it := range pb.GetItems() {
		itID, _ := uuid.Parse(it.GetId())
		itemID, _ := uuid.Parse(it.GetItemId())
		artistID, _ := uuid.Parse(it.GetArtistId())

		items[i] = OrderItemResponse{
			ID:       itID,
			ItemType: it.GetItemType(),
			ItemID:   itemID,
			ArtistID: artistID,
			Price:    it.GetPrice(),
			Currency: domain.Currency(it.GetCurrency()),
		}
	}

	return &OrderResponse{
		ID:            orderID,
		BuyerUserID:   buyerID,
		Status:        pb.GetStatus(),
		TotalAmount:   pb.GetTotalAmount(),
		Currency:      domain.Currency(pb.GetCurrency()),
		PaymentID:     paymentID,
		FailureReason: pb.FailureReason,
		Items:         items,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
