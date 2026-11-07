package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/config"
)

type SagaJanitor struct {
	orderRepo    OrderRepository
	orderService OrderTimeoutService
	logger       *slog.Logger
	interval     time.Duration
	batchSize    int
}

func NewSagaJanitor(orderRepo OrderRepository, orderService OrderTimeoutService, logger *slog.Logger, cfg config.SagaConfig) *SagaJanitor {
	interval := cfg.JanitorInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	batchSize := cfg.JanitorBatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	return &SagaJanitor{
		orderRepo:    orderRepo,
		orderService: orderService,
		logger:       logger,
		interval:     interval,
		batchSize:    batchSize,
	}
}

func (j *SagaJanitor) Start(ctx context.Context) error {
	j.logger.Info("started saga janitor worker",
		slog.Duration("interval", j.interval),
		slog.Int("batch_size", j.batchSize),
	)

	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("stopping saga janitor worker")
				return
			case <-ticker.C:
				j.processTimedOutOrders(ctx)
			}
		}
	}()

	return nil
}

func (j *SagaJanitor) processTimedOutOrders(ctx context.Context) {
	now := time.Now().UTC()
	orders, err := j.orderRepo.GetTimedOut(ctx, now, j.batchSize)
	if err != nil {
		j.logger.Error("failed to query timed out saga orders", slog.String("error", err.Error()))
		return
	}

	if len(orders) == 0 {
		return
	}

	j.logger.Info("found timed out saga orders", slog.Int("count", len(orders)))

	for _, o := range orders {
		if err := j.orderService.HandleTimeout(ctx, o.ID); err != nil {
			j.logger.Error("failed to handle saga timeout for order",
				slog.String("order_id", o.ID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		j.logger.Info("handled saga timeout for order",
			slog.String("order_id", o.ID.String()),
			slog.String("saga_step", string(o.Saga.Step)),
		)
	}
}
