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

// Money & Currency errors
var (
	ErrUnsupportedCurrency = money.ErrUnsupportedCurrency
	ErrCurrencyMismatch    = money.ErrCurrencyMismatch
	ErrNegativeAmount      = money.ErrNegativeAmount
	ErrZeroAmount          = money.ErrZeroAmount
)

// Order errors
var (
	ErrOrderNotFound          = fmt.Errorf("%w: order not found", ErrNotFound)
	ErrOrderAlreadyPaid       = fmt.Errorf("%w: order is already paid", ErrConflict)
	ErrOrderNotPending        = fmt.Errorf("%w: order is not in pending state", ErrConflict)
	ErrOrderCancelled         = fmt.Errorf("%w: order has been cancelled", ErrConflict)
	ErrEmptyOrderItems        = fmt.Errorf("%w: order must contain at least one item", ErrInvalidInput)
	ErrNotOrderBuyer          = fmt.Errorf("%w: you are not the buyer of this order", ErrForbidden)
	ErrTrackAlreadyPurchased  = fmt.Errorf("%w: track has already been purchased", ErrAlreadyExists)
)

// Wallet errors
var (
	ErrWalletNotFound           = fmt.Errorf("%w: wallet not found", ErrNotFound)
	ErrWalletAlreadyExists       = fmt.Errorf("%w: wallet already exists for artist", ErrAlreadyExists)
	ErrInsufficientFunds        = fmt.Errorf("%w: insufficient available funds on wallet balance", ErrInvalidInput)
	ErrInsufficientLockedFunds  = fmt.Errorf("%w: insufficient funds on locked balance", ErrInvalidInput)
	ErrNotWalletOwner           = fmt.Errorf("%w: you are not the owner of this wallet", ErrForbidden)
)

// Payout errors
var (
	ErrPayoutNotFound               = fmt.Errorf("%w: payout request not found", ErrNotFound)
	ErrPayoutInvalidStateTransition = fmt.Errorf("%w: invalid payout state transition", ErrConflict)
	ErrPayoutMinAmountNotMet        = fmt.Errorf("%w: payout amount is below minimum threshold", ErrInvalidInput)
	ErrInvalidDestinationAccount    = fmt.Errorf("%w: invalid destination card or account", ErrInvalidInput)
	ErrNotPayoutOwner               = fmt.Errorf("%w: you are not the owner of this payout request", ErrForbidden)
)

// Payment & Gateway errors
var (
	ErrPaymentFailed        = fmt.Errorf("%w: payment processing failed", ErrConflict)
	ErrPaymentRejected      = fmt.Errorf("%w: payment was rejected by provider", ErrConflict)
	ErrInvalidPaymentMethod = fmt.Errorf("%w: invalid payment method", ErrInvalidInput)
)
