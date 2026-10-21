package domain

type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyUZS Currency = "UZS"
	CurrencyRUB Currency = "RUB"
)

func (c Currency) IsValid() bool {
	switch c {
	case CurrencyUSD, CurrencyRUB, CurrencyUZS:
		return true
	default:
		return false
	}
}
