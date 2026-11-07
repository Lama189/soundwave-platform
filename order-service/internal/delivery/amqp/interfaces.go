package amqp

import (
	"context"

	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/google/uuid"
)

type OrderService interface {
	PayOrder(ctx context.Context, input order.PayOrderDTO) (*order.OrderOutputDTO, error)
	FailOrder(ctx context.Context, input order.FailOrderDTO) (*order.OrderOutputDTO, error)
	OnCatalogEntitlementsGranted(ctx context.Context, orderID uuid.UUID) error
	OnCatalogEntitlementsFailed(ctx context.Context, orderID uuid.UUID, reason string) error
	OnBillingSettled(ctx context.Context, orderID uuid.UUID) error
	OnBillingFailed(ctx context.Context, orderID uuid.UUID, reason string) error
	OnCatalogEntitlementsRevoked(ctx context.Context, orderID uuid.UUID) error
	OnPaymentRefunded(ctx context.Context, orderID uuid.UUID) error
}
