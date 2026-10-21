package domain

import (
	"github.com/Lama189/soundwave-platform/pkg/money"
)

type Currency = money.Currency

const (
	CurrencyUSD Currency = money.USD
	CurrencyUZS Currency = money.UZS
	CurrencyRUB Currency = money.RUB
)
