package grpcclient

import (
	"time"

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
