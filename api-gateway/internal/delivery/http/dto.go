package http

import (
	"time"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/order"
	"github.com/Lama189/soundwave-platform/pkg/money"
	"github.com/google/uuid"
)

type BecomeArtistRequest struct {
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

type TrackResponse struct {
	ID              uuid.UUID  `json:"id"`
	ArtistID        uuid.UUID  `json:"artist_id"`
	AlbumID         *uuid.UUID `json:"album_id,omitempty"`
	Title           string     `json:"title"`
	DurationSeconds int64      `json:"duration_seconds"`
	CoverKey        string     `json:"cover_key,omitempty"`
	FileKey         string     `json:"file_key,omitempty"`
	PreviewKey      string     `json:"preview_key,omitempty"`
	RawKey          string     `json:"raw_key,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UploadTrackRequest struct {
	Title    string     `json:"title"`
	FileName string     `json:"file_name"`
	AlbumID  *uuid.UUID `json:"album_id,omitempty"`
}

type UploadTrackResponse struct {
	Track     TrackResponse `json:"track"`
	UploadURL string        `json:"upload_url"`
}

type ConfirmUploadRequest struct {
	TrackID uuid.UUID `json:"track_id,omitempty"`
}

type CompleteProcessingRequest struct {
	TrackID         uuid.UUID `json:"track_id,omitempty"`
	DurationSeconds int64     `json:"duration_seconds"`
	FileKey         string    `json:"file_key"`
	PreviewKey      string    `json:"preview_key"`
	CoverKey        string    `json:"cover_key,omitempty"`
}

type FailProcessingRequest struct {
	TrackID uuid.UUID `json:"track_id,omitempty"`
}

type CreateOrderItemRequest struct {
	ItemType string    `json:"item_type"`
	ItemID   uuid.UUID `json:"item_id"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItemRequest `json:"items"`
}

func (r CreateOrderRequest) ToInput(buyerUserID uuid.UUID) order.CreateOrderInput {
	items := make([]order.CreateOrderItemInput, len(r.Items))
	for i, item := range r.Items {
		items[i] = order.CreateOrderItemInput{
			ItemType: item.ItemType,
			ItemID:   item.ItemID,
		}
	}

	return order.CreateOrderInput{
		BuyerUserID: buyerUserID,
		Items:       items,
	}
}

type OrderItemResponse struct {
	ID       uuid.UUID      `json:"id"`
	ItemType string         `json:"item_type"`
	ItemID   uuid.UUID      `json:"item_id"`
	ArtistID uuid.UUID      `json:"artist_id"`
	Price    int64          `json:"price"`
	Currency money.Currency `json:"currency"`
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

func ToOrderResponse(out *order.OrderOutput) OrderResponse {
	if out == nil {
		return OrderResponse{}
	}

	items := make([]OrderItemResponse, len(out.Items))
	for i, it := range out.Items {
		items[i] = OrderItemResponse{
			ID:       it.ID,
			ItemType: it.ItemType,
			ItemID:   it.ItemID,
			ArtistID: it.ArtistID,
			Price:    it.Price,
			Currency: it.Currency,
		}
	}

	return OrderResponse{
		ID:            out.ID,
		BuyerUserID:   out.BuyerUserID,
		Status:        out.Status,
		TotalAmount:   out.TotalAmount,
		Currency:      out.Currency,
		PaymentID:     out.PaymentID,
		FailureReason: out.FailureReason,
		Items:         items,
		CreatedAt:     out.CreatedAt,
		UpdatedAt:     out.UpdatedAt,
	}
}
