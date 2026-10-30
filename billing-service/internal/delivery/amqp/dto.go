package amqp

import (
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
)

type OrderPaidItemEvent struct {
	ItemID   uuid.UUID       `json:"item_id"`
	ItemType string          `json:"item_type"`
	ArtistID uuid.UUID       `json:"artist_id"`
	Price    int64           `json:"price"`
	Currency domain.Currency `json:"currency"`
}

type OrderPaidEvent struct {
	OrderID     uuid.UUID            `json:"order_id"`
	BuyerUserID uuid.UUID            `json:"buyer_user_id"`
	TotalAmount int64                `json:"total_amount"`
	Currency    domain.Currency      `json:"currency"`
	PaymentID   uuid.UUID            `json:"payment_id"`
	PaidAt      time.Time            `json:"paid_at"`
	Items       []OrderPaidItemEvent `json:"items"`
}

func (e *OrderPaidEvent) toSettlementInput() settlement.ProcessPaidOrderInput {
	items := make([]settlement.PaidOrderItemInput, len(e.Items))
	for i, item := range e.Items {
		items[i] = settlement.PaidOrderItemInput{
			ItemID:   item.ItemID,
			ItemType: settlement.ItemType(item.ItemType),
			ArtistID: item.ArtistID,
			Price:    item.Price,
			Currency: item.Currency,
		}
	}

	return settlement.ProcessPaidOrderInput{
		OrderID:     e.OrderID,
		BuyerUserID: e.BuyerUserID,
		TotalAmount: e.TotalAmount,
		Currency:    e.Currency,
		PaymentID:   e.PaymentID,
		PaidAt:      e.PaidAt,
		Items:       items,
	}
}
