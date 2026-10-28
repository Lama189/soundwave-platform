package payment

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/grpcclient"
	"github.com/google/uuid"
)

type PaymentGateway interface {
	Provider() domain.PaymentProvider
	CreateSession(ctx context.Context, params domain.SessionParams) (*domain.PaymentSession, error)
	ParseAndVerifyWebhook(ctx context.Context, payload []byte, signature string) (*domain.NormalizedWebhookEvent, error)
}

type OrderClient interface {
	GetOrder(ctx context.Context, req grpcclient.GetOrderRequest) (*grpcclient.OrderResponse, error)
}

type PaymentService interface {
	CreatePaymentSession(ctx context.Context, input CreatePaymentSessionInput) (*PaymentSessionOutput, error)
	HandleWebhook(ctx context.Context, input HandleWebhookInput) error
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*PaymentOutput, error)
	GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*PaymentOutput, error)
}
