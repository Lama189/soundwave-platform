package v1

import (
	"context"
	"fmt"

	orderpb "github.com/Lama189/soundwave-platform/gen/go/order/v1"
	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGRPCServer struct {
	orderpb.UnimplementedOrderServiceServer
	service OrderService
}

func NewOrderGRPCServer(service OrderService) *OrderGRPCServer {
	return &OrderGRPCServer{
		service: service,
	}
}

func (s *OrderGRPCServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	if req.GetBuyerUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "buyer user id is required")
	}
	buyerID, err := uuid.Parse(req.GetBuyerUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer user id format")
	}

	items, err := toCreateOrderItemDTOs(req.GetItems())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order items: %v", err)
	}

	dto := order.CreateOrderDTO{
		BuyerUserID: buyerID,
		Items:       items,
	}

	order, err := s.service.CreateOrder(ctx, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orderpb.CreateOrderResponse{
		Order: toProtoOrder(order),
	}, nil
}

func (s *OrderGRPCServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id format")
	}

	var buyerID uuid.UUID
	if req.GetBuyerUserId() != "" {
		var err error
		buyerID, err = uuid.Parse(req.GetBuyerUserId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid buyer user id format")
		}
	}

	dto := order.GetOrderDTO{
		OrderID:     orderID,
		BuyerUserID: buyerID,
	}

	order, err := s.service.GetOrder(ctx, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orderpb.GetOrderResponse{
		Order: toProtoOrder(order),
	}, nil
}

func (s *OrderGRPCServer) PayOrder(ctx context.Context, req *orderpb.PayOrderRequest) (*orderpb.PayOrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id format")
	}

	paymentID, err := uuid.Parse(req.GetPaymentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment id format")
	}

	dto := order.PayOrderDTO{
		OrderID:   orderID,
		PaymentID: paymentID,
	}

	order, err := s.service.PayOrder(ctx, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orderpb.PayOrderResponse{
		Order: toProtoOrder(order),
	}, nil
}

func (s *OrderGRPCServer) CancelOrder(ctx context.Context, req *orderpb.CancelOrderRequest) (*orderpb.CancelOrderResponse, error) {
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id format")
	}

	var buyerID uuid.UUID
	if req.GetBuyerUserId() != "" {
		var err error
		buyerID, err = uuid.Parse(req.GetBuyerUserId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid buyer user id format")
		}
	}

	dto := order.CancelOrderDTO{
		OrderID:     orderID,
		BuyerUserID: buyerID,
	}

	order, err := s.service.CancelOrder(ctx, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orderpb.CancelOrderResponse{
		Order: toProtoOrder(order),
	}, nil
}

func (s *OrderGRPCServer) ListBuyerOrders(ctx context.Context, req *orderpb.ListBuyerOrdersRequest) (*orderpb.ListBuyerOrdersResponse, error) {
	if req.GetBuyerUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "buyer user id is required")
	}
	buyerID, err := uuid.Parse(req.GetBuyerUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer user id format")
	}

	dto := order.ListBuyerOrdersDTO{
		BuyerUserID: buyerID,
		Limit:       int(req.GetLimit()),
		Offset:      int(req.GetOffset()),
	}

	orders, err := s.service.ListBuyerOrders(ctx, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &orderpb.ListBuyerOrdersResponse{
		Orders: toProtoOrders(orders.Orders),
	}, nil
}

func toCreateOrderItemDTOs(pbItems []*orderpb.CreateOrderItem) ([]order.CreateOrderItemDTO, error) {
	if len(pbItems) == 0 {
		return nil, fmt.Errorf("items list cannot be empty")
	}

	items := make([]order.CreateOrderItemDTO, 0, len(pbItems))
	for i, item := range pbItems {
		if item == nil {
			return nil, fmt.Errorf("item at index %d is nil", i)
		}

		itemID, err := uuid.Parse(item.GetItemId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid order item id format")
		}

		artistID, err := uuid.Parse(item.GetArtistId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid artist id format")
		}

		items = append(items, order.CreateOrderItemDTO{
			ItemType: domain.ItemType(item.GetItemType()),
			ItemID:   itemID,
			ArtistID: artistID,
			Price:    item.GetPrice(),
			Currency: domain.Currency(item.GetCurrency()),
		})
	}

	return items, nil
}

func toProtoOrder(o *order.OrderOutputDTO) *orderpb.Order {
	if o == nil {
		return nil
	}

	var paymentID *string
	if o.PaymentID != nil {
		idStr := o.PaymentID.String()
		paymentID = &idStr
	}

	return &orderpb.Order{
		Id:            o.ID.String(),
		BuyerUserId:   o.BuyerUserID.String(),
		Status:        string(o.Status),
		TotalAmount:   o.TotalAmount,
		Currency:      string(o.Currency),
		PaymentId:     paymentID,
		FailureReason: o.FailureReason,
		Items:         toProtoOrderItems(o.Items),
		CreatedAt:     timestamppb.New(o.CreatedAt),
		UpdatedAt:     timestamppb.New(o.UpdatedAt),
	}
}

func toProtoOrderItem(it order.OrderItemOutputDTO) *orderpb.OrderItem {
	return &orderpb.OrderItem{
		Id:       it.ID.String(),
		OrderId:  it.OrderID.String(),
		ItemType: string(it.ItemType),
		ItemId:   it.ItemID.String(),
		ArtistId: it.ArtistID.String(),
		Price:    it.Price,
		Currency: string(it.Currency),
	}
}

func toProtoOrderItems(items []order.OrderItemOutputDTO) []*orderpb.OrderItem {
	if items == nil {
		return nil
	}

	pbItems := make([]*orderpb.OrderItem, len(items))
	for i, it := range items {
		pbItems[i] = toProtoOrderItem(it)
	}
	return pbItems
}

func toProtoOrders(orders []*order.OrderOutputDTO) []*orderpb.Order {
	if orders == nil {
		return nil
	}

	pbOrders := make([]*orderpb.Order, len(orders))
	for i, o := range orders {
		pbOrders[i] = toProtoOrder(o)
	}
	return pbOrders
}
