package user_purchase

import (
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type PaidItemInput struct {
	ItemID   uuid.UUID
	ItemType itemtype.ItemType
}

type ProcessPaidOrderInput struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
	Items       []PaidItemInput
}

func (in ProcessPaidOrderInput) Validate() error {
	if in.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if in.BuyerUserID == uuid.Nil {
		return fmt.Errorf("%w: buyer user id is required", domain.ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return fmt.Errorf("%w: paid order items cannot be empty", domain.ErrInvalidInput)
	}

	for i, item := range in.Items {
		if item.ItemID == uuid.Nil {
			return fmt.Errorf("%w: item [%d] id cannot be nil", domain.ErrInvalidInput, i)
		}
		if !item.ItemType.IsValid() {
			return fmt.Errorf("%w: item [%d] type %q is unsupported", domain.ErrInvalidInput, i, item.ItemType)
		}
	}
	return nil
}

type ListUserPurchasesInput struct {
	UserID   uuid.UUID
	ItemType *itemtype.ItemType
	Limit    int
	Offset   int
}

type UserPurchaseOutput struct {
	ID        uuid.UUID         `json:"id"`
	UserID    uuid.UUID         `json:"user_id"`
	OrderID   uuid.UUID         `json:"order_id"`
	ItemType  itemtype.ItemType `json:"item_type"`
	TrackID   *uuid.UUID        `json:"track_id,omitempty"`
	AlbumID   *uuid.UUID        `json:"album_id,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type ListUserPurchasesOutput struct {
	Purchases []*UserPurchaseOutput `json:"purchases"`
}
