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
	GetByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}

type Service interface {
	CreateOrder(ctx context.Context, input CreateOrderDTO) (*OrderOutputDTO, error)
	GetOrder(ctx context.Context, input GetOrderDTO) (*OrderOutputDTO, error)
	ListBuyerOrders(ctx context.Context, input ListBuyerOrdersDTO) (*ListOrdersOutputDTO, error)
	PayOrder(ctx context.Context, input PayOrderDTO) (*OrderOutputDTO, error)
	CancelOrder(ctx context.Context, input CancelOrderDTO) (*OrderOutputDTO, error)
}
