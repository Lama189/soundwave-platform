package amqp

import (
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
	"github.com/google/uuid"
)

type RefundCommand struct {
	CommandID uuid.UUID `json:"command_id"`
	SagaID    uuid.UUID `json:"saga_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Reason    string    `json:"reason"`
}

func (c *RefundCommand) ToServiceInput() payment.RefundCommandInput {
	return payment.RefundCommandInput{
		CommandID: c.CommandID,
		SagaID:    c.SagaID,
		OrderID:   c.OrderID,
		Reason:    c.Reason,
	}
}
