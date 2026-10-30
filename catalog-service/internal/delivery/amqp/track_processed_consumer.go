package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	CatalogEventsExchange    = "catalog-events"
	TrackProcessedQueue      = "catalog-track-processed"
	RoutingKeyTrackProcessed = "track.processed"
	RoutingKeyTrackFailed    = "track.processing_failed"
)

type TrackProcessedEvent struct {
	TrackID         uuid.UUID `json:"track_id"`
	DurationSeconds int64     `json:"duration_seconds"`
	FileKey         string    `json:"file_key"`
	PreviewKey      string    `json:"preview_key"`
	CoverKey        *string   `json:"cover_key"`
}

type TrackProcessingFailedEvent struct {
	TrackID      uuid.UUID `json:"track_id"`
	ErrorMessage string    `json:"error_message"`
}

type TrackProcessedConsumer struct {
	conn         *amqp.Connection
	trackService TrackService
	logger       *slog.Logger
}

func NewTrackProcessedConsumer(conn *amqp.Connection, trackService TrackService, logger *slog.Logger) *TrackProcessedConsumer {
	return &TrackProcessedConsumer{
		conn:         conn,
		trackService: trackService,
		logger:       logger,
	}
}

func (c *TrackProcessedConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare(CatalogEventsExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", CatalogEventsExchange, err)
	}

	q, err := ch.QueueDeclare(TrackProcessedQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", TrackProcessedQueue, err)
	}

	routingKeys := []string{RoutingKeyTrackProcessed, RoutingKeyTrackFailed}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(q.Name, rk, CatalogEventsExchange, false, nil); err != nil {
			_ = ch.Close()
			return fmt.Errorf("bind queue %s to %s with routing key %s: %w", q.Name, CatalogEventsExchange, rk, err)
		}
	}

	msgs, err := ch.Consume(q.Name, "catalog-track-processed-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started track processed amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping track processed consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("track processed consumer channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *TrackProcessedConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	switch msg.RoutingKey {
	case RoutingKeyTrackProcessed:
		c.handleTrackProcessed(ctx, msg)
	case RoutingKeyTrackFailed:
		c.handleTrackFailed(ctx, msg)
	default:
		c.logger.Warn("unknown routing key in track processed consumer", slog.String("routing_key", msg.RoutingKey))
		_ = msg.Ack(false)
	}
}

func (c *TrackProcessedConsumer) handleTrackProcessed(ctx context.Context, msg amqp.Delivery) {
	var event TrackProcessedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal track.processed json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	var coverKey string
	if event.CoverKey != nil {
		coverKey = *event.CoverKey
	}

	dto := track.CompleteProcessingDTO{
		CoverKey:   coverKey,
		FileKey:    event.FileKey,
		PreviewKey: event.PreviewKey,
		Duration:   time.Duration(event.DurationSeconds) * time.Second,
	}

	c.logger.Info("received track.processed event",
		slog.String("track_id", event.TrackID.String()),
		slog.Int64("duration_seconds", event.DurationSeconds),
		slog.String("file_key", event.FileKey),
	)

	_, err := c.trackService.CompleteProcessing(ctx, event.TrackID, dto)
	if err != nil {
		c.logger.Error("failed to complete track processing",
			slog.String("track_id", event.TrackID.String()),
			slog.String("error", err.Error()),
		)

		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}

func (c *TrackProcessedConsumer) handleTrackFailed(ctx context.Context, msg amqp.Delivery) {
	var event TrackProcessingFailedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal track.processing_failed json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Warn("received track.processing_failed event",
		slog.String("track_id", event.TrackID.String()),
		slog.String("reason", event.ErrorMessage),
	)

	if err := c.trackService.FailProcessing(ctx, event.TrackID); err != nil {
		c.logger.Error("failed to mark track as failed",
			slog.String("track_id", event.TrackID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
