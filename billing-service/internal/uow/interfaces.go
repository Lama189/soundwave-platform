package uow

import (
	"context"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

type UnitOfWork interface {
	Orders() OrderRepository
	Wallets() WalletRepository
	Ledger() LedgerRepository
	Payouts() PayoutRepository
	Outbox() OutboxRepository
}

type Manager interface {
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error
}

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.Wallet) error
	GetByArtistID(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error)
	GetByArtistIDForUpdate(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error)
	Update(ctx context.Context, wallet *domain.Wallet) error
}

type LedgerRepository interface {
	Append(ctx context.Context, entry *domain.LedgerEntry) error
	GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.LedgerEntry, error)
}

type PayoutRepository interface {
	Create(ctx context.Context, payout *domain.Payout) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payout, error)
	GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Payout, error)
	Update(ctx context.Context, payout *domain.Payout) error
}

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}
