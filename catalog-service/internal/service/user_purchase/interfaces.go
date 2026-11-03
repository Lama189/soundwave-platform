package user_purchase

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/uow"
	"github.com/google/uuid"
)

type UnitOfWorkManager interface {
	Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error
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

type TrackRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error)
	GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error)
}

type AlbumRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error)
}
