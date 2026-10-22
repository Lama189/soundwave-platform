package track

import (
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type UploadTrackDTO struct {
	AlbumID  *uuid.UUID
	Title    string
	FileName string
	Price    int64
	Currency domain.Currency
}

type CompleteProcessingDTO struct {
	CoverKey   string
	FileKey    string
	PreviewKey string
	Duration   time.Duration
}

type UpdateTrackDTO struct {
	Title    string
	AlbumID  *uuid.UUID
	CoverKey string
	Price    int64
	Currency domain.Currency
}

type TrackOutputDTO struct {
	ID         uuid.UUID
	ArtistID   uuid.UUID
	AlbumID    *uuid.UUID
	Title      string
	Duration   time.Duration
	CoverKey   string
	FileKey    string
	PreviewKey string
	RawKey     string
	Price      int64
	Currency   domain.Currency
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UploadTrackOutputDTO struct {
	Track     TrackOutputDTO
	UploadURL string
}

func toOutputDTO(t *domain.Track) *TrackOutputDTO {
	if t == nil {
		return nil
	}

	return &TrackOutputDTO{
		ID:         t.ID,
		ArtistID:   t.ArtistID,
		AlbumID:    t.AlbumID,
		Title:      t.Title,
		Duration:   t.Duration,
		CoverKey:   t.CoverKey,
		FileKey:    t.FileKey,
		PreviewKey: t.PreviewKey,
		RawKey:     t.RawKey,
		Price:      t.Price,
		Currency:   t.Currency,
		Status:     string(t.Status),
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func toOutputDTOList(tracks []*domain.Track) []*TrackOutputDTO {
	if tracks == nil {
		return []*TrackOutputDTO{}
	}

	list := make([]*TrackOutputDTO, 0, len(tracks))
	for _, t := range tracks {
		list = append(list, toOutputDTO(t))
	}

	return list
}
