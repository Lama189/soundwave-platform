package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type RateCacheRepository struct {
	client  *redis.Client
	rateTTL time.Duration
}

func NewRateCacheRepository(client *redis.Client, rateTTL time.Duration) *RateCacheRepository {
	return &RateCacheRepository{
		client:  client,
		rateTTL: rateTTL,
	}
}

func (r *RateCacheRepository) Get(ctx context.Context, from, to domain.Currency) (decimal.Decimal, error) {
	key := fmt.Sprintf("fx:rate:%s:%s", from, to)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return decimal.Zero, ErrCacheMiss
		}
		return decimal.Zero, fmt.Errorf("corrupted decimal rate in cache for %s: %w", key, err)
	}

	rate, err := decimal.NewFromString(val)
	if err != nil {
		return decimal.Zero, fmt.Errorf("corrupted cache entry for %s: %w", key, err)
	}

	if !rate.IsPositive() {
		return decimal.Zero, fmt.Errorf("invalid non-positive rate %s for key %s", rate, key)
	}

	return rate, nil
}

func (r *RateCacheRepository) Set(ctx context.Context, from, to domain.Currency, rate decimal.Decimal) error {
	if !rate.IsPositive() {
		return fmt.Errorf("cannot cache non-positive rate: %s", rate)
	}

	key := fmt.Sprintf("fx:rate:%s:%s", from, to)

	if err := r.client.Set(ctx, key, rate.String(), r.rateTTL).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}

	return nil
}
