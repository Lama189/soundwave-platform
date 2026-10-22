package uow

import (
	"context"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/google/uuid"
)

type UnitOfWork interface {
	Orders() OrderRepository
	Outbox() OutboxRepository
}

type Manager interface {
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error
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
