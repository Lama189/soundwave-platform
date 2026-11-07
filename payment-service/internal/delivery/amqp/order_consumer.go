package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderCommandsExchange    = "order.commands"
	PaymentSagaCommandsQueue = "payment-saga-commands"
	RoutingKeyRefundPayment  = "payment.refund"
)

type OrderCommandConsumer struct {
	conn           *amqp.Connection
	paymentService PaymentService
	logger         *slog.Logger
}

func NewOrderCommandConsumer(conn *amqp.Connection, paymentService PaymentService, logger *slog.Logger) *OrderCommandConsumer {
	return &OrderCommandConsumer{
		conn:           conn,
		paymentService: paymentService,
		logger:         logger,
	}
}

func (c *OrderCommandConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare(OrderCommandsExchange, "direct", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", OrderCommandsExchange, err)
	}

	q, err := ch.QueueDeclare(PaymentSagaCommandsQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", PaymentSagaCommandsQueue, err)
	}

	if err := ch.QueueBind(q.Name, RoutingKeyRefundPayment, OrderCommandsExchange, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind queue %s to %s with routing key %s: %w", q.Name, OrderCommandsExchange, RoutingKeyRefundPayment, err)
	}

	msgs, err := ch.Consume(q.Name, "payment-saga-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started payment saga command consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping payment saga consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("payment saga consumer amqp channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *OrderCommandConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	if msg.RoutingKey != RoutingKeyRefundPayment {
		c.logger.Warn("unhandled routing key, acknowledging message", slog.String("routing_key", msg.RoutingKey))
		_ = msg.Ack(false)
		return
	}

	var cmd RefundCommand
	if err := json.Unmarshal(msg.Body, &cmd); err != nil {
		c.logger.Error("failed to unmarshal refund command", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received refund command",
		slog.String("command_id", cmd.CommandID.String()),
		slog.String("saga_id", cmd.SagaID.String()),
		slog.String("order_id", cmd.OrderID.String()),
		slog.String("reason", cmd.Reason),
	)

	input := cmd.ToServiceInput()
	if err := c.paymentService.ProcessRefundCommand(ctx, input); err != nil {
		c.logger.Error("failed to process refund command",
			slog.String("order_id", cmd.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
