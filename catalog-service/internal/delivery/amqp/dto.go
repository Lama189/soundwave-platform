package amqp

import (
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

type GrantEntitlementsCommandItem struct {
	ItemID   uuid.UUID         `json:"item_id"`
	ItemType itemtype.ItemType `json:"item_type"`
}

type GrantEntitlementsCommand struct {
	CommandID   uuid.UUID                      `json:"command_id"`
	SagaID      uuid.UUID                      `json:"saga_id"`
	OrderID     uuid.UUID                      `json:"order_id"`
	BuyerUserID uuid.UUID                      `json:"buyer_user_id"`
	Items       []GrantEntitlementsCommandItem `json:"items"`
}

func (c *GrantEntitlementsCommand) ToServiceInput() user_purchase.GrantEntitlementsInput {
	items := make([]user_purchase.PaidItemInput, len(c.Items))
	for i, item := range c.Items {
		items[i] = user_purchase.PaidItemInput{
			ItemID:   item.ItemID,
			ItemType: item.ItemType,
		}
	}

	return user_purchase.GrantEntitlementsInput{
		CommandID:   c.CommandID,
		SagaID:      c.SagaID,
		OrderID:     c.OrderID,
		BuyerUserID: c.BuyerUserID,
		Items:       items,
	}
}

type RevokeEntitlementsCommand struct {
	CommandID uuid.UUID `json:"command_id"`
	SagaID    uuid.UUID `json:"saga_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Reason    string    `json:"reason"`
}

func (c *RevokeEntitlementsCommand) ToServiceInput() user_purchase.RevokePurchasesInput {
	return user_purchase.RevokePurchasesInput{
		CommandID: c.CommandID,
		SagaID:    c.SagaID,
		OrderID:   c.OrderID,
		Reason:    c.Reason,
	}
}
