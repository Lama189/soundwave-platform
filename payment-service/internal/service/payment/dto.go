package payment

import (
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
)

type CreatePaymentSessionInput struct {
	OrderID        uuid.UUID
	PayerUserID    uuid.UUID
	Provider       domain.PaymentProvider
	CustomerEmail  string
	IdempotencyKey string
}

type PaymentSessionOutput struct {
	PaymentID    uuid.UUID
	OrderID      uuid.UUID
	Status       domain.PaymentStatus
	Provider     domain.PaymentProvider
	ExternalID   string
	ClientSecret string
	RedirectURL  string
	Amount       int64
	Currency     domain.Currency
	ExpiresAt    *time.Time
}

type HandleWebhookInput struct {
	Provider  domain.PaymentProvider
	Payload   []byte
	Signature string
}

type PaymentOutput struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	PayerUserID    uuid.UUID
	Amount         int64
	Currency       domain.Currency
	Status         domain.PaymentStatus
	Provider       domain.PaymentProvider
	ExternalID     string
	ClientSecret   string
	FailureReason  *string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
