package amqp

import (
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/user_purchase"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type TrackProcessedEvent struct {
	TrackID         uuid.UUID `json:"track_id"`
	DurationSeconds int64     `json:"duration_seconds"`
	FileKey         string    `json:"file_key"`
	PreviewKey      string    `json:"preview_key"`
	CoverKey        *string   `json:"cover_key"`
}

type TrackProcessingFailedEvent struct {
	TrackID      uuid.UUID `json:"track_id"`
	ErrorMessage string    `json:"error_message"`
}

type OrderPaidItemEvent struct {
	ItemID   uuid.UUID         `json:"item_id"`
	ItemType itemtype.ItemType `json:"item_type"`
	ArtistID uuid.UUID         `json:"artist_id"`
	Price    int64             `json:"price"`
	Currency string            `json:"currency"`
}

type OrderPaidEvent struct {
	OrderID     uuid.UUID            `json:"order_id"`
	BuyerUserID uuid.UUID            `json:"buyer_user_id"`
	TotalAmount int64                `json:"total_amount"`
	Currency    string               `json:"currency"`
	PaymentID   uuid.UUID            `json:"payment_id"`
	PaidAt      time.Time            `json:"paid_at"`
	Items       []OrderPaidItemEvent `json:"items"`
}

func (e *OrderPaidEvent) toServiceInput() user_purchase.ProcessPaidOrderInput {
	items := make([]user_purchase.PaidItemInput, len(e.Items))
	for i, item := range e.Items {
		items[i] = user_purchase.PaidItemInput{
			ItemID:   item.ItemID,
			ItemType: item.ItemType,
		}
	}

	return user_purchase.ProcessPaidOrderInput{
		OrderID:     e.OrderID,
		BuyerUserID: e.BuyerUserID,
		Items:       items,
	}
}
