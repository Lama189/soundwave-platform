package grpcclient

import (
	"time"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/Lama189/soundwave-platform/pkg/money"
	"github.com/google/uuid"
)

type ArtistResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateArtistRequest struct {
	UserID uuid.UUID
	Name   string
	Bio    string
}

type UpdateArtistRequest struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string
	Bio    string
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TrackResponse struct {
	ID              uuid.UUID  `json:"id"`
	ArtistID        uuid.UUID  `json:"artist_id"`
	AlbumID         *uuid.UUID `json:"album_id,omitempty"`
	Title           string     `json:"title"`
	DurationSeconds int64      `json:"duration_seconds"`
	CoverKey        string     `json:"cover_key"`
	FileKey         string     `json:"file_key"`
	PreviewKey      string     `json:"preview_key"`
	RawKey          string     `json:"raw_key"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UploadTrackRequest struct {
	UserID   uuid.UUID
	AlbumID  *uuid.UUID
	Title    string
	FileName string
}

type UploadTrackResponse struct {
	Track     TrackResponse `json:"track"`
	UploadURL string        `json:"upload_url"`
}

type ConfirmUploadRequest struct {
	TrackID uuid.UUID
	UserID  uuid.UUID
}

type CompleteProcessingRequest struct {
	TrackID         uuid.UUID
	DurationSeconds int64
	FileKey         string
	PreviewKey      string
	CoverKey        string
}

type FailProcessingRequest struct {
	TrackID uuid.UUID
}

type FailProcessingResponse struct{}

type CreateOrderItemRequest struct {
	ItemType itemtype.ItemType
	ItemID   uuid.UUID
}

type CreateOrderRequest struct {
	BuyerUserID uuid.UUID
	Items       []CreateOrderItemRequest
}

type OrderItemResponse struct {
	ID       uuid.UUID         `json:"id"`
	ItemType itemtype.ItemType `json:"item_type"`
	ItemID   uuid.UUID         `json:"item_id"`
	ArtistID uuid.UUID         `json:"artist_id"`
	Price    int64             `json:"price"`
	Currency money.Currency    `json:"currency"`
}

type OrderResponse struct {
	ID            uuid.UUID           `json:"id"`
	BuyerUserID   uuid.UUID           `json:"buyer_user_id"`
	Status        string              `json:"status"`
	TotalAmount   int64               `json:"total_amount"`
	Currency      money.Currency      `json:"currency"`
	PaymentID     *uuid.UUID          `json:"payment_id,omitempty"`
	FailureReason *string             `json:"failure_reason,omitempty"`
	Items         []OrderItemResponse `json:"items"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

func toOrderResponse(pb *orderpb.Order) *OrderResponse {
	orderID, _ := uuid.Parse(pb.GetId())
	buyerID, _ := uuid.Parse(pb.GetBuyerUserId())

	var paymentID *uuid.UUID
	if pb.PaymentId != nil && *pb.PaymentId != "" {
		if parsed, err := uuid.Parse(*pb.PaymentId); err == nil {
			paymentID = &parsed
		}
	}

	var createdAt time.Time
	if pb.GetCreatedAt() != nil {
		createdAt = pb.GetCreatedAt().AsTime()
	}

	var updatedAt time.Time
	if pb.GetUpdatedAt() != nil {
		updatedAt = pb.GetUpdatedAt().AsTime()
	}

	items := make([]OrderItemResponse, len(pb.GetItems()))
	for i, it := range pb.GetItems() {
		itID, _ := uuid.Parse(it.GetId())
		itemID, _ := uuid.Parse(it.GetItemId())
		artistID, _ := uuid.Parse(it.GetArtistId())

		items[i] = OrderItemResponse{
			ID:       itID,
			ItemType: itemtype.ItemType(it.GetItemType()),
			ItemID:   itemID,
			ArtistID: artistID,
			Price:    it.GetPrice(),
			Currency: money.Currency(it.GetCurrency()),
		}
	}

	return &OrderResponse{
		ID:            orderID,
		BuyerUserID:   buyerID,
		Status:        pb.GetStatus(),
		TotalAmount:   pb.GetTotalAmount(),
		Currency:      money.Currency(pb.GetCurrency()),
		PaymentID:     paymentID,
		FailureReason: pb.FailureReason,
		Items:         items,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
