package v1

import (
	"context"

	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
)

type OrderService interface {
	CreateOrder(ctx context.Context, input order.CreateOrderDTO) (*order.OrderOutputDTO, error)
	PayOrder(ctx context.Context, input order.PayOrderDTO) (*order.OrderOutputDTO, error)
	GetOrder(ctx context.Context, input order.GetOrderDTO) (*order.OrderOutputDTO, error)
	ListBuyerOrders(ctx context.Context, input order.ListBuyerOrdersDTO) (*order.ListOrdersOutputDTO, error)
	CancelOrder(ctx context.Context, input order.CancelOrderDTO) (*order.OrderOutputDTO, error)
}
