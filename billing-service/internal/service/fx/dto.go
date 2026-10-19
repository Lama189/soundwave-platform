package fx

import (
	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

type ConvertCurrency struct {
	Source         domain.Money
	TargetCurrency domain.Currency
}

type ConvertCurrencyOutputDTO struct {
	Converted domain.Money
	Rate      decimal.Decimal
}
