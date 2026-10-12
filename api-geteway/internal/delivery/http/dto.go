package http

import (
	"time"

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
