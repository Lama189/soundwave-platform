package settlement

import (
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type PaidOrderItemInput struct {
	ItemID   uuid.UUID
	ItemType itemtype.ItemType
	ArtistID uuid.UUID
	Price    int64
	Currency domain.Currency
}

type ProcessPaidOrderInput struct {
	OrderID     uuid.UUID
	BuyerUserID uuid.UUID
	TotalAmount int64
	Currency    domain.Currency
	PaymentID   uuid.UUID
	PaidAt      time.Time
	Items       []PaidOrderItemInput
}
