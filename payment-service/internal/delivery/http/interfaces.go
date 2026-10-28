package http

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
	"github.com/google/uuid"
)

type PaymentService interface {
	CreatePaymentSession(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error)
	HandleWebhook(ctx context.Context, input payment.HandleWebhookInput) error
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*payment.PaymentOutput, error)
	GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*payment.PaymentOutput, error)
}
