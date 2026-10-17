package domain

import (
	"fmt"

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

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	ItemType    ItemType
	ItemID      uuid.UUID
	ArtistID    uuid.UUID
	Price       Money
	PlatformFee Money
	ArtistShare Money
}

func NewOrderItem(orderID uuid.UUID, itemType ItemType, itemID, artistID uuid.UUID, price, platformFee, artistShare Money) (*OrderItem, error) {
	oi := &OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ItemType:    itemType,
		ItemID:      itemID,
		ArtistID:    artistID,
		Price:       price,
		PlatformFee: platformFee,
		ArtistShare: artistShare,
	}

	if err := oi.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create order item: %w", err)
	}

	return oi, nil
}

func RestoreOrderItem(id, orderID uuid.UUID, itemType ItemType, itemID, artistID uuid.UUID, price, platformFee, artistShare Money) OrderItem {
	return OrderItem{
		ID:          id,
		OrderID:     orderID,
		ItemType:    itemType,
		ItemID:      itemID,
		ArtistID:    artistID,
		Price:       price,
		PlatformFee: platformFee,
		ArtistShare: artistShare,
	}
}

func (oi *OrderItem) Validate() error {
	if oi.ID == uuid.Nil {
		return fmt.Errorf("%w: order item id is required", ErrInvalidInput)
	}

	if !oi.ItemType.IsValid() {
		return fmt.Errorf("%w: unsupported item type %q", ErrInvalidInput, oi.ItemType)
	}

	if oi.ItemID == uuid.Nil {
		return fmt.Errorf("%w: item id is required", ErrInvalidInput)
	}

	if oi.ArtistID == uuid.Nil {
		return fmt.Errorf("%w: artist id is required", ErrInvalidInput)
	}

	if err := oi.Price.Validate(); err != nil {
		return fmt.Errorf("invalid price: %w", err)
	}
	if err := oi.PlatformFee.Validate(); err != nil {
		return fmt.Errorf("invalid platform fee: %w", err)
	}
	if err := oi.ArtistShare.Validate(); err != nil {
		return fmt.Errorf("invalid artist share: %w", err)
	}

	totalParts, err := oi.PlatformFee.Add(oi.ArtistShare)
	if err != nil || oi.Price.Currency != totalParts.Currency {
		return fmt.Errorf("%w: price (%s), platform fee (%s), artist share (%s)",
			ErrCurrencyMismatch, oi.Price.Currency, oi.PlatformFee.Currency, oi.ArtistShare.Currency)
	}

	if oi.Price.Amount != totalParts.Amount {
		return fmt.Errorf("%w: price (%d) does not equal platform fee (%d) + artist share (%d)",
			ErrInvalidInput, oi.Price.Amount, oi.PlatformFee.Amount, oi.ArtistShare.Amount)
	}

	return nil
}
