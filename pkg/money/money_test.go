package money_test

import (
	"errors"
	"testing"

	"github.com/Lama189/soundwave-platform/pkg/money"
)

func TestCurrency_IsValid(t *testing.T) {
	tests := []struct {
		currency money.Currency
		expected bool
	}{
		{money.USD, true},
		{money.UZS, true},
		{money.RUB, true},
		{money.Currency("EUR"), false},
		{money.Currency(""), false},
	}

	for _, tt := range tests {
		if got := tt.currency.IsValid(); got != tt.expected {
			t.Errorf("Currency(%q).IsValid() = %v; want %v", tt.currency, got, tt.expected)
		}
	}
}

func TestMoney_New(t *testing.T) {
	t.Run("valid money", func(t *testing.T) {
		m, err := money.New(100, money.USD)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Amount != 100 || m.Currency != money.USD {
			t.Errorf("got %v, want 100 USD", m)
		}
	})

	t.Run("negative amount fails", func(t *testing.T) {
		_, err := money.New(-1, money.USD)
		if !errors.Is(err, money.ErrNegativeAmount) {
			t.Errorf("expected ErrNegativeAmount, got %v", err)
		}
	})

	t.Run("invalid currency fails", func(t *testing.T) {
		_, err := money.New(100, money.Currency("INVALID"))
		if !errors.Is(err, money.ErrUnsupportedCurrency) {
			t.Errorf("expected ErrUnsupportedCurrency, got %v", err)
		}
	})
}

func TestMoney_Add(t *testing.T) {
	m1, _ := money.New(100, money.USD)
	m2, _ := money.New(50, money.USD)

	res, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Amount != 150 {
		t.Errorf("got amount %d, want 150", res.Amount)
	}

	mDiff, _ := money.New(50, money.RUB)
	_, err = m1.Add(mDiff)
	if !errors.Is(err, money.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestMoney_Subtract(t *testing.T) {
	m1, _ := money.New(100, money.USD)
	m2, _ := money.New(40, money.USD)

	res, err := m1.Subtract(m2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Amount != 60 {
		t.Errorf("got amount %d, want 60", res.Amount)
	}

	mBigger, _ := money.New(150, money.USD)
	_, err = m1.Subtract(mBigger)
	if !errors.Is(err, money.ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestMoney_Multiply(t *testing.T) {
	m, _ := money.New(100, money.USD)

	res, err := m.Multiply(1.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Amount != 150 {
		t.Errorf("got amount %d, want 150", res.Amount)
	}

	_, err = m.Multiply(-0.5)
	if !errors.Is(err, money.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
