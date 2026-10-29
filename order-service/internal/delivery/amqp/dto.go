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
