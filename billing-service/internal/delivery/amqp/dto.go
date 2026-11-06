package amqp

import (
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
)

type SettleOrderCommandItem struct {
	ItemID   uuid.UUID       `json:"item_id"`
	ItemType string          `json:"item_type"`
	ArtistID uuid.UUID       `json:"artist_id"`
	Price    int64           `json:"price"`
	Currency domain.Currency `json:"currency"`
}

type SettleOrderCommand struct {
	CommandID   uuid.UUID                `json:"command_id"`
	SagaID      uuid.UUID                `json:"saga_id"`
	OrderID     uuid.UUID                `json:"order_id"`
	BuyerUserID uuid.UUID                `json:"buyer_user_id"`
	TotalAmount int64                    `json:"total_amount"`
	Currency    domain.Currency          `json:"currency"`
	PaymentID   uuid.UUID                `json:"payment_id"`
	PaidAt      time.Time                `json:"paid_at"`
	Items       []SettleOrderCommandItem `json:"items"`
}

func (c *SettleOrderCommand) ToServiceInput() settlement.SettleOrderInput {
	items := make([]settlement.SettleOrderItemInput, len(c.Items))
	for i, item := range c.Items {
		items[i] = settlement.SettleOrderItemInput{
			ItemID:   item.ItemID,
			ItemType: itemtype.ItemType(item.ItemType),
			ArtistID: item.ArtistID,
			Price:    item.Price,
			Currency: item.Currency,
		}
	}

	return settlement.SettleOrderInput{
		CommandID:   c.CommandID,
		SagaID:      c.SagaID,
		OrderID:     c.OrderID,
		BuyerUserID: c.BuyerUserID,
		TotalAmount: c.TotalAmount,
		Currency:    c.Currency,
		PaymentID:   c.PaymentID,
		PaidAt:      c.PaidAt,
		Items:       items,
	}
}
