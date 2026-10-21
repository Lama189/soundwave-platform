package money

import (
	"fmt"
	"math"
)

type Money struct {
	Amount   int64
	Currency Currency
}

func New(amount int64, currency Currency) (Money, error) {
	m := Money{
		Amount:   amount,
		Currency: currency,
	}

	if err := m.Validate(); err != nil {
		return Money{}, fmt.Errorf("failed to create money: %w", err)
	}

	return m, nil
}

func NewZero(currency Currency) (Money, error) {
	if !currency.IsValid() {
		return Money{}, fmt.Errorf("%w: %q", ErrUnsupportedCurrency, currency)
	}

	return Money{
		Amount:   0,
		Currency: currency,
	}, nil
}

func (m Money) Add(amount Money) (Money, error) {
	if m.Currency != amount.Currency {
		return Money{}, fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.Currency, amount.Currency)
	}

	return Money{
		Amount:   m.Amount + amount.Amount,
		Currency: m.Currency,
	}, nil
}

func (m Money) Subtract(amount Money) (Money, error) {
	if m.Currency != amount.Currency {
		return Money{}, fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.Currency, amount.Currency)
	}

	if m.Amount < amount.Amount {
		return Money{}, fmt.Errorf("%w: cannot subtract %d from %d", ErrNegativeAmount, amount.Amount, m.Amount)
	}

	return Money{
		Amount:   m.Amount - amount.Amount,
		Currency: m.Currency,
	}, nil
}

func (m Money) Multiply(factor float64) (Money, error) {
	if factor < 0 {
		return Money{}, fmt.Errorf("%w: multiplication factor cannot be negative (%f)", ErrInvalidInput, factor)
	}

	newAmount := int64(math.Round(float64(m.Amount) * factor))
	return New(newAmount, m.Currency)
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}

func (m Money) IsPositive() bool {
	return m.Amount > 0
}

func (m Money) Validate() error {
	if m.Amount < 0 {
		return fmt.Errorf("%w: %d", ErrNegativeAmount, m.Amount)
	}

	if !m.Currency.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnsupportedCurrency, m.Currency)
	}

	return nil
}
