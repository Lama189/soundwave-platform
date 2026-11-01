package grpcclient

import (
	"context"
	"fmt"
	"time"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"google.golang.org/grpc"
)

type OrderClient struct {
	grpcClient orderpb.OrderServiceClient
	timeout    time.Duration
}

func NewOrderClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *OrderClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &OrderClient{
		grpcClient: orderpb.NewOrderServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *OrderClient) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbItems := make([]*orderpb.CreateOrderItem, len(req.Items))
	for i, item := range req.Items {
		pbItems[i] = &orderpb.CreateOrderItem{
			ItemType: item.ItemType,
			ItemId:   item.ItemID.String(),
		}
	}

	pbReq := &orderpb.CreateOrderRequest{
		BuyerUserId: req.BuyerUserID.String(),
		Items:       pbItems,
	}

	pbResp, err := c.grpcClient.CreateOrder(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	order := pbResp.GetOrder()
	if order == nil {
		return nil, fmt.Errorf("empty order in grpc response")
	}

	return toOrderResponse(order), nil
}
