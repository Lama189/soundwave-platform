package domain

import (
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type ItemType = itemtype.ItemType

const (
	ItemTypeTrack = itemtype.Track
	ItemTypeAlbum = itemtype.Album
)

type UserPurchase struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	OrderID   uuid.UUID
	ItemType  ItemType
	TrackID   *uuid.UUID
	AlbumID   *uuid.UUID
	CreatedAt time.Time
}

func NewUserPurchase(userID, orderID uuid.UUID, itemType ItemType, trackID, albumID *uuid.UUID) (*UserPurchase, error) {
	now := time.Now().UTC()

	p := &UserPurchase{
		ID:        uuid.New(),
		UserID:    userID,
		OrderID:   orderID,
		ItemType:  itemType,
		TrackID:   trackID,
		AlbumID:   albumID,
		CreatedAt: now,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create user purchase: %w", err)
	}

	return p, nil
}

func NewTrackPurchase(userID, orderID, trackID uuid.UUID) (*UserPurchase, error) {
	return NewUserPurchase(userID, orderID, ItemTypeTrack, &trackID, nil)
}

func NewAlbumPurchase(userID, orderID, albumID uuid.UUID) (*UserPurchase, error) {
	return NewUserPurchase(userID, orderID, ItemTypeAlbum, nil, &albumID)
}

func RestoreUserPurchase(id, userID, orderID uuid.UUID, itemType ItemType, trackID, albumID *uuid.UUID, createdAt time.Time) *UserPurchase {
	return &UserPurchase{
		ID:        id,
		UserID:    userID,
		OrderID:   orderID,
		ItemType:  itemType,
		TrackID:   trackID,
		AlbumID:   albumID,
		CreatedAt: createdAt,
	}
}

func (p *UserPurchase) IsTrack() bool {
	return p.ItemType == ItemTypeTrack
}

func (p *UserPurchase) IsAlbum() bool {
	return p.ItemType == ItemTypeAlbum
}

func (p *UserPurchase) ItemID() uuid.UUID {
	if p.TrackID != nil {
		return *p.TrackID
	}
	if p.AlbumID != nil {
		return *p.AlbumID
	}
	return uuid.Nil
}

func (p *UserPurchase) Validate() error {
	if p.UserID == uuid.Nil {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if p.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}

	if !p.ItemType.IsValid() {
		return fmt.Errorf("%w: invalid item type: %s", ErrInvalidInput, p.ItemType)
	}

	switch p.ItemType {
	case ItemTypeTrack:
		if p.TrackID == nil || *p.TrackID == uuid.Nil {
			return fmt.Errorf("%w: track id is required", ErrInvalidInput)
		}
		if p.AlbumID != nil {
			return fmt.Errorf("%w: album id must be nil for track purchase", ErrInvalidInput)
		}

	case ItemTypeAlbum:
		if p.AlbumID == nil || *p.AlbumID == uuid.Nil {
			return fmt.Errorf("%w: album id is required", ErrInvalidInput)
		}
		if p.TrackID != nil {
			return fmt.Errorf("%w: track id must be nil for album purchase", ErrInvalidInput)
		}
	}

	return nil
}
