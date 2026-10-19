package postgres

import (
	"time"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
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
		domain.Money{
			Amount:   m.TotalAmount,
			Currency: domain.Currency(m.Currency),
		},
		m.PaymentID,
		m.FailureReason,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type orderItemModel struct {
	ID          uuid.UUID `db:"id"`
	OrderID     uuid.UUID `db:"order_id"`
	ItemType    string    `db:"item_type"`
	ItemID      uuid.UUID `db:"item_id"`
	ArtistID    uuid.UUID `db:"artist_id"`
	Price       int64     `db:"price"`
	PlatformFee int64     `db:"platform_fee"`
	ArtistShare int64     `db:"artist_share"`
	Currency    string    `db:"currency"`
}

func (m *orderItemModel) toDomain() domain.OrderItem {
	currency := domain.Currency(m.Currency)
	return domain.RestoreOrderItem(
		m.ID,
		m.OrderID,
		domain.ItemType(m.ItemType),
		m.ItemID,
		m.ArtistID,
		domain.Money{Amount: m.Price, Currency: currency},
		domain.Money{Amount: m.PlatformFee, Currency: currency},
		domain.Money{Amount: m.ArtistShare, Currency: currency},
	)
}

type walletModel struct {
	ID            uuid.UUID `db:"id"`
	ArtistID      uuid.UUID `db:"artist_id"`
	Currency      string    `db:"currency"`
	Balance       int64     `db:"balance"`
	LockedBalance int64     `db:"locked_balance"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func (m *walletModel) toDomain() *domain.Wallet {
	currency := domain.Currency(m.Currency)
	return domain.RestoreWallet(
		m.ID,
		m.ArtistID,
		currency,
		domain.Money{Amount: m.Balance, Currency: currency},
		domain.Money{Amount: m.LockedBalance, Currency: currency},
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type ledgerEntryModel struct {
	ID            uuid.UUID `db:"id"`
	WalletID      uuid.UUID `db:"wallet_id"`
	OperationType string    `db:"operation_type"`
	Amount        int64     `db:"amount"`
	BalanceAfter  int64     `db:"balance_after"`
	Currency      string    `db:"currency"`
	ReferenceType string    `db:"reference_type"`
	ReferenceID   uuid.UUID `db:"reference_id"`
	Description   string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
}

func (m *ledgerEntryModel) toDomain() *domain.LedgerEntry {
	currency := domain.Currency(m.Currency)
	return domain.RestoreLedgerEntry(
		m.ID,
		m.WalletID,
		domain.LedgerOperationType(m.OperationType),
		domain.Money{Amount: m.Amount, Currency: currency},
		domain.Money{Amount: m.BalanceAfter, Currency: currency},
		domain.ReferenceType(m.ReferenceType),
		m.ReferenceID,
		m.Description,
		m.CreatedAt,
	)
}

type payoutModel struct {
	ID              uuid.UUID  `db:"id"`
	ArtistID        uuid.UUID  `db:"artist_id"`
	WalletID        uuid.UUID  `db:"wallet_id"`
	Amount          int64      `db:"amount"`
	Currency        string     `db:"currency"`
	DestinationCard string     `db:"destination_card"`
	Status          string     `db:"status"`
	FailureReason   *string    `db:"failure_reason"`
	RequestedAt     time.Time  `db:"requested_at"`
	ProcessedAt     *time.Time `db:"processed_at"`
}

func (m *payoutModel) toDomain() *domain.Payout {
	currency := domain.Currency(m.Currency)
	return domain.RestorePayout(
		m.ID,
		m.ArtistID,
		m.WalletID,
		domain.Money{Amount: m.Amount, Currency: currency},
		m.DestinationCard,
		domain.PayoutStatus(m.Status),
		m.FailureReason,
		m.RequestedAt,
		m.ProcessedAt,
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

