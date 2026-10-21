package postgres

import (
	"time"

	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/google/uuid"
)

type orderModel struct {
	ID            uuid.UUID  `db:"id"`
	BuyerUserID   uuid.UUID  `db:"buyer_user_id"`
	Status        string     `db:"status"`
	TotalAmount   int64      `db:"total_amount"`
	Currency      string     `db:"currency"`
	PaymentID     *uuid.UUID `db:"payment_id"`
	FailureReason *string    `db:"failure_reason"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

func (m *orderModel) toDomain(items []domain.OrderItem) *domain.Order {
	return domain.RestoreOrder(
		m.ID,
		m.BuyerUserID,
		domain.OrderStatus(m.Status),
		items,
		m.TotalAmount,
		domain.Currency(m.Currency),
		m.PaymentID,
		m.FailureReason,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type orderItemModel struct {
	ID       uuid.UUID `db:"id"`
	OrderID  uuid.UUID `db:"order_id"`
	ItemType string    `db:"item_type"`
	ItemID   uuid.UUID `db:"item_id"`
	ArtistID uuid.UUID `db:"artist_id"`
	Price    int64     `db:"price"`
	Currency string    `db:"currency"`
}

func (m *orderItemModel) toDomain() domain.OrderItem {
	return domain.RestoreOrderItem(
		m.ID,
		m.OrderID,
		domain.ItemType(m.ItemType),
		m.ItemID,
		m.ArtistID,
		m.Price,
		domain.Currency(m.Currency),
	)
}

type eventModel struct {
	ID            uuid.UUID  `db:"id"`
	AggregateType string     `db:"aggregate_type"`
	AggregateID   uuid.UUID  `db:"aggregate_id"`
	EventType     string     `db:"event_type"`
	Payload       []byte     `db:"payload"`
	CreatedAt     time.Time  `db:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at"`
}

func (m *eventModel) toDomain() *domain.Event {
	return domain.RestoreEvent(
		m.ID,
		domain.AggregateType(m.AggregateType),
		m.AggregateID,
		domain.EventType(m.EventType),
		m.Payload,
		m.CreatedAt,
		m.ProcessedAt,
	)
}
