package track

import (
	"context"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type TrackRepository interface {
	Create(ctx context.Context, track *domain.Track) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error)
	GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Track, error)
	GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error)
	Update(ctx context.Context, track *domain.Track) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TrackStatus, fileKey, previewKey string, duration time.Duration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ArtistRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artist, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Artist, error)
}

type AlbumRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error)
}

type TrackLikeRepository interface {
	Add(ctx context.Context, like *domain.TrackLike) error
	Remove(ctx context.Context, userID, trackID uuid.UUID) error
	IsLiked(ctx context.Context, userID, trackID uuid.UUID) (bool, error)
	GetLikedTrackIDs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]uuid.UUID, error)
	CountByTrackID(ctx context.Context, trackID uuid.UUID) (int64, error)
}

type Storage interface {
	GenerateUploadURL(ctx context.Context, key string) (string, error)
	ObjectExists(ctx context.Context, key string) (bool, error)
}
