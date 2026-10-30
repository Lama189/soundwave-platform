package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	CatalogEventsExchange = "catalog-events"
)

type OutboxRelay struct {
	conn       *amqp.Connection
	outboxRepo OutboxRepository
	logger     *slog.Logger
	cfg        config.RelayConfig
	workerID   string
}

func NewOutboxRelay(conn *amqp.Connection, outboxRepo OutboxRepository, logger *slog.Logger, cfg config.RelayConfig) *OutboxRelay {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.LockDuration <= 0 {
		cfg.LockDuration = 30 * time.Second
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 7
	}

	workerID, err := os.Hostname()
	if err != nil || workerID == "" {
		workerID = uuid.New().String()
	}

	return &OutboxRelay{
		conn:       conn,
		outboxRepo: outboxRepo,
		logger:     logger,
		cfg:        cfg,
		workerID:   workerID,
	}
}

func (r *OutboxRelay) Start(ctx context.Context) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel for outbox relay: %w", err)
	}

	err = ch.ExchangeDeclare(CatalogEventsExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", CatalogEventsExchange, err)
	}

	r.logger.Info("started outbox relay worker",
		slog.String("worker_id", r.workerID),
		slog.Duration("interval", r.cfg.PollInterval),
		slog.Int("batch_size", r.cfg.BatchSize),
		slog.Duration("lock_duration", r.cfg.LockDuration),
		slog.Int("max_retries", r.cfg.MaxRetries),
		slog.Int("retention_days", r.cfg.RetentionDays),
	)

	go func() {
		defer ch.Close()

		ticker := time.NewTicker(r.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.logger.Info("stopping outbox relay worker")
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

	go func() {
		cleanupTicker := time.NewTicker(1 * time.Hour)
		defer cleanupTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-cleanupTicker.C:
				retention := time.Duration(r.cfg.RetentionDays) * 24 * time.Hour
				deleted, err := r.outboxRepo.CleanupProcessed(ctx, retention)
				if err != nil {
					r.logger.Error("failed to cleanup processed outbox events", slog.String("error", err.Error()))
				} else if deleted > 0 {
					r.logger.Info("cleaned up old processed outbox events", slog.Int64("deleted_count", deleted))
				}
			}
		}
	}()

	return nil
}

func (r *OutboxRelay) processBatch(ctx context.Context, ch *amqp.Channel) {
	events, err := r.outboxRepo.ClaimUnprocessed(ctx, r.cfg.BatchSize, r.cfg.LockDuration, r.workerID)
	if err != nil {
		r.logger.Error("failed to claim unprocessed outbox events", slog.String("error", err.Error()))
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

		err := ch.PublishWithContext(ctx, CatalogEventsExchange, routingKey, false, false, msg)
		if err != nil {
			r.logger.Error("failed to publish outbox event to rabbitmq",
				slog.String("event_id", event.ID.String()),
				slog.String("event_type", routingKey),
				slog.String("error", err.Error()),
			)

			nextRetry := time.Now().UTC().Add(10 * time.Second)
			if recErr := r.outboxRepo.RecordFailure(ctx, event.ID, err.Error(), nextRetry, r.cfg.MaxRetries); recErr != nil {
				r.logger.Error("failed to record outbox event failure",
					slog.String("event_id", event.ID.String()),
					slog.String("error", recErr.Error()),
				)
			}
			break
		}

		successfulIDs = append(successfulIDs, event.ID)
	}

	if len(successfulIDs) > 0 {
		if err := r.outboxRepo.MarkProcessed(ctx, successfulIDs); err != nil {
			r.logger.Error("failed to mark outbox events as processed",
				slog.Int("count", len(successfulIDs)),
				slog.String("error", err.Error()),
			)
			return
		}

		r.logger.Info("published outbox events to rabbitmq", slog.Int("count", len(successfulIDs)))
	}
}
