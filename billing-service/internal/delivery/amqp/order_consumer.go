package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderEventsExchange     = "order-events"
	BillingOrderEventsQueue = "billing-order-events"
	RoutingKeyOrderPaid     = "order.paid"
)

type OrderPaidConsumer struct {
	conn              *amqp.Connection
	settlementService SettlementService
	logger            *slog.Logger
}

func NewOrderPaidConsumer(conn *amqp.Connection, settlementService SettlementService, logger *slog.Logger) *OrderPaidConsumer {
	return &OrderPaidConsumer{
		conn:              conn,
		settlementService: settlementService,
		logger:            logger,
	}
}

func (c *OrderPaidConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open amqp channel: %w", err)
	}

	err = ch.ExchangeDeclare(OrderEventsExchange, "topic", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", OrderEventsExchange, err)
	}

	q, err := ch.QueueDeclare(BillingOrderEventsQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", BillingOrderEventsQueue, err)
	}

	if err := ch.QueueBind(q.Name, RoutingKeyOrderPaid, OrderEventsExchange, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind queue %s to %s with routing key %s: %w", q.Name, OrderEventsExchange, RoutingKeyOrderPaid, err)
	}

	msgs, err := ch.Consume(q.Name, "billing-order-paid-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started billing order paid amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping billing order paid consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("billing order paid consumer amqp channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *OrderPaidConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var event OrderPaidEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal order.paid json", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received order.paid event",
		slog.String("order_id", event.OrderID.String()),
		slog.String("payment_id", event.PaymentID.String()),
		slog.Int("items_count", len(event.Items)),
		slog.Int64("total_amount", event.TotalAmount),
	)

	input := event.toSettlementInput()
	err := c.settlementService.ProcessPaidOrder(ctx, input)
	if err != nil {
		if errors.Is(err, domain.ErrOrderAlreadyProcessed) {
			c.logger.Warn("order is already settled, skipping duplicate event", slog.String("order_id", event.OrderID.String()))
			_ = msg.Ack(false)
			return
		}

		c.logger.Error("failed to settle paid order via event",
			slog.String("order_id", event.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
