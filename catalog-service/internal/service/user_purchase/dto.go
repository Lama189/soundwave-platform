package user_purchase

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
)

type PaidItemInput struct {
	ItemID   uuid.UUID
	ItemType itemtype.ItemType
}

type GrantEntitlementsInput struct {
	CommandID   uuid.UUID
	SagaID      uuid.UUID
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
	Items       []PaidItemInput
}

func (in GrantEntitlementsInput) Validate() error {
	if in.CommandID == uuid.Nil {
		return fmt.Errorf("%w: command id is required", domain.ErrInvalidInput)
	}
	if in.SagaID == uuid.Nil {
		return fmt.Errorf("%w: saga id is required", domain.ErrInvalidInput)
	}
	if in.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if in.BuyerUserID == uuid.Nil {
		return fmt.Errorf("%w: buyer user id is required",
			domain.ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return fmt.Errorf("%w: order items cannot be empty",
			domain.ErrInvalidInput)
	}

	for i, item := range in.Items {
		if item.ItemID == uuid.Nil {
			return fmt.Errorf("%w: item [%d] id cannot be nil",
				domain.ErrInvalidInput, i)
		}
		if !item.ItemType.IsValid() {
			return fmt.Errorf("%w: item [%d] type %q is unsupported",
				domain.ErrInvalidInput, i, item.ItemType)
		}
	}
	return nil
}

type RevokePurchasesInput struct {
	CommandID uuid.UUID
	SagaID    uuid.UUID
	OrderID   uuid.UUID
	Reason    string
}

func (in RevokePurchasesInput) Validate() error {
	if in.CommandID == uuid.Nil {
		return fmt.Errorf("%w: command id is required", domain.ErrInvalidInput)
	}
	if in.SagaID == uuid.Nil {
		return fmt.Errorf("%w: saga id is required", domain.ErrInvalidInput)
	}
	if in.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	return nil
}

type CatalogReplyPayload struct {
	CommandID uuid.UUID `json:"command_id"`
	SagaID    uuid.UUID `json:"saga_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type ListUserPurchasesInput struct {
	UserID   uuid.UUID
	ItemType *itemtype.ItemType
	Limit    int
	Offset   int
}

type UserPurchaseOutput struct {
	ID        uuid.UUID             `json:"id"`
	UserID    uuid.UUID             `json:"user_id"`
	OrderID   uuid.UUID             `json:"order_id"`
	ItemType  itemtype.ItemType     `json:"item_type"`
	TrackID   *uuid.UUID            `json:"track_id,omitempty"`
	AlbumID   *uuid.UUID            `json:"album_id,omitempty"`
	Status    domain.PurchaseStatus `json:"status"`
	RevokedAt *time.Time            `json:"revoked_at,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
}

type ListUserPurchasesOutput struct {
	Purchases []*UserPurchaseOutput `json:"purchases"`
}
