package order

import (
	"context"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/uow"
	"github.com/google/uuid"
)

type UnitOfWorkManager interface {
	Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error
}

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}

type CatalogItemInfo struct {
	ID       uuid.UUID
	ArtistID uuid.UUID
	Price    int64
	Currency domain.Currency
	Status   string
}

type CatalogClient interface {
	GetTracksByIDs(ctx context.Context, ids []uuid.UUID) ([]CatalogItemInfo, error)
	GetAlbumsByIDs(ctx context.Context, ids []uuid.UUID) ([]CatalogItemInfo, error)
}
