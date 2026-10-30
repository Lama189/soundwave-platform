package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	PaymentEventsExchange    = "payment-events"
	OrderPaymentEventsQueue  = "order-payment-events"
	RoutingKeyPaymentSuccess = "payment.succeeded"
	RoutingKeyPaymentFailed  = "payment.failed"
)

type PaymentConsumer struct {
	conn         *amqp.Connection
	orderService OrderService
	logger       *slog.Logger
}

func NewPaymentConsumer(conn *amqp.Connection, orderService OrderService, logger *slog.Logger) *PaymentConsumer {
	return &PaymentConsumer{
		conn:         conn,
		orderService: orderService,
		logger:       logger,
	}
}

func (c *PaymentConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare(PaymentEventsExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", PaymentEventsExchange, err)
	}

	q, err := ch.QueueDeclare(OrderPaymentEventsQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", OrderPaymentEventsQueue, err)
	}

	routingKeys := []string{RoutingKeyPaymentSuccess, RoutingKeyPaymentFailed}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(q.Name, rk, PaymentEventsExchange, false, nil); err != nil {
			_ = ch.Close()
			return fmt.Errorf("bind queue %s to %s with key %s: %w", q.Name, PaymentEventsExchange, rk, err)
		}
	}

	msgs, err := ch.Consume(q.Name, "order-payment-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started payment amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping payment consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("payment consumer amqp channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *PaymentConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	switch msg.RoutingKey {
	case RoutingKeyPaymentSuccess:
		c.handlePaymentSucceeded(ctx, msg)
	case RoutingKeyPaymentFailed:
		c.handlePaymentFailed(ctx, msg)
	default:
		c.logger.Warn("unhandled routing key, acknowledging message", slog.String("routing_key", msg.RoutingKey))
		_ = msg.Ack(false)
	}
}

func (c *PaymentConsumer) handlePaymentSucceeded(ctx context.Context, msg amqp.Delivery) {
	var event PaymentSucceededEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal payment.succeeded json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received payment.succeeded event",
		slog.String("order_id", event.OrderID.String()),
		slog.String("payment_id", event.PaymentID.String()),
		slog.Int64("amount", event.Amount),
	)

	_, err := c.orderService.PayOrder(ctx, order.PayOrderDTO{
		OrderID:   event.OrderID,
		PaymentID: event.PaymentID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrOrderAlreadyPaid) {
			c.logger.Warn("order is already paid, skipping duplicate event", slog.String("order_id", event.OrderID.String()))
			_ = msg.Ack(false)
			return
		}

		c.logger.Error("failed to pay order via event",
			slog.String("order_id", event.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}

func (c *PaymentConsumer) handlePaymentFailed(ctx context.Context, msg amqp.Delivery) {
	var event PaymentFailedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal payment.failed json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received payment.failed event",
		slog.String("order_id", event.OrderID.String()),
		slog.String("payment_id", event.PaymentID.String()),
		slog.String("failure_reason", event.FailureReason),
	)

	_, err := c.orderService.FailOrder(ctx, order.FailOrderDTO{
		OrderID:       event.OrderID,
		PaymentID:     event.PaymentID,
		FailureReason: event.FailureReason,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			c.logger.Warn("cannot fail order due to conflict, skipping event",
				slog.String("order_id", event.OrderID.String()),
				slog.String("error", err.Error()),
			)
			_ = msg.Ack(false)
			return
		}

		c.logger.Error("failed to fail order via event",
			slog.String("order_id", event.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
