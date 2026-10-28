package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
)

type PaymentService interface {
	CreatePaymentSession(ctx context.Context, input payment.CreatePaymentSessionInput) (*payment.PaymentSessionOutput, error)
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*payment.PaymentOutput, error)
	GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*payment.PaymentOutput, error)
}
