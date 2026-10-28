package grpcclient

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
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

func (c *OrderClient) GetOrder(ctx context.Context, req GetOrderRequest) (*OrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var buyerUserID *string
	if req.BuyerUserID != nil {
		id := req.BuyerUserID.String()
		buyerUserID = &id
	}

	pbReq := &orderpb.GetOrderRequest{
		OrderId:     req.OrderID.String(),
		BuyerUserId: buyerUserID,
	}

	pbResp, err := c.grpcClient.GetOrder(ctx, pbReq)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	order := pbResp.GetOrder()
	if order == nil {
		return nil, fmt.Errorf("%w: empty order in grpc response", domain.ErrOrderCommunication)
	}

	return toOrderResponse(order), nil
}

func (c *OrderClient) PayOrder(ctx context.Context, req PayOrderRequest) (*OrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if req.OrderID == uuid.Nil || req.PaymentID == uuid.Nil {
		return nil, fmt.Errorf("%w: order_id and payment_id are required", domain.ErrInvalidInput)
	}

	pbReq := &orderpb.PayOrderRequest{
		OrderId:   req.OrderID.String(),
		PaymentId: req.PaymentID.String(),
	}

	pbResp, err := c.grpcClient.PayOrder(ctx, pbReq)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	order := pbResp.GetOrder()
	if order == nil {
		return nil, fmt.Errorf("%w: empty order in grpc response", domain.ErrOrderCommunication)
	}

	return toOrderResponse(order), nil
}
