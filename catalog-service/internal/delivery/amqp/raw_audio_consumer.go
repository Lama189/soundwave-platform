package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	amqp "github.com/rabbitmq/amqp091-go"
)

type S3EventNotification struct {
	EventType string `json:"EventName"`
	Key       string `json:"Key"`
	Records   []struct {
		EventName string `json:"eventName"`
		S3        struct {
			Object struct {
				Key string `json:"key"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

type RawAudioConsumer struct {
	conn         *amqp.Connection
	trackService TrackService
	logger       *slog.Logger
}

func NewRawAudioConsumer(conn *amqp.Connection, trackService TrackService, logger *slog.Logger) *RawAudioConsumer {
	return &RawAudioConsumer{
		conn:         conn,
		trackService: trackService,
		logger:       logger,
	}
}

func (c *RawAudioConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare("minio-events", "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare minio-events exchange: %w", err)
	}

	q, err := ch.QueueDeclare("raw-audio-uploaded", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare raw-audio-uploaded queue: %w", err)
	}

	err = ch.QueueBind(q.Name, "tracks.raw.uploaded", "minio-events", false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind raw-audio-uploaded queue: %w", err)
	}

	msgs, err := ch.Consume(q.Name, "catalog-raw-audio-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started raw audio amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping raw audio consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("raw audio consumer channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *RawAudioConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event S3EventNotification
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Warn("failed to unmarshal s3 event json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	objectKey := event.Key
	if objectKey == "" && len(event.Records) > 0 {
		objectKey = event.Records[0].S3.Object.Key
	}

	if objectKey == "" {
		c.logger.Warn("empty object key in s3 notification")
		_ = msg.Ack(false)
		return
	}

	fileName := filepath.Base(objectKey)
	trackIDStr := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	trackID, err := uuid.Parse(trackIDStr)
	if err != nil {
		c.logger.Warn("failed to parse track uuid from key",
			slog.String("key", objectKey),
			slog.String("error", err.Error()),
		)
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received raw audio upload notification", slog.String("track_id", trackID.String()))

	if err := c.trackService.MarkProcessing(ctx, trackID); err != nil {
		c.logger.Error("failed to mark track as processing",
			slog.String("track_id", trackID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
