package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID            uuid.UUID
	ArtistID      uuid.UUID
	Currency      Currency
	Balance       Money
	LockedBalance Money
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewWallet(artistID uuid.UUID, currency Currency) (*Wallet, error) {
	if artistID == uuid.Nil {
		return nil, fmt.Errorf("%w: artist id is required", ErrInvalidInput)
	}

	zeroBalance, err := NewZeroMoney(currency)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wallet balance: %w", err)
	}

	now := time.Now().UTC()
	w := &Wallet{
		ID:            uuid.New(),
		ArtistID:      artistID,
		Currency:      currency,
		Balance:       zeroBalance,
		LockedBalance: zeroBalance,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate wallet aggregate: %w", err)
	}

	return w, nil
}

func RestoreWallet(id, artistID uuid.UUID, currency Currency, balance, lockedBalance Money, createdAt, updatedAt time.Time) *Wallet {
	return &Wallet{
		ID:            id,
		ArtistID:      artistID,
		Currency:      currency,
		Balance:       balance,
		LockedBalance: lockedBalance,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func (w *Wallet) Credit(amount Money) error {
	if err := w.validateOperationAmount(amount); err != nil {
		return err
	}

	newBalance, err := w.Balance.Add(amount)
	if err != nil {
		return err
	}

	w.Balance = newBalance
	w.UpdatedAt = time.Now().UTC()

	return nil
}

func (w *Wallet) LockForPayout(amount Money) error {
	if err := w.validateOperationAmount(amount); err != nil {
		return err
	}

	if w.Balance.Amount < amount.Amount {
		return fmt.Errorf("%w: requested %d, available %d", ErrInsufficientFunds, amount.Amount, w.Balance.Amount)
	}

	newBalance, err := w.Balance.Subtract(amount)
	if err != nil {
		return err
	}
	newLocked, err := w.LockedBalance.Add(amount)
	if err != nil {
		return err
	}

	w.Balance = newBalance
	w.LockedBalance = newLocked
	w.UpdatedAt = time.Now().UTC()

	return nil
}

func (w *Wallet) Unlock(amount Money) error {
	if err := w.validateOperationAmount(amount); err != nil {
		return err
	}

	if w.LockedBalance.Amount < amount.Amount {
		return fmt.Errorf("%w: requested %d, locked %d", ErrInsufficientLockedFunds, amount.Amount, w.LockedBalance.Amount)
	}

	newLocked, err := w.LockedBalance.Subtract(amount)
	if err != nil {
		return err
	}
	newBalance, err := w.Balance.Add(amount)
	if err != nil {
		return err
	}

	w.LockedBalance = newLocked
	w.Balance = newBalance
	w.UpdatedAt = time.Now().UTC()

	return nil
}

func (w *Wallet) ConfirmPayout(amount Money) error {
	if err := w.validateOperationAmount(amount); err != nil {
		return err
	}

	if w.LockedBalance.Amount < amount.Amount {
		return fmt.Errorf("%w: requested %d, locked %d", ErrInsufficientLockedFunds, amount.Amount, w.LockedBalance.Amount)
	}

	newLocked, err := w.LockedBalance.Subtract(amount)
	if err != nil {
		return err
	}

	w.LockedBalance = newLocked
	w.UpdatedAt = time.Now().UTC()

	return nil
}

func (w *Wallet) Validate() error {
	if w.ID == uuid.Nil {
		return fmt.Errorf("%w: wallet id is required", ErrInvalidInput)
	}
	if w.ArtistID == uuid.Nil {
		return fmt.Errorf("%w: artist id is required", ErrInvalidInput)
	}

	if !w.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, w.Currency)
	}

	if err := w.Balance.Validate(); err != nil {
		return fmt.Errorf("invalid balance: %w", err)
	}
	if err := w.LockedBalance.Validate(); err != nil {
		return fmt.Errorf("invalid locked balance: %w", err)
	}

	return nil
}

func (w *Wallet) validateOperationAmount(amount Money) error {
	if err := amount.Validate(); err != nil {
		return fmt.Errorf("invalid money: %w", err)
	}

	if amount.Currency != w.Currency {
		return fmt.Errorf("%w: wallet currency is %q, but got %q", ErrCurrencyMismatch, w.Currency, amount.Currency)
	}

	if !amount.IsPositive() {
		return ErrZeroAmount
	}

	return nil
}
