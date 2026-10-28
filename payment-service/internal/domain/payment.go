package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusProcessing, PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

type PaymentProvider string

const (
	PaymentProviderStripe PaymentProvider = "stripe"
	PaymentProviderMock   PaymentProvider = "mock"
)

func (p PaymentProvider) IsValid() bool {
	switch p {
	case PaymentProviderMock, PaymentProviderStripe:
		return true
	default:
		return false
	}
}

type Payment struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	PayerUserID    uuid.UUID
	Amount         int64
	Currency       Currency
	Status         PaymentStatus
	Provider       PaymentProvider
	ExternalID     string
	ClientSecret   string
	FailureReason  *string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewPayment(orderID, payerUserID uuid.UUID, amount int64, currency Currency, provider PaymentProvider, idempotencyKey string) (*Payment, error) {
	now := time.Now().UTC()

	p := &Payment{
		ID:             uuid.New(),
		OrderID:        orderID,
		PayerUserID:    payerUserID,
		Amount:         amount,
		Currency:       currency,
		Status:         PaymentStatusPending,
		Provider:       provider,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate new payment: %w", err)
	}

	return p, nil
}

func RestorePayment(id, orderID, payerUserID uuid.UUID, amount int64, currency Currency, status PaymentStatus, provider PaymentProvider, externalID, clientSecret, idempotencyKey string, failureReason *string, createdAt, updatedAt time.Time) (*Payment, error) {
	p := &Payment{
		ID:             id,
		OrderID:        orderID,
		PayerUserID:    payerUserID,
		Amount:         amount,
		Currency:       currency,
		Status:         status,
		Provider:       provider,
		ExternalID:     externalID,
		ClientSecret:   clientSecret,
		FailureReason:  failureReason,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to restore corrupted payment: %w", err)
	}

	return p, nil
}

func (p *Payment) IsTerminal() bool {
	if p == nil {
		return false
	}
	switch p.Status {
	case PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

func (p *Payment) MarkProcessing(externalID, clientSecret string) error {
	if err := p.ensureCanTransition(PaymentStatusProcessing); err != nil {
		return err
	}

	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return ErrExternalIDRequired
	}

	p.Status = PaymentStatusProcessing
	p.ExternalID = externalID
	p.ClientSecret = strings.TrimSpace(clientSecret)
	p.UpdatedAt = time.Now().UTC()

	return nil
}

func (p *Payment) MarkSucceeded(externalID string) error {
	if err := p.ensureCanTransition(PaymentStatusSucceeded); err != nil {
		return err
	}

	externalID = strings.TrimSpace(externalID)
	if p.ExternalID == "" && externalID == "" {
		return ErrExternalIDRequired
	}
	if externalID != "" {
		p.ExternalID = externalID
	}

	p.Status = PaymentStatusSucceeded
	p.FailureReason = nil
	p.UpdatedAt = time.Now().UTC()

	return nil
}

func (p *Payment) MarkFailed(reason string, externalID string) error {
	if err := p.ensureCanTransition(PaymentStatusFailed); err != nil {
		return err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "payment failed without specific reason"
	}

	if extID := strings.TrimSpace(externalID); extID != "" {
		p.ExternalID = extID
	}

	p.Status = PaymentStatusFailed
	p.FailureReason = &reason
	p.UpdatedAt = time.Now().UTC()

	return nil
}

func (p *Payment) MarkCancelled() error {
	if err := p.ensureCanTransition(PaymentStatusCancelled); err != nil {
		return err
	}

	p.Status = PaymentStatusCancelled
	p.UpdatedAt = time.Now().UTC()

	return nil
}

func (p *Payment) MarkRefunded() error {
	if err := p.ensureCanTransition(PaymentStatusRefunded); err != nil {
		return err
	}

	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now().UTC()

	return nil
}

func (p *Payment) Validate() error {
	if p == nil {
		return ErrInvalidInput
	}
	if p.ID == uuid.Nil {
		return fmt.Errorf("%w: payment id is required", ErrInvalidInput)
	}
	if p.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order id is required", ErrInvalidInput)
	}
	if p.PayerUserID == uuid.Nil {
		return fmt.Errorf("%w: payer user id is required", ErrInvalidInput)
	}

	if p.Amount <= 0 {
		if p.Amount == 0 {
			return ErrZeroAmount
		}
		return ErrNegativeAmount
	}
	if !p.Currency.IsValid() {
		return fmt.Errorf("%w: %s", ErrUnsupportedCurrency, p.Currency)
	}

	if !p.Status.IsValid() {
		return fmt.Errorf("%w: invalid status %s", ErrInvalidInput, p.Status)
	}
	if !p.Provider.IsValid() {
		return fmt.Errorf("%w: unknown provider %s", ErrUnsupportedProvider, p.Provider)
	}

	if strings.TrimSpace(p.IdempotencyKey) == "" {
		return ErrIdempotencyKeyRequired
	}

	switch p.Status {
	case PaymentStatusProcessing, PaymentStatusSucceeded, PaymentStatusRefunded:
		if strings.TrimSpace(p.ExternalID) == "" {
			return fmt.Errorf("%w: external_id required for status %s", ErrExternalIDRequired, p.Status)
		}
	case PaymentStatusFailed:
		if p.FailureReason == nil || strings.TrimSpace(*p.FailureReason) == "" {
			return fmt.Errorf("%w: failure reason required when payment failed", ErrInvalidInput)
		}
	}

	return nil
}

func (p *Payment) CanTransitionTo(next PaymentStatus) bool {
	if p == nil {
		return false
	}
	if p.Status == next {
		return false
	}

	switch p.Status {
	case PaymentStatusPending:
		switch next {
		case PaymentStatusProcessing, PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusCancelled:
			return true
		}

	case PaymentStatusProcessing:
		switch next {
		case PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusCancelled:
			return true
		}

	case PaymentStatusSucceeded:
		return next == PaymentStatusRefunded
	}

	return false
}

func (p *Payment) ensureCanTransition(next PaymentStatus) error {
	if p == nil {
		return ErrNotFound
	}
	if p.IsTerminal() && !(p.Status == PaymentStatusSucceeded && next == PaymentStatusRefunded) {
		return ErrPaymentAlreadyCompleted
	}
	if !p.CanTransitionTo(next) {
		return ErrInvalidStatusTransition
	}
	return nil
}
