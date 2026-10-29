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

	if err := ch.QueueBind(q.Name, RoutingKeyPaymentSuccess, PaymentEventsExchange, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind queue %s to %s: %w", q.Name, PaymentEventsExchange, err)
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
