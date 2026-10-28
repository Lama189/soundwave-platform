package mock

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/payment-service/internal/config"
	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
)

type Adapter struct {
	webhookSecret string
}

func NewAdapter(cfg config.MockAdapterConfig) *Adapter {
	return &Adapter{
		webhookSecret: cfg.WebhookSecret,
	}
}

func (a *Adapter) Provider() domain.PaymentProvider {
	return domain.PaymentProviderMock
}

func (a *Adapter) CreateSession(ctx context.Context, params domain.SessionParams) (*domain.PaymentSession, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate session params: %w", err)
	}

	sessionID := uuid.New().String()
	externalID := fmt.Sprintf("mock_pi_%s", sessionID)
	ClientSecret := fmt.Sprintf("mock_secret_%s", sessionID)
	redirectURL := fmt.Sprintf("http://localhost:8080/mock-checkout/%s", externalID)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	session := &domain.PaymentSession{
		ExternalID:   externalID,
		ClientSecret: ClientSecret,
		RedirectURL:  redirectURL,
		ExpiresAt:    &expiresAt,
	}

	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("validate mock session: %w", err)
	}

	return session, nil
}

func (a *Adapter) ParseAndVerifyWebhook(ctx context.Context, payload []byte, signature string) (*domain.NormalizedWebhookEvent, error) {
	if len(payload) == 0 {
		return nil, domain.ErrWebhookPayloadEmpty
	}

	if a.webhookSecret != "" {
		if subtle.ConstantTimeCompare([]byte(signature), []byte(a.webhookSecret)) != 1 {
			return nil, domain.ErrInvalidWebhookSignature
		}
	}

	var raw webhookPayload
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("%w: invalid json payload: %v", domain.ErrInvalidInput, err)
	}

	var eventType domain.NormalizedWebhookEventType
	switch raw.EventType {
	case "payment.succeeded":
		eventType = domain.WebhookTypePaymentSucceeded
	case "payment.failed":
		eventType = domain.WebhookTypePaymentFailed
	case "payment.refunded":
		eventType = domain.WebhookTypePaymentRefunded
	default:
		eventType = domain.WebhookTypeIgnored
	}

	event := domain.NormalizedWebhookEvent{
		ExternalEventID:   raw.EventID,
		ExternalPaymentID: raw.PaymentID,
		Type:              eventType,
		Amount:            raw.Amount,
		Currency:          domain.Currency(raw.Currency),
		FailureReason:     raw.FailureReason,
		RawPayload:        payload,
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate normalized mock event: %w", err)
	}

	return &event, nil
}
