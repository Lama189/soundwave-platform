package domain

import (
	"fmt"

	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type ItemType = itemtype.ItemType

const (
	ItemTypeTrack = itemtype.Track
	ItemTypeAlbum = itemtype.Album
)

type OrderItem struct {
	ID       uuid.UUID
	OrderID  uuid.UUID
	ItemType itemtype.ItemType
	ItemID   uuid.UUID
	ArtistID uuid.UUID
	Price    int64
	Currency Currency
}

func NewOrderItem(orderID uuid.UUID, itemType itemtype.ItemType, itemID, artistID uuid.UUID, price int64, currency Currency) (*OrderItem, error) {
	oi := &OrderItem{
		ID:       uuid.New(),
		OrderID:  orderID,
		ItemType: itemType,
		ItemID:   itemID,
		ArtistID: artistID,
		Price:    price,
		Currency: currency,
	}

	if err := oi.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create order item: %w", err)
	}

	return oi, nil
}

func RestoreOrderItem(id, orderID uuid.UUID, itemType itemtype.ItemType, itemID, artistID uuid.UUID, price int64, currency Currency) OrderItem {
	return OrderItem{
		ID:       id,
		OrderID:  orderID,
		ItemType: itemType,
		ItemID:   itemID,
		ArtistID: artistID,
		Price:    price,
		Currency: currency,
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

	if oi.Price <= 0 {
		return fmt.Errorf("%w: price must be positive, got %d", ErrInvalidInput, oi.Price)
	}

	if !oi.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, oi.Currency)
	}

	return nil
}
