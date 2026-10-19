package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted  PayoutStatus = "completed"
	PayoutStatusFailed     PayoutStatus = "failed"
)

func (s PayoutStatus) IsValid() bool {
	switch s {
	case PayoutStatusPending, PayoutStatusProcessing, PayoutStatusCompleted, PayoutStatusFailed:
		return true
	default:
		return false
	}
}

type Payout struct {
	ID              uuid.UUID
	ArtistID        uuid.UUID
	WalletID        uuid.UUID
	Amount          Money
	DestinationCard string
	Status          PayoutStatus
	FailureReason   *string
	RequestedAt     time.Time
	ProcessedAt     *time.Time
}

func NewPayout(artistID uuid.UUID, walletID uuid.UUID, amount Money, destinationCard string) (*Payout, error) {
	p := &Payout{
		ID:              uuid.New(),
		ArtistID:        artistID,
		WalletID:        walletID,
		Amount:          amount,
		DestinationCard: strings.TrimSpace(destinationCard),
		Status:          PayoutStatusPending,
		FailureReason:   nil,
		RequestedAt:     time.Now().UTC(),
		ProcessedAt:     nil,
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create payout: %w", err)
	}

	return p, nil
}

func (p *Payout) MarkProcessing() error {
	if p.Status != PayoutStatusPending {
		return fmt.Errorf("%w: cannot process payout in status %s", ErrPayoutInvalidStateTransition, p.Status)
	}

	p.Status = PayoutStatusProcessing
	return nil
}

func (p *Payout) Complete() error {
	if p.Status != PayoutStatusProcessing {
		return fmt.Errorf("%w: cannot complete payout in status %s", ErrPayoutInvalidStateTransition, p.Status)
	}

	now := time.Now().UTC()
	p.Status = PayoutStatusCompleted
	p.ProcessedAt = &now
	p.FailureReason = nil

	return nil
}

func (p *Payout) Fail(reason string) error {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return fmt.Errorf("%w: failure reason cannot be empty", ErrInvalidInput)
	}

	if p.Status != PayoutStatusPending && p.Status != PayoutStatusProcessing {
		return fmt.Errorf("%w: cannot fail payout in status %s", ErrPayoutInvalidStateTransition, p.Status)
	}

	now := time.Now().UTC()
	p.Status = PayoutStatusFailed
	p.ProcessedAt = &now
	p.FailureReason = &trimmedReason

	return nil
}

func RestorePayout(
	id uuid.UUID,
	artistID uuid.UUID,
	walletID uuid.UUID,
	amount Money,
	destinationCard string,
	status PayoutStatus,
	failureReason *string,
	requestedAt time.Time,
	processedAt *time.Time,
) *Payout {
	return &Payout{
		ID:              id,
		ArtistID:        artistID,
		WalletID:        walletID,
		Amount:          amount,
		DestinationCard: destinationCard,
		Status:          status,
		FailureReason:   failureReason,
		RequestedAt:     requestedAt,
		ProcessedAt:     processedAt,
	}
}

func (p *Payout) Validate() error {
	if p.ID == uuid.Nil {
		return fmt.Errorf("%w: payout id is required", ErrInvalidInput)
	}
	if p.ArtistID == uuid.Nil {
		return fmt.Errorf("%w: artist id is required", ErrInvalidInput)
	}
	if p.WalletID == uuid.Nil {
		return fmt.Errorf("%w: wallet id is required", ErrInvalidInput)
	}

	if err := p.Amount.Validate(); err != nil {
		return fmt.Errorf("invalid payout amount: %w", err)
	}
	if !p.Amount.IsPositive() {
		return ErrZeroAmount
	}

	if strings.TrimSpace(p.DestinationCard) == "" {
		return ErrInvalidDestinationAccount
	}

	if !p.Status.IsValid() {
		return fmt.Errorf("%w: invalid payout status: %s", ErrInvalidInput, p.Status)
	}

	if p.RequestedAt.IsZero() {
		return fmt.Errorf("%w: requested_at timestamp is required", ErrInvalidInput)
	}

	switch p.Status {
	case PayoutStatusPending:
		if p.ProcessedAt != nil {
			return fmt.Errorf("%w: pending payout cannot have processed_at timestamp", ErrConflict)
		}
		if p.FailureReason != nil {
			return fmt.Errorf("%w: pending payout cannot have failure_reason", ErrConflict)
		}

	case PayoutStatusCompleted:
		if p.ProcessedAt == nil {
			return fmt.Errorf("%w: completed payout must have processed_at timestamp", ErrConflict)
		}
		if p.FailureReason != nil {
			return fmt.Errorf("%w: completed payout cannot have failure_reason", ErrConflict)
		}

	case PayoutStatusFailed:
		if p.ProcessedAt == nil {
			return fmt.Errorf("%w: failed payout must have processed_at timestamp", ErrConflict)
		}
		if p.FailureReason == nil || strings.TrimSpace(*p.FailureReason) == "" {
			return fmt.Errorf("%w: failed payout must have a non-empty failure_reason", ErrInvalidInput)
		}
	}

	return nil
}
