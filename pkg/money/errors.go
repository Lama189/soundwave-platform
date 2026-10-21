package money

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnsupportedCurrency = fmt.Errorf("%w: unsupported currency", ErrInvalidInput)
	ErrCurrencyMismatch    = fmt.Errorf("%w: currency mismatch", ErrInvalidInput)
	ErrNegativeAmount      = fmt.Errorf("%w: money amount cannot be negative", ErrInvalidInput)
	ErrZeroAmount          = fmt.Errorf("%w: money amount must be greater than zero", ErrInvalidInput)
)
