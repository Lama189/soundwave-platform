package settlement

import (
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type SettleOrderItemInput struct {
	ItemID   uuid.UUID
	ItemType itemtype.ItemType
	ArtistID uuid.UUID
	Price    int64
	Currency domain.Currency
}

type SettleOrderInput struct {
	CommandID   uuid.UUID
	SagaID      uuid.UUID
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
	TotalAmount int64
	Currency    domain.Currency
	PaymentID   uuid.UUID
	PaidAt      time.Time
	Items       []SettleOrderItemInput
}

func (in SettleOrderInput) Validate() error {
	if in.CommandID == uuid.Nil {
		return fmt.Errorf("%w: command id is required", domain.ErrInvalidInput)
	}
	if in.SagaID == uuid.Nil {
		return fmt.Errorf("%w: saga id is required", domain.ErrInvalidInput)
	}
	if in.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if len(in.Items) == 0 {
		return fmt.Errorf("%w: order items cannot be empty", domain.ErrInvalidInput)
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

type BillingReplyPayload struct {
	CommandID uuid.UUID `json:"command_id"`
	SagaID    uuid.UUID `json:"saga_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}
