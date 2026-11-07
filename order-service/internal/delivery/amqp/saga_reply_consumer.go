package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	SagaRepliesExchange = "order.saga.replies"
	SagaRepliesQueue    = "order-saga-replies"
	SagaRepliesTopicKey = "saga.reply.#"

	RoutingKeyCatalogGranted      = "saga.reply.catalog.entitlements_granted"
	RoutingKeyCatalogGrantFailed  = "saga.reply.catalog.entitlements_grant_failed"
	RoutingKeyCatalogRevoked      = "saga.reply.catalog.entitlements_revoked"
	RoutingKeyBillingSettled      = "saga.reply.billing.settled"
	RoutingKeyBillingFailed       = "saga.reply.billing.failed"
	RoutingKeyPaymentRefunded     = "saga.reply.payment.refunded"
	RoutingKeyPaymentRefundFailed = "saga.reply.payment.refund_failed"
)

type SagaReplyConsumer struct {
	conn         *amqp.Connection
	orderService OrderService
	logger       *slog.Logger
}

func NewSagaReplyConsumer(conn *amqp.Connection, orderService OrderService, logger *slog.Logger) *SagaReplyConsumer {
	return &SagaReplyConsumer{
		conn:         conn,
		orderService: orderService,
		logger:       logger,
	}
}

func (c *SagaReplyConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare(SagaRepliesExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", SagaRepliesExchange, err)
	}

	q, err := ch.QueueDeclare(SagaRepliesQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", SagaRepliesQueue, err)
	}

	if err := ch.QueueBind(q.Name, SagaRepliesTopicKey, SagaRepliesExchange, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind queue %s to %s with key %s: %w", q.Name, SagaRepliesExchange, SagaRepliesTopicKey, err)
	}

	msgs, err := ch.Consume(q.Name, "order-saga-reply-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started saga reply amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping saga reply consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("saga reply consumer amqp channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *SagaReplyConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event SagaReplyEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal saga reply json",
			slog.String("routing_key", msg.RoutingKey),
			slog.String("error", err.Error()),
		)
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received saga reply",
		slog.String("routing_key", msg.RoutingKey),
		slog.String("order_id", event.OrderID.String()),
		slog.Bool("success", event.Success),
	)

	var err error
	switch msg.RoutingKey {
	case RoutingKeyCatalogGranted:
		err = c.orderService.OnCatalogEntitlementsGranted(ctx, event.OrderID)
	case RoutingKeyCatalogGrantFailed:
		err = c.orderService.OnCatalogEntitlementsFailed(ctx, event.OrderID, event.Error)
	case RoutingKeyBillingSettled:
		err = c.orderService.OnBillingSettled(ctx, event.OrderID)
	case RoutingKeyBillingFailed:
		err = c.orderService.OnBillingFailed(ctx, event.OrderID, event.Error)
	case RoutingKeyCatalogRevoked:
		err = c.orderService.OnCatalogEntitlementsRevoked(ctx, event.OrderID)
	case RoutingKeyPaymentRefunded:
		err = c.orderService.OnPaymentRefunded(ctx, event.OrderID)
	case RoutingKeyPaymentRefundFailed:
		c.logger.Error("saga payment refund reply reported failure",
			slog.String("order_id", event.OrderID.String()),
			slog.String("error", event.Error),
		)
		_ = msg.Ack(false)
		return
	default:
		c.logger.Warn("unhandled saga reply routing key, acknowledging message", slog.String("routing_key", msg.RoutingKey))
		_ = msg.Ack(false)
		return
	}

	if err != nil {
		if errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrNotFound) {
			c.logger.Warn("saga reply conflict/not found, acknowledging duplicate or obsolete message",
				slog.String("order_id", event.OrderID.String()),
				slog.String("routing_key", msg.RoutingKey),
				slog.String("error", err.Error()),
			)
			_ = msg.Ack(false)
			return
		}

		c.logger.Error("failed to process saga reply",
			slog.String("order_id", event.OrderID.String()),
			slog.String("routing_key", msg.RoutingKey),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
