package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/config"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PaymentEventsExchange = "payment-events"
)

type OutboxRelay struct {
	conn       *amqp.Connection
	outboxRepo OutboxRepository
	logger     *slog.Logger
	cfg        config.RelayConfig
}

func NewOutboxRelay(conn *amqp.Connection, outboxRepo OutboxRepository, logger *slog.Logger, cfg config.RelayConfig) *OutboxRelay {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}

	return &OutboxRelay{
		conn:       conn,
		outboxRepo: outboxRepo,
		logger:     logger,
		cfg:        cfg,
	}
}

func (r *OutboxRelay) Start(ctx context.Context) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel for payment outbox relay: %w", err)
	}

	err = ch.ExchangeDeclare(PaymentEventsExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", PaymentEventsExchange, err)
	}

	r.logger.Info("started payment outbox relay worker",
		slog.Duration("interval", r.cfg.PollInterval),
		slog.Int("batch_size", r.cfg.BatchSize),
	)

	go func() {
		defer ch.Close()

		ticker := time.NewTicker(r.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.logger.Info("stopping payment outbox relay worker")
				return
			case <-ticker.C:
				if ch.IsClosed() {
					newCh, err := r.conn.Channel()
					if err != nil {
						r.logger.Error("failed to reopen closed amqp channel", slog.String("error", err.Error()))
						continue
					}
					ch = newCh
				}
				r.processBatch(ctx, ch)
			}
		}
	}()

	return nil
}

func (r *OutboxRelay) processBatch(ctx context.Context, ch *amqp.Channel) {
	events, err := r.outboxRepo.FetchUnprocessed(ctx, r.cfg.BatchSize)
	if err != nil {
		r.logger.Error("failed to fetch unprocessed payment outbox events", slog.String("error", err.Error()))
		return
	}

	if len(events) == 0 {
		return
	}

	successfulIDs := make([]uuid.UUID, 0, len(events))

	for _, event := range events {
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			MessageId:    event.ID.String(),
			Type:         string(event.EventType),
			Timestamp:    event.CreatedAt,
			Body:         event.Payload,
		}

		routingKey := string(event.EventType)

		err := ch.PublishWithContext(ctx, PaymentEventsExchange, routingKey, false, false, msg)
		if err != nil {
			r.logger.Error("failed to publish payment outbox event to rabbitmq",
				slog.String("event_id", event.ID.String()),
				slog.String("event_type", routingKey),
				slog.String("error", err.Error()),
			)
			break
		}

		successfulIDs = append(successfulIDs, event.ID)
	}

	if len(successfulIDs) > 0 {
		if err := r.outboxRepo.MarkProcessed(ctx, successfulIDs); err != nil {
			r.logger.Error("failed to mark payment outbox events as processed",
				slog.Int("count", len(successfulIDs)),
				slog.String("error", err.Error()),
			)
			return
		}

		r.logger.Info("published payment outbox events to rabbitmq", slog.Int("count", len(successfulIDs)))
	}
}
