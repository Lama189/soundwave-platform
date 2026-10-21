package domain

import (
	"github.com/Lama189/soundwave-platform/pkg/money"
)

type Currency = money.Currency
type Money = money.Money

const (
	CurrencyUSD Currency = money.USD
	CurrencyUZS Currency = money.UZS
	CurrencyRUB Currency = money.RUB
)

var (
	NewMoney     = money.New
	NewZeroMoney = money.NewZero
)
