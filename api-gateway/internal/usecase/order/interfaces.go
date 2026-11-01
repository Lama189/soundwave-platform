package order

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type OrderClient interface {
	CreateOrder(ctx context.Context, req grpcclient.CreateOrderRequest) (*grpcclient.OrderResponse, error)
}
