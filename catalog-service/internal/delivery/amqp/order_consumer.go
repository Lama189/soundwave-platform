package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderEventsExchange     = "order-events"
	CatalogOrderEventsQueue = "catalog-order-events"
	RoutingKeyOrderPaid     = "order.paid"
)

type OrderPaidConsumer struct {
	conn            *amqp.Connection
	purchaseService UserPurchaseService
	logger          *slog.Logger
}

func NewOrderPaidConsumer(conn *amqp.Connection, purchaseService UserPurchaseService, logger *slog.Logger) *OrderPaidConsumer {
	return &OrderPaidConsumer{
		conn:            conn,
		purchaseService: purchaseService,
		logger:          logger,
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

	q, err := ch.QueueDeclare(CatalogOrderEventsQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", CatalogOrderEventsQueue, err)
	}

	if err := ch.QueueBind(q.Name, RoutingKeyOrderPaid, OrderEventsExchange, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("bind queue %s to %s with routing key %s: %w", q.Name, OrderEventsExchange, RoutingKeyOrderPaid, err)
	}

	msgs, err := ch.Consume(q.Name, "catalog-order-paid-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started catalog order paid amqp consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping catalog order paid consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("catalog order paid consumer amqp channel closed")
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
		c.logger.Error(
			"failed to unmarshal order.paid json",
			slog.String("error", err.Error()),
		)
		_ = msg.Ack(false)
		return
	}

	c.logger.Info(
		"received order.paid event in catalog",
		slog.String("order_id", event.OrderID.String()),
		slog.String("buyer_user_id", event.BuyerUserID.String()),
		slog.Int("items_count", len(event.Items)),
	)

	input := event.toServiceInput()
	err := c.purchaseService.ProcessPaidOrder(ctx, input)
	if err != nil {
		c.logger.Error(
			"failed to process paid order entitlements",
			slog.String("order_id", event.OrderID.String()),
			slog.String("error", err.Error()),
		)

		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
