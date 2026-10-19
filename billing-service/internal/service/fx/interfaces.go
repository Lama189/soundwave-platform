package fx

import (
	"context"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/shopspring/decimal"
)

type RateCache interface {
	Get(ctx context.Context, from, to domain.Currency) (decimal.Decimal, error)
	Set(ctx context.Context, from, to domain.Currency, rate decimal.Decimal) error
}

type RateProvider interface {
	FetchRates(ctx context.Context, base domain.Currency) (map[domain.Currency]decimal.Decimal, error)
}
