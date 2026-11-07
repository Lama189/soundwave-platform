package amqp

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
)

type PaymentService interface {
	ProcessRefundCommand(ctx context.Context, input payment.RefundCommandInput) error
}
