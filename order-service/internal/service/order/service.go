package order

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/uow"
	"github.com/Lama189/soundwave-platform/pkg/itemtype"
	"github.com/google/uuid"
)

type Service struct {
	uow            UnitOfWorkManager
	orders         OrderRepository
	catalog        CatalogClient
	stepTimeout    time.Duration
	maxStepRetries int
}

func NewService(uow UnitOfWorkManager, orders OrderRepository, catalog CatalogClient, defaultStepTimeout time.Duration, maxStepRetries int) *Service {
	if defaultStepTimeout <= 0 {
		defaultStepTimeout = 30 * time.Second
	}
	if maxStepRetries <= 0 {
		maxStepRetries = 2
	}

	return &Service{
		uow:            uow,
		orders:         orders,
		catalog:        catalog,
		stepTimeout:    defaultStepTimeout,
		maxStepRetries: maxStepRetries,
	}
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderDTO) (*OrderOutputDTO, error) {
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
		case itemtype.Track:
			trackIDs = append(trackIDs, it.ItemID)
		case itemtype.Album:
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

func (s *Service) PayOrder(ctx context.Context, input PayOrderDTO) (*OrderOutputDTO, error) {
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

		if order.Status == domain.OrderStatusProcessing || order.Status == domain.OrderStatusPaid {
			if order.PaymentID != nil && *order.PaymentID == input.PaymentID {
				updatedOrder = order
				return nil
			}
			return domain.ErrOrderAlreadyPaid
		}

		if err := order.StartFulfillment(input.PaymentID, s.stepTimeout); err != nil {
			return err
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update order status: %w", err)
		}

		if err := s.recordGrantEntitlementsCmd(ctx, work, order); err != nil {
			return err
		}

		updatedOrder = order
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("pay order transaction: %w", err)
	}

	return toOrderOutputDTO(updatedOrder), nil
}

func (s *Service) OnCatalogEntitlementsGranted(ctx context.Context, orderID uuid.UUID) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Step != domain.StepGrantEntitlements {
			return nil
		}

		if err := order.OnEntitlementsGranted(s.stepTimeout); err != nil {
			return fmt.Errorf("fsm on entitlements granted: %w", err)
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("updated order: %w", err)
		}

		return s.recordSettleBillingCmd(ctx, work, order)
	})
}

func (s *Service) OnBillingSettled(ctx context.Context, orderID uuid.UUID) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Status == domain.SagaStatusCompleted {
			return nil
		}

		if err := order.CompleteFulfillment(); err != nil {
			return fmt.Errorf("complete fulfillment: %w", err)
		}

		return work.Orders().Update(ctx, order)
	})
}

func (s *Service) OnBillingFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Status == domain.SagaStatusCompensating || order.Saga.Status == domain.SagaStatusFailed {
			return nil
		}

		if err := order.OnSettlementFailed(reason, s.stepTimeout); err != nil {
			return fmt.Errorf("fsm on settlement failed: %w", err)
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("updated order: %w", err)
		}

		return s.recordRevokeEntitlementsCmd(ctx, work, order.ID, reason)
	})
}

func (s *Service) OnCatalogEntitlementsRevoked(ctx context.Context, orderID uuid.UUID) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Step == domain.StepRefundPayment || order.Saga.Status == domain.SagaStatusFailed {
			return nil
		}

		if err := order.OnEntitlementsRevoked(s.stepTimeout); err != nil {
			return fmt.Errorf("fsm on entitlements revoked: %w", err)
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		reason := "order fulfillment failed"
		if order.Saga.FailureReason != nil {
			reason = *order.Saga.FailureReason
		}

		return s.recordRefundPaymentCmd(ctx, work, order.ID, reason)
	})
}

func (s *Service) OnCatalogEntitlementsFailed(ctx context.Context, orderID uuid.UUID, reason string) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Step == domain.StepRefundPayment || order.Saga.Status == domain.SagaStatusFailed {
			return nil
		}

		if err := order.OnEntitlementsGrantFailed(reason, s.stepTimeout); err != nil {
			return fmt.Errorf("fsm on entitlements grant failed: %w", err)
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		return s.recordRefundPaymentCmd(ctx, work, order.ID, reason)
	})
}

func (s *Service) OnPaymentRefunded(ctx context.Context, orderID uuid.UUID) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s: %w", orderID, err)
		}

		if order.Saga.Status == domain.SagaStatusFailed {
			return nil
		}

		if err := order.CompleteCompensation(); err != nil {
			return fmt.Errorf("complete compensation: %w", err)
		}

		return work.Orders().Update(ctx, order)
	})
}

func (s *Service) HandleTimeout(ctx context.Context, orderID uuid.UUID) error {
	return s.uow.Do(ctx, func(work uow.UnitOfWork) error {
		order, err := work.Orders().GetByIDForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("get order %s for timeout: %w", orderID, err)
		}

		if !order.IsTimedOut(time.Now().UTC()) {
			return nil
		}

		step, isRetry, err := order.HandleTimeout(time.Now().UTC(), s.stepTimeout, s.maxStepRetries)
		if err != nil {
			return fmt.Errorf("handle timeout: %w", err)
		}

		if err := work.Orders().Update(ctx, order); err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		if isRetry {
			switch step {
			case domain.StepGrantEntitlements:
				return s.recordGrantEntitlementsCmd(ctx, work, order)
			case domain.StepSettleBilling:
				return s.recordSettleBillingCmd(ctx, work, order)
			case domain.StepRevokeEntitlements:
				return s.recordRevokeEntitlementsCmd(ctx, work, order.ID, "step timeout retry")
			case domain.StepRefundPayment:
				return s.recordRefundPaymentCmd(ctx, work, order.ID, "step timeout retry")
			}
			return nil
		}

		switch step {
		case domain.StepRevokeEntitlements:
			return s.recordRevokeEntitlementsCmd(ctx, work, order.ID, "step timeout after max retries")
		case domain.StepRefundPayment:
			return s.recordRefundPaymentCmd(ctx, work, order.ID, "step timeout after max retries")
		}

		return nil
	})
}

func (s *Service) FailOrder(ctx context.Context, input FailOrderDTO) (*OrderOutputDTO, error) {
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

func (s *Service) GetOrder(ctx context.Context, input GetOrderDTO) (*OrderOutputDTO, error) {
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

func (s *Service) ListBuyerOrders(ctx context.Context, input ListBuyerOrdersDTO) (*ListOrdersOutputDTO, error) {
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

func (s *Service) CancelOrder(ctx context.Context, input CancelOrderDTO) (*OrderOutputDTO, error) {
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

func (s *Service) recordRevokeEntitlementsCmd(ctx context.Context, work uow.UnitOfWork, orderID uuid.UUID, reason string) error {
	cmd := RevokeEntitlementsCommand{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Reason:    reason,
	}

	event, err := domain.NewEvent(
		domain.AggregateTypeOrder,
		orderID,
		domain.EventTypeCmdCatalogRevokeEntitlements,
		cmd,
	)
	if err != nil {
		return fmt.Errorf("build revoke command: %w", err)
	}

	return work.Outbox().Record(ctx, event)
}

func (s *Service) recordRefundPaymentCmd(ctx context.Context, work uow.UnitOfWork, orderID uuid.UUID, reason string) error {
	cmd := RefundPaymentCommand{
		CommandID: uuid.New(),
		SagaID:    orderID,
		OrderID:   orderID,
		Reason:    reason,
	}

	event, err := domain.NewEvent(
		domain.AggregateTypeOrder,
		orderID,
		domain.EventTypeCmdPaymentRefund,
		cmd,
	)
	if err != nil {
		return fmt.Errorf("build refund command: %w", err)
	}

	return work.Outbox().Record(ctx, event)
}

func (s *Service) recordGrantEntitlementsCmd(ctx context.Context, work uow.UnitOfWork, order *domain.Order) error {
	items := make([]GrantEntitlementsCommandItem, len(order.Items))
	for i, it := range order.Items {
		items[i] = GrantEntitlementsCommandItem{
			ItemID:   it.ItemID,
			ItemType: it.ItemType,
		}
	}

	cmd := GrantEntitlementsCommand{
		CommandID:   uuid.New(),
		SagaID:      order.ID,
		OrderID:     order.ID,
		BuyerUserID: order.BuyerUserID,
		Items:       items,
	}

	outboxEvent, err := domain.NewEvent(
		domain.AggregateTypeOrder,
		order.ID,
		domain.EventTypeCmdCatalogGrantEntitlements,
		cmd,
	)
	if err != nil {
		return fmt.Errorf("build catalog command event: %w", err)
	}

	return work.Outbox().Record(ctx, outboxEvent)
}

func (s *Service) recordSettleBillingCmd(ctx context.Context, work uow.UnitOfWork, order *domain.Order) error {
	items := make([]SettleOrderCommandItem, len(order.Items))
	for i, it := range order.Items {
		items[i] = SettleOrderCommandItem{
			ItemID:   it.ItemID,
			ItemType: string(it.ItemType),
			ArtistID: it.ArtistID,
			Price:    it.Price,
			Currency: it.Currency,
		}
	}

	cmd := SettleOrderCommand{
		CommandID:   uuid.New(),
		SagaID:      order.ID,
		OrderID:     order.ID,
		BuyerUserID: order.BuyerUserID,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
		PaymentID:   *order.PaymentID,
		PaidAt:      order.UpdatedAt,
		Items:       items,
	}

	event, err := domain.NewEvent(
		domain.AggregateTypeOrder,
		order.ID,
		domain.EventTypeCmdBillingSettleOrder,
		cmd,
	)
	if err != nil {
		return fmt.Errorf("build billing command event: %w", err)
	}

	return work.Outbox().Record(ctx, event)
}
