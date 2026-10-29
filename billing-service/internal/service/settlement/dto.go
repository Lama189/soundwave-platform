package settlement

import (
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

type ItemType string

const (
	ItemTypeTrack ItemType = "track"
	ItemTypeAlbum ItemType = "album"
)

func (t ItemType) IsValid() bool {
	switch t {
	case ItemTypeTrack, ItemTypeAlbum:
		return true
	default:
		return false
	}
}

type PaidOrderItemInput struct {
	ItemID   uuid.UUID
	ItemType ItemType
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
