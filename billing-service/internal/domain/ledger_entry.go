package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LedgerOperationType string

const (
	LedgerOpTrackSale     LedgerOperationType = "track_sale"
	LedgerOpPlatformFee   LedgerOperationType = "platform_fee"
	LedgerOpPayoutLock    LedgerOperationType = "payout_lock"
	LedgerOpPayoutSuccess LedgerOperationType = "payout_success"
	LedgerOpPayoutRefund  LedgerOperationType = "payout_refund"
)

func (t LedgerOperationType) IsValid() bool {
	switch t {
	case LedgerOpTrackSale, LedgerOpPlatformFee, LedgerOpPayoutLock, LedgerOpPayoutSuccess, LedgerOpPayoutRefund:
		return true
	default:
		return false
	}
}

type ReferenceType string

const (
	ReferenceTypeOrder  ReferenceType = "order"
	ReferenceTypePayout ReferenceType = "payout"
)

func (t ReferenceType) IsValid() bool {
	switch t {
	case ReferenceTypeOrder, ReferenceTypePayout:
		return true
	default:
		return false
	}
}

type LedgerEntry struct {
	ID            uuid.UUID
	WalletID      uuid.UUID
	OperationType LedgerOperationType
	Amount        Money
	BalanceAfter  Money
	ReferenceType ReferenceType
	ReferenceID   uuid.UUID
	Description   string
	CreatedAt     time.Time
}

func NewLedgerEntry(
	walletID uuid.UUID,
	opType LedgerOperationType,
	amount Money,
	balanceAfter Money,
	refType ReferenceType,
	refID uuid.UUID,
	description string,
) (*LedgerEntry, error) {
	entry := &LedgerEntry{
		ID:            uuid.New(),
		WalletID:      walletID,
		OperationType: opType,
		Amount:        amount,
		BalanceAfter:  balanceAfter,
		ReferenceType: refType,
		ReferenceID:   refID,
		Description:   description,
		CreatedAt:     time.Now().UTC(),
	}

	if err := entry.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create ledger entry: %w", err)
	}

	return entry, nil
}

func RestoreLedgerEntry(
	id uuid.UUID,
	walletID uuid.UUID,
	opType LedgerOperationType,
	amount Money,
	balanceAfter Money,
	refType ReferenceType,
	refID uuid.UUID,
	description string,
	createdAt time.Time,
) *LedgerEntry {
	return &LedgerEntry{
		ID:            id,
		WalletID:      walletID,
		OperationType: opType,
		Amount:        amount,
		BalanceAfter:  balanceAfter,
		ReferenceType: refType,
		ReferenceID:   refID,
		Description:   description,
		CreatedAt:     createdAt,
	}
}

func (t LedgerOperationType) IsDebit() bool {
	switch t {
	case LedgerOpPlatformFee, LedgerOpPayoutLock, LedgerOpPayoutSuccess:
		return true
	default:
		return false
	}
}

func (t LedgerOperationType) IsCredit() bool {
	switch t {
	case LedgerOpTrackSale, LedgerOpPayoutRefund:
		return true
	default:
		return false
	}
}

func (e *LedgerEntry) Validate() error {
	if e.ID == uuid.Nil {
		return fmt.Errorf("%w: ledger entry id is required", ErrInvalidInput)
	}
	if e.WalletID == uuid.Nil {
		return fmt.Errorf("%w: ledger entry wallet id is required", ErrInvalidInput)
	}
	if e.ReferenceID == uuid.Nil {
		return fmt.Errorf("%w: ledger reference id is required", ErrInvalidInput)
	}

	if !e.OperationType.IsValid() {
		return fmt.Errorf("%w: invalid operation type: %s", ErrInvalidInput, e.OperationType)
	}
	if !e.ReferenceType.IsValid() {
		return fmt.Errorf("%w: invalid reference type: %s", ErrInvalidInput, e.ReferenceType)
	}

	if err := e.Amount.Validate(); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := e.BalanceAfter.Validate(); err != nil {
		return fmt.Errorf("invalid balance after: %w", err)
	}
	if e.Amount.Currency != e.BalanceAfter.Currency {
		return fmt.Errorf("%w: amount (%s), balance after (%s)", ErrCurrencyMismatch, e.Amount.Currency, e.BalanceAfter.Currency)
	}

	if e.OperationType.IsDebit() && e.Amount.IsPositive() {
		return fmt.Errorf("%w: debit operation %s must have negative amount", ErrInvalidInput, e.OperationType)
	}

	switch e.OperationType {
	case LedgerOpTrackSale, LedgerOpPlatformFee:
		if e.ReferenceType != ReferenceTypeOrder {
			return fmt.Errorf("%w: operation %s requires reference type %s, got %s",
				ErrInvalidInput, e.OperationType, ReferenceTypeOrder, e.ReferenceType)
		}
	case LedgerOpPayoutLock, LedgerOpPayoutSuccess, LedgerOpPayoutRefund:
		if e.ReferenceType != ReferenceTypePayout {
			return fmt.Errorf("%w: operation %s requires reference type %s, got %s",
				ErrInvalidInput, e.OperationType, ReferenceTypePayout, e.ReferenceType)
		}
	}

	return nil
}
