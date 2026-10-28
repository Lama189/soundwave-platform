package payment

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/grpcclient"
)

type PaymentGateway interface {
	Provider() domain.PaymentProvider
	CreateSession(ctx context.Context, params domain.SessionParams) (*domain.PaymentSession, error)
	ParseAndVerifyWebhook(ctx context.Context, payload []byte, signature string) (*domain.NormalizedWebhookEvent, error)
}

type OrderClient interface {
	GetOrder(ctx context.Context, req grpcclient.GetOrderRequest) (*grpcclient.OrderResponse, error)
}
