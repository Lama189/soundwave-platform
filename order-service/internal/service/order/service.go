package order

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/uow"
	"github.com/google/uuid"
)

type OrderService struct {
	uow     UnitOfWorkManager
	orders  OrderRepository
	catalog CatalogClient
}

func NewOrderService(uow UnitOfWorkManager, orders OrderRepository, catalog CatalogClient) *OrderService {
	return &OrderService{
		uow:     uow,
		orders:  orders,
		catalog: catalog,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderDTO) (*OrderOutputDTO, error) {
	if input.BuyerUserID == uuid.Nil {
		return nil, fmt.Errorf("%w: buyer user id is required", domain.ErrInvalidInput)
	}
	if len(input.Items) == 0 {
		return nil, domain.ErrEmptyOrderItems
	}

	var trackIDs []uuid.UUID
	var albumIDs []uuid.UUID

	for i, it := range input.Items {
		if it.ItemID == uuid.Nil {
			return nil, fmt.Errorf("%w: item [%d] id is nil", domain.ErrInvalidInput, i)
		}
		switch it.ItemType {
		case domain.ItemTypeTrack:
			trackIDs = append(trackIDs, it.ItemID)
		case domain.ItemTypeAlbum:
			albumIDs = append(albumIDs, it.ItemID)
		default:
			return nil, fmt.Errorf("%w: unsupported item type %q", domain.ErrInvalidInput, it.ItemType)
		}
	}

	catalogItemsMap := make(map[uuid.UUID]CatalogItemInfo, len(input.Items))

	if len(trackIDs) > 0 {
		tracks, err := s.catalog.GetTracksByIDs(ctx, trackIDs)
		if err != nil {
			return nil, fmt.Errorf("fetch tracks from catalog: %w", err)
		}
		for _, t := range tracks {
			if t.Status != "ready" {
				return nil, fmt.Errorf("%w: track %s is not ready (status: %s)", domain.ErrConflict, t.ID, t.Status)
			}
			catalogItemsMap[t.ID] = t
		}
	}

	if len(albumIDs) > 0 {
		albums, err := s.catalog.GetAlbumsByIDs(ctx, albumIDs)
		if err != nil {
			return nil, fmt.Errorf("fetch albums from catalog: %w", err)
		}
		for _, a := range albums {
			catalogItemsMap[a.ID] = a
		}
	}

	domainItems := make([]domain.OrderItem, len(input.Items))
	for i, it := range input.Items {
		info, found := catalogItemsMap[it.ItemID]
		if !found {
			return nil, fmt.Errorf("%w: item %s of type %s not found in catalog", domain.ErrNotFound, it.ItemID, it.ItemType)
		}

		item, err := domain.NewOrderItem(
			uuid.Nil,
			it.ItemType,
			it.ItemID,
			info.ArtistID,
			info.Price,
			info.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf("invalid order item [%d]: %w", i, err)
		}

		domainItems[i] = *item
	}

	order, err := domain.NewOrder(input.BuyerUserID, domainItems)
	if err != nil {
		return nil, fmt.Errorf("create order aggregate: %w", err)
	}

	err = s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		if err := work.Orders().Create(ctx, order); err != nil {
			return fmt.Errorf("persist order %s: %w", order.ID, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create order transaction: %w", err)
	}

	return toOrderOutputDTO(order), nil
}

func (s *OrderService) PayOrder(ctx context.Context, input PayOrderDTO) (*OrderOutputDTO, error) {
	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if input.PaymentID == uuid.Nil {
		return nil, fmt.Errorf("%w: payment id is required", domain.ErrInvalidInput)
	}

	var updatedOrder *domain.Order
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, input.OrderID)
		if err != nil {
			return fmt.Errorf("get order %s for payment: %w", input.OrderID, err)
		}

		if order.Status == domain.OrderStatusPaid {
			if order.PaymentID != nil && *order.PaymentID == input.PaymentID {
				updatedOrder = order
				return nil
			}
			return domain.ErrOrderAlreadyPaid
		}

		if err := order.MarkPaid(input.PaymentID); err != nil {
			return err
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update order status: %w", err)
		}

		paidItems := make([]OrderPaidItemEventPayload, len(order.Items))
		for i, it := range order.Items {
			paidItems[i] = OrderPaidItemEventPayload{
				ItemID:   it.ItemID,
				ItemType: it.ItemType,
				ArtistID: it.ArtistID,
				Price:    it.Price,
				Currency: it.Currency,
			}
		}

		payload := OrderPaidEventPayload{
			OrderID:     order.ID,
			BuyerUserID: order.BuyerUserID,
			TotalAmount: order.TotalAmount,
			Currency:    order.Currency,
			PaymentID:   input.PaymentID,
			PaidAt:      time.Now().UTC(),
			Items:       paidItems,
		}

		outboxEvent, err := domain.NewEvent(
			domain.AggregateTypeOrder,
			order.ID,
			domain.EventTypeOrderPaid,
			payload,
		)
		if err != nil {
			return fmt.Errorf("build outbox event: %w", err)
		}

		if err := work.Outbox().Record(ctx, outboxEvent); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		updatedOrder = order
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("pay order transaction: %w", err)
	}

	return toOrderOutputDTO(updatedOrder), nil
}

func (s *OrderService) FailOrder(ctx context.Context, input FailOrderDTO) (*OrderOutputDTO, error) {
	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if input.PaymentID == uuid.Nil {
		return nil, fmt.Errorf("%w: payment id is required", domain.ErrInvalidInput)
	}

	var failedOrder *domain.Order
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, input.OrderID)
		if err != nil {
			return fmt.Errorf("get order %s for fail: %w", input.OrderID, err)
		}

		if order.Status == domain.OrderStatusFailed {
			failedOrder = order
			return nil
		}

		if err := order.MarkFailed(input.PaymentID, input.FailureReason); err != nil {
			return err
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("updated failed order: %w", err)
		}

		payload := OrderFailedEventPayload{
			OrderID:       order.ID,
			BuyerUserID:   order.BuyerUserID,
			PaymentID:     input.PaymentID,
			FailureReason: input.FailureReason,
			FailedAt:      time.Now().UTC(),
		}

		outboxEvent, err := domain.NewEvent(
			domain.AggregateTypeOrder,
			order.ID,
			domain.EventTypeOrderFailed,
			payload,
		)
		if err != nil {
			return fmt.Errorf("build outbox event: %w", err)
		}

		if err := work.Outbox().Record(ctx, outboxEvent); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		failedOrder = order
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fail order transaction: %w", err)
	}

	return toOrderOutputDTO(failedOrder), nil
}

func (s *OrderService) GetOrder(ctx context.Context, input GetOrderDTO) (*OrderOutputDTO, error) {
	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}

	order, err := s.orders.GetByID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("get order by id %s: %w", input.OrderID, err)
	}

	if input.BuyerUserID != uuid.Nil && input.BuyerUserID != order.BuyerUserID {
		return nil, domain.ErrNotOrderBuyer
	}

	return toOrderOutputDTO(order), nil
}

func (s *OrderService) ListBuyerOrders(ctx context.Context, input ListBuyerOrdersDTO) (*ListOrdersOutputDTO, error) {
	if input.BuyerUserID == uuid.Nil {
		return nil, fmt.Errorf("%w: buyer user id is required", domain.ErrInvalidInput)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	orders, err := s.orders.GetByBuyerID(ctx, input.BuyerUserID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders for buyer %s: %w", input.BuyerUserID, err)
	}

	return &ListOrdersOutputDTO{
		Orders: toOrderOutputDTOList(orders),
	}, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, input CancelOrderDTO) (*OrderOutputDTO, error) {
	if input.OrderID == uuid.Nil {
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}

	var cancelledOrder *domain.Order
	err := s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, input.OrderID)
		if err != nil {
			return fmt.Errorf("get order for cancellation: %w", err)
		}

		if input.BuyerUserID != uuid.Nil && order.BuyerUserID != input.BuyerUserID {
			return domain.ErrNotOrderBuyer
		}

		if order.Status == domain.OrderStatusCancelled {
			cancelledOrder = order
			return nil
		}

		if err := order.Cancel(); err != nil {
			return err
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update cancelled order: %w", err)
		}

		cancelledOrder = order
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cancel order transaction: %w", err)
	}

	return toOrderOutputDTO(cancelledOrder), nil
}
