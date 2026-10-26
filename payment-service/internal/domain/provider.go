package domain

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Lama189/soundwave-platform/pkg/money"
	"github.com/google/uuid"
)

type PaymentGateway interface {
	Provider() PaymentProvider
	CreateSession(ctx context.Context, params SessionParams) (*PaymentSession, error)
	ParseAndVerifyWebhook(ctx context.Context, payload []byte, signature string) (*NormalizedWebhookEvent, error)
}

type SessionParams struct {
	PaymentID      uuid.UUID
	OrderID        uuid.UUID
	Amount         int64
	Currency       money.Currency
	CustomerEmail  string
	IdempotencyKey string
}

func (p SessionParams) Validate() error {
	if p.PaymentID == uuid.Nil {
		return fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}
	if p.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}
	if p.Amount <= 0 {
		if p.Amount == 0 {
			return ErrZeroAmount
		}
		return ErrNegativeAmount
	}
	if !p.Currency.IsValid() {
		return ErrUnsupportedCurrency
	}
	if strings.TrimSpace(p.IdempotencyKey) == "" {
		return ErrIdempotencyKeyRequired
	}
	return nil
}

type PaymentSession struct {
	ExternalID   string
	ClientSecret string
	RedirectURL  string
	ExpiresAt    *time.Time
}

func (s PaymentSession) Validate() error {
	if strings.TrimSpace(s.ExternalID) == "" {
		return ErrExternalIDRequired
	}
	if s.RedirectURL != "" {
		if _, err := url.ParseRequestURI(s.RedirectURL); err != nil {
			return fmt.Errorf("%w: invalid redirect url", ErrInvalidInput)
		}
	}
	if s.ExpiresAt != nil && s.ExpiresAt.Before(time.Now().UTC()) {
		return fmt.Errorf("%w: session already expired", ErrInvalidInput)
	}
	return nil
}

type NormalizedWebhookEventType string

const (
	WebhookTypePaymentSucceeded NormalizedWebhookEventType = "payment.succeeded"
	WebhookTypePaymentFailed    NormalizedWebhookEventType = "payment.failed"
	WebhookTypePaymentRefunded  NormalizedWebhookEventType = "payment.refunded"
	WebhookTypeIgnored          NormalizedWebhookEventType = "ignored"
)

func (t NormalizedWebhookEventType) IsValid() bool {
	switch t {
	case WebhookTypePaymentSucceeded, WebhookTypePaymentFailed, WebhookTypePaymentRefunded, WebhookTypeIgnored:
		return true
	default:
		return false
	}
}

type NormalizedWebhookEvent struct {
	ExternalEventID   string
	ExternalPaymentID string
	Type              NormalizedWebhookEventType
	Amount            int64
	Currency          money.Currency
	FailureReason     string
	RawPayload        []byte
}

func (e NormalizedWebhookEvent) Validate() error {
	if strings.TrimSpace(e.ExternalEventID) == "" {
		return fmt.Errorf("%w: external event id is required", ErrInvalidInput)
	}
	if !e.Type.IsValid() {
		return fmt.Errorf("%w: invalid normalized event type", ErrInvalidInput)
	}
	if e.Type != WebhookTypeIgnored && strings.TrimSpace(e.ExternalPaymentID) == "" {
		return ErrExternalIDRequired
	}
	if len(e.RawPayload) == 0 {
		return ErrWebhookPayloadEmpty
	}
	return nil
}
