package order

import (
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/google/uuid"
)

type CreateOrderItemDTO struct {
	ItemType domain.ItemType
	ItemID   uuid.UUID
	ArtistID uuid.UUID
	Price    int64
	Currency domain.Currency
}

type CreateOrderDTO struct {
	BuyerUserID uuid.UUID
	Items       []CreateOrderItemDTO
}

type GetOrderDTO struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
}

type ListBuyerOrdersDTO struct {
	BuyerUserID uuid.UUID
	Limit       int
	Offset      int
}

type PayOrderDTO struct {
	OrderID   uuid.UUID
	PaymentID uuid.UUID
}

type CancelOrderDTO struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
}

type OrderItemOutputDTO struct {
	ID       uuid.UUID        `json:"id"`
	OrderID  uuid.UUID        `json:"order_id"`
	ItemType domain.ItemType  `json:"item_type"`
	ItemID   uuid.UUID        `json:"item_id"`
	ArtistID uuid.UUID        `json:"artist_id"`
	Price    int64            `json:"price"`
	Currency domain.Currency  `json:"currency"`
}

type OrderOutputDTO struct {
	ID            uuid.UUID            `json:"id"`
	BuyerUserID   uuid.UUID            `json:"buyer_user_id"`
	Status        domain.OrderStatus   `json:"status"`
	TotalAmount   int64                `json:"total_amount"`
	Currency      domain.Currency      `json:"currency"`
	PaymentID     *uuid.UUID           `json:"payment_id,omitempty"`
	FailureReason *string              `json:"failure_reason,omitempty"`
	Items         []OrderItemOutputDTO `json:"items"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type ListOrdersOutputDTO struct {
	Orders []*OrderOutputDTO `json:"orders"`
}

type OrderPaidItemEventPayload struct {
	ItemID   uuid.UUID        `json:"item_id"`
	ItemType domain.ItemType  `json:"item_type"`
	ArtistID uuid.UUID        `json:"artist_id"`
	Price    int64            `json:"price"`
	Currency domain.Currency  `json:"currency"`
}

type OrderPaidEventPayload struct {
	OrderID     uuid.UUID                   `json:"order_id"`
	BuyerUserID uuid.UUID                   `json:"buyer_user_id"`
	TotalAmount int64                       `json:"total_amount"`
	Currency    domain.Currency             `json:"currency"`
	PaymentID   uuid.UUID                   `json:"payment_id"`
	PaidAt      time.Time                   `json:"paid_at"`
	Items       []OrderPaidItemEventPayload `json:"items"`
}

func toOrderOutputDTO(o *domain.Order) *OrderOutputDTO {
	if o == nil {
		return nil
	}

	return &OrderOutputDTO{
		ID:            o.ID,
		BuyerUserID:   o.BuyerUserID,
		Status:        o.Status,
		TotalAmount:   o.TotalAmount,
		Currency:      o.Currency,
		PaymentID:     o.PaymentID,
		FailureReason: o.FailureReason,
		Items:         toOrderItemOutputDTOs(o.Items),
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
}

func toOrderItemOutputDTOs(items []domain.OrderItem) []OrderItemOutputDTO {
	if items == nil {
		return nil
	}

	dtos := make([]OrderItemOutputDTO, len(items))
	for i, item := range items {
		dtos[i] = OrderItemOutputDTO{
			ID:       item.ID,
			OrderID:  item.OrderID,
			ItemType: item.ItemType,
			ItemID:   item.ItemID,
			ArtistID: item.ArtistID,
			Price:    item.Price,
			Currency: item.Currency,
		}
	}
	return dtos
}

func toOrderOutputDTOList(orders []*domain.Order) []*OrderOutputDTO {
	if orders == nil {
		return nil
	}

	dtos := make([]*OrderOutputDTO, len(orders))
	for i, o := range orders {
		dtos[i] = toOrderOutputDTO(o)
	}
	return dtos
}
