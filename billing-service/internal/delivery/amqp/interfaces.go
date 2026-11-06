package amqp

import (
	"context"

	"github.com/Lama189/soundwave-platform/billing-service/internal/service/settlement"
)

type SettlementService interface {
	ProcessSettleOrder(ctx context.Context, input settlement.SettleOrderInput) error
}
