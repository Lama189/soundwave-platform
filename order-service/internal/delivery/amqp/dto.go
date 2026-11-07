package amqp

import "github.com/google/uuid"

type PaymentSucceededEvent struct {
	PaymentID   uuid.UUID `json:"payment_id"`
	OrderID     uuid.UUID `json:"order_id"`
	PayerUserID uuid.UUID `json:"payer_user_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	ExternalID  string    `json:"external_id"`
}

type PaymentFailedEvent struct {
	PaymentID     uuid.UUID `json:"payment_id"`
	OrderID       uuid.UUID `json:"order_id"`
	FailureReason string    `json:"failure_reason"`
}

type SagaReplyEvent struct {
	CommandID uuid.UUID `json:"command_id"`
	SagaID    uuid.UUID `json:"saga_id"`
	OrderID   uuid.UUID `json:"order_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}
