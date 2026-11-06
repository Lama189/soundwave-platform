package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderCommandsExchange        = "order.commands"
	CatalogSagaCommandsQueue     = "catalog-saga-commands"
	RoutingKeyGrantEntitlements  = "catalog.grant_entitlements"
	RoutingKeyRevokeEntitlements = "catalog.revoke_entitlements"
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

	err = ch.ExchangeDeclare(OrderCommandsExchange, "direct", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s exchange: %w", OrderCommandsExchange, err)
	}

	q, err := ch.QueueDeclare(CatalogSagaCommandsQueue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("declare %s queue: %w", CatalogSagaCommandsQueue, err)
	}

	routingKeys := []string{RoutingKeyGrantEntitlements, RoutingKeyRevokeEntitlements}
	for _, rk := range routingKeys {
		if err := ch.QueueBind(q.Name, rk, OrderCommandsExchange, false, nil); err != nil {
			_ = ch.Close()
			return fmt.Errorf("bind queue %s to %s with routing key %s: %w", q.Name, OrderCommandsExchange, rk, err)
		}
	}

	msgs, err := ch.Consume(q.Name, "catalog-saga-consumer", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume messages: %w", err)
	}

	c.logger.Info("started catalog saga command consumer", slog.String("queue", q.Name))

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("stopping catalog saga consumer due to context cancellation")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn("catalog saga consumer amqp channel closed")
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *OrderPaidConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	switch msg.RoutingKey {
	case RoutingKeyGrantEntitlements:
		c.handleGrantEntitlements(ctx, msg)
	case RoutingKeyRevokeEntitlements:
		c.handleRevokeEntitlements(ctx, msg)
	default:
		c.logger.Warn("unhandled routing key, acknowledging message", slog.String("routing_key", msg.RoutingKey))
		_ = msg.Ack(false)
	}
}

func (c *OrderPaidConsumer) handleGrantEntitlements(ctx context.Context, msg amqp.Delivery) {
	var cmd GrantEntitlementsCommand
	if err := json.Unmarshal(msg.Body, &cmd); err != nil {
		c.logger.Error("failed to unmarshal grant entitlements command", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received grant entitlements command",
		slog.String("command_id", cmd.CommandID.String()),
		slog.String("saga_id", cmd.SagaID.String()),
		slog.String("order_id", cmd.OrderID.String()),
	)

	input := cmd.ToServiceInput()
	if err := c.purchaseService.ProcessGrantEntitlements(ctx, input); err != nil {
		c.logger.Error("failed to process grant entitlements",
			slog.String("order_id", cmd.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}

func (c *OrderPaidConsumer) handleRevokeEntitlements(ctx context.Context, msg amqp.Delivery) {
	var cmd RevokeEntitlementsCommand
	if err := json.Unmarshal(msg.Body, &cmd); err != nil {
		c.logger.Error("failed to unmarshal revoke entitlements command", slog.String("error", err.Error()))
		_ = msg.Ack(false)
		return
	}

	c.logger.Info("received revoke entitlements command",
		slog.String("command_id", cmd.CommandID.String()),
		slog.String("saga_id", cmd.SagaID.String()),
		slog.String("order_id", cmd.OrderID.String()),
	)

	input := cmd.ToServiceInput()
	if err := c.purchaseService.RevokePurchases(ctx, input); err != nil {
		c.logger.Error("failed to revoke purchases",
			slog.String("order_id", cmd.OrderID.String()),
			slog.String("error", err.Error()),
		)
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}
