package domain

import (
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/pkg/money"
)

// Base sentinel errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
	ErrInvalidInput  = errors.New("invalid input")
	ErrForbidden     = errors.New("permission denied")
	ErrUnauthorized  = errors.New("unauthorized")
)

// Amount and currency errors (re-exported from pkg/money)
var (
	ErrUnsupportedCurrency = money.ErrUnsupportedCurrency
	ErrCurrencyMismatch    = money.ErrCurrencyMismatch
	ErrNegativeAmount      = money.ErrNegativeAmount
	ErrZeroAmount          = money.ErrZeroAmount
)

// Payment domain errors
var (
	ErrPaymentNotFound         = fmt.Errorf("%w: payment not found", ErrNotFound)
	ErrPaymentAlreadyExists    = fmt.Errorf("%w: payment for this order already exists", ErrAlreadyExists)
	ErrPaymentAlreadyCompleted = fmt.Errorf("%w: payment is already in terminal state", ErrConflict)
	ErrInvalidStatusTransition = fmt.Errorf("%w: invalid payment status transition", ErrConflict)
	ErrUnsupportedProvider     = fmt.Errorf("%w: unsupported payment provider", ErrInvalidInput)
	ErrExternalIDRequired      = fmt.Errorf("%w: external payment id is required", ErrInvalidInput)
	ErrIdempotencyKeyRequired  = fmt.Errorf("%w: idempotency key is required", ErrInvalidInput)
)

// Webhook and provider domain errors
var (
	ErrInvalidWebhookSignature = fmt.Errorf("%w: invalid webhook signature", ErrUnauthorized)
	ErrWebhookAlreadyProcessed = fmt.Errorf("%w: webhook event already processed", ErrConflict)
	ErrWebhookPayloadEmpty     = fmt.Errorf("%w: webhook payload is empty", ErrInvalidInput)
	ErrProviderCommunication   = errors.New("payment provider communication failure")
)

// External service integration errors
var (
	ErrOrderNotFound      = fmt.Errorf("%w: order not found", ErrNotFound)
	ErrOrderAlreadyPaid   = fmt.Errorf("%w: order is already marked as paid", ErrConflict)
	ErrOrderCommunication = errors.New("order service communication failure")
)
