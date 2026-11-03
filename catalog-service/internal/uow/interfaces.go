package uow

import (
	"context"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type UnitOfWork interface {
	Tracks() TrackRepository
	Outbox() OutboxRepository
	Artists() ArtistRepository
	Albums() AlbumRepository
	Purchases() UserPurchaseRepository
}

type Manager interface {
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error
}

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

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}

type ArtistRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artist, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Artist, error)
}

type AlbumRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error)
}

type UserPurchaseRepository interface {
	Create(ctx context.Context, purchase *domain.UserPurchase) error
	CreateBatch(ctx context.Context, purchases []*domain.UserPurchase) error
	HasPurchasedTrack(ctx context.Context, userID, trackID uuid.UUID) (bool, error)
	HasPurchasedAlbum(ctx context.Context, userID, albumID uuid.UUID) (bool, error)
	GetPurchasedTrackIDsIn(ctx context.Context, userID uuid.UUID, trackIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, itemType *domain.ItemType, limit, offset int) ([]*domain.UserPurchase, error)
	ExistsByOrderID(ctx context.Context, orderID uuid.UUID) (bool, error)
}
