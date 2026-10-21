package domain

import (
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/pkg/money"
)

// Base errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
	ErrInvalidInput  = errors.New("invalid input")
	ErrForbidden     = errors.New("permission denied")
)

// Currency & Amount errors
var (
	ErrUnsupportedCurrency = money.ErrUnsupportedCurrency
	ErrCurrencyMismatch    = money.ErrCurrencyMismatch
	ErrNegativeAmount      = money.ErrNegativeAmount
	ErrZeroAmount          = money.ErrZeroAmount
)

// Order errors
var (
	ErrOrderNotFound         = fmt.Errorf("%w: order not found", ErrNotFound)
	ErrOrderAlreadyPaid      = fmt.Errorf("%w: order is already paid", ErrConflict)
	ErrOrderNotPending       = fmt.Errorf("%w: order is not in pending state", ErrConflict)
	ErrOrderCancelled        = fmt.Errorf("%w: order has been cancelled", ErrConflict)
	ErrEmptyOrderItems       = fmt.Errorf("%w: order must contain at least one item", ErrInvalidInput)
	ErrNotOrderBuyer         = fmt.Errorf("%w: you are not the buyer of this order", ErrForbidden)
	ErrTrackAlreadyPurchased = fmt.Errorf("%w: track has already been purchased", ErrAlreadyExists)
)
