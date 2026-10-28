package payment

import (
	"fmt"
	"strings"
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

func (input *CreatePaymentSessionInput) Validate() error {
	if input.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", domain.ErrInvalidInput)
	}
	if input.PayerUserID == uuid.Nil {
		return fmt.Errorf("%w: payer user id is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return domain.ErrIdempotencyKeyRequired
	}

	return nil
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

func toPaymentSessionOutput(p *domain.Payment, redirectURL string) *PaymentSessionOutput {
	return &PaymentSessionOutput{
		PaymentID:    p.ID,
		OrderID:      p.OrderID,
		Status:       p.Status,
		Provider:     p.Provider,
		ExternalID:   p.ExternalID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  redirectURL,
		Amount:       p.Amount,
		Currency:     p.Currency,
	}
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

func toPaymentOutput(p *domain.Payment) *PaymentOutput {
	return &PaymentOutput{
		ID:             p.ID,
		OrderID:        p.OrderID,
		PayerUserID:    p.PayerUserID,
		Amount:         p.Amount,
		Currency:       p.Currency,
		Status:         p.Status,
		Provider:       p.Provider,
		ExternalID:     p.ExternalID,
		ClientSecret:   p.ClientSecret,
		FailureReason:  p.FailureReason,
		IdempotencyKey: p.IdempotencyKey,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}
