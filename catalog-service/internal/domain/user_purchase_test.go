package domain_test

import (
	"errors"
	"testing"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

func TestNewTrackPurchase(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()
	trackID := uuid.New()

	purchase, err := domain.NewTrackPurchase(userID, orderID, trackID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if purchase.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if purchase.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, purchase.UserID)
	}
	if purchase.OrderID != orderID {
		t.Errorf("expected orderID %s, got %s", orderID, purchase.OrderID)
	}
	if !purchase.IsTrack() {
		t.Error("expected IsTrack() to be true")
	}
	if purchase.IsAlbum() {
		t.Error("expected IsAlbum() to be false")
	}
	if purchase.ItemID() != trackID {
		t.Errorf("expected ItemID %s, got %s", trackID, purchase.ItemID())
	}
	if purchase.AlbumID != nil {
		t.Errorf("expected nil AlbumID, got %v", purchase.AlbumID)
	}
}

func TestNewAlbumPurchase(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()
	albumID := uuid.New()

	purchase, err := domain.NewAlbumPurchase(userID, orderID, albumID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if purchase.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if !purchase.IsAlbum() {
		t.Error("expected IsAlbum() to be true")
	}
	if purchase.IsTrack() {
		t.Error("expected IsTrack() to be false")
	}
	if purchase.ItemID() != albumID {
		t.Errorf("expected ItemID %s, got %s", albumID, purchase.ItemID())
	}
	if purchase.TrackID != nil {
		t.Errorf("expected nil TrackID, got %v", purchase.TrackID)
	}
}

func TestUserPurchase_Validate_Errors(t *testing.T) {
	validUserID := uuid.New()
	validOrderID := uuid.New()
	validID := uuid.New()

	tests := []struct {
		name     string
		userID   uuid.UUID
		orderID  uuid.UUID
		itemType domain.ItemType
		trackID  *uuid.UUID
		albumID  *uuid.UUID
		wantErr  error
	}{
		{
			name:     "missing user id",
			userID:   uuid.Nil,
			orderID:  validOrderID,
			itemType: domain.ItemTypeTrack,
			trackID:  &validID,
			albumID:  nil,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "missing order id",
			userID:   validUserID,
			orderID:  uuid.Nil,
			itemType: domain.ItemTypeTrack,
			trackID:  &validID,
			albumID:  nil,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "invalid item type",
			userID:   validUserID,
			orderID:  validOrderID,
			itemType: "single",
			trackID:  &validID,
			albumID:  nil,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "track purchase with nil track id",
			userID:   validUserID,
			orderID:  validOrderID,
			itemType: domain.ItemTypeTrack,
			trackID:  nil,
			albumID:  nil,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "track purchase with album id present",
			userID:   validUserID,
			orderID:  validOrderID,
			itemType: domain.ItemTypeTrack,
			trackID:  &validID,
			albumID:  &validID,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "album purchase with nil album id",
			userID:   validUserID,
			orderID:  validOrderID,
			itemType: domain.ItemTypeAlbum,
			trackID:  nil,
			albumID:  nil,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "album purchase with track id present",
			userID:   validUserID,
			orderID:  validOrderID,
			itemType: domain.ItemTypeAlbum,
			trackID:  &validID,
			albumID:  &validID,
			wantErr:  domain.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewUserPurchase(tt.userID, tt.orderID, tt.itemType, tt.trackID, tt.albumID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
