package amqp

import (
	"context"

	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
)

type OrderService interface {
	PayOrder(ctx context.Context, input order.PayOrderDTO) (*order.OrderOutputDTO, error)
}
