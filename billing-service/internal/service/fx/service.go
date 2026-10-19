package fx

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/config"
	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
)

type FXService struct {
	cache           RateCache
	provider        RateProvider
	defaultBaseCurr domain.Currency
	sf              singleflight.Group
}

func NewFXService(cache RateCache, provider RateProvider, cfg config.FXConfig) *FXService {
	baseCurr := cfg.DefaultCurrency
	if !baseCurr.IsValid() {
		baseCurr = domain.CurrencyUSD
	}

	return &FXService{
		cache:           cache,
		provider:        provider,
		defaultBaseCurr: baseCurr,
	}
}

func (s *FXService) Convert(ctx context.Context, input ConvertCurrency) (ConvertCurrencyOutputDTO, error) {
	if err := input.Source.Validate(); err != nil {
		return ConvertCurrencyOutputDTO{}, fmt.Errorf("invalid source money: %w", err)
	}
	if !input.TargetCurrency.IsValid() {
		return ConvertCurrencyOutputDTO{}, fmt.Errorf("invalid target currency: %s", input.TargetCurrency)
	}

	if input.Source.Currency == input.TargetCurrency {
		return ConvertCurrencyOutputDTO{
			Converted: input.Source,
			Rate:      decimal.NewFromInt(1),
		}, nil
	}

	rate, err := s.getRate(ctx, input.Source.Currency, input.TargetCurrency)
	if err != nil {
		rate, err = s.fetchRate(ctx, input.Source.Currency, input.TargetCurrency)
		if err != nil {
			return ConvertCurrencyOutputDTO{}, fmt.Errorf("get exchange rate: %w", err)
		}
	}

	sourceAmountDec := decimal.NewFromInt(input.Source.Amount)
	convertedAmountDec := sourceAmountDec.Mul(rate).Round(0)

	convertedMoney, err := domain.NewMoney(convertedAmountDec.IntPart(), input.TargetCurrency)
	if err != nil {
		return ConvertCurrencyOutputDTO{}, fmt.Errorf("build converted money: %w", err)
	}

	return ConvertCurrencyOutputDTO{
		Converted: convertedMoney,
		Rate:      rate,
	}, nil
}

func (s *FXService) getRate(ctx context.Context, from, to domain.Currency) (decimal.Decimal, error) {
	return s.cache.Get(ctx, from, to)
}

func (s *FXService) fetchRate(ctx context.Context, from, to domain.Currency) (decimal.Decimal, error) {
	key := string(s.defaultBaseCurr)

	ch := s.sf.DoChan(key, func() (any, error) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rates, err := s.provider.FetchRates(bgCtx, s.defaultBaseCurr)
		if err != nil {
			s.sf.Forget(key)
			return nil, err
		}

		for targetCurr, r := range rates {
			_ = s.cache.Set(bgCtx, s.defaultBaseCurr, targetCurr, r)
		}

		return rates, nil
	})

	var baseRates map[domain.Currency]decimal.Decimal

	select {
	case <-ctx.Done():
		return decimal.Zero, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return decimal.Zero, fmt.Errorf("fetch external rates for base %s: %w", s.defaultBaseCurr, res.Err)
		}

		var ok bool
		baseRates, ok = res.Val.(map[domain.Currency]decimal.Decimal)
		if !ok {
			return decimal.Zero, fmt.Errorf("unexpected rates type from singleflight: %T", res.Val)
		}
	}

	return s.calculateRate(ctx, from, to, baseRates)
}

func (s *FXService) calculateRate(ctx context.Context, from, to domain.Currency, baseRates map[domain.Currency]decimal.Decimal) (decimal.Decimal, error) {
	if from == s.defaultBaseCurr {
		targetRate, exists := baseRates[to]
		if !exists || !targetRate.IsPositive() {
			return decimal.Zero, fmt.Errorf("target currency %s not supported by provider", to)
		}
		return targetRate, nil
	}

	fromRate, fromOk := baseRates[from]
	toRate, toOk := baseRates[to]
	if !fromOk || !toOk || !fromRate.IsPositive() {
		return decimal.Zero, fmt.Errorf("unsupported pair conversion: %s to %s", from, to)
	}

	finalRate := toRate.DivRound(fromRate, 8)
	_ = s.cache.Set(ctx, from, to, finalRate)

	return finalRate, nil
}
