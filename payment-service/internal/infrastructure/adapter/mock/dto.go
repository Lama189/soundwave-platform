package mock

type webhookPayload struct {
	EventID       string `json:"event_id"`
	PaymentID     string `json:"payment_id"`
	EventType     string `json:"event_type"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	FailureReason string `json:"failure_reason"`
}
