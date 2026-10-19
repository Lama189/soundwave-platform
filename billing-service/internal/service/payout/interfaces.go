package payout

import (
	"context"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/google/uuid"
)

type UnitOfWorkManager interface {
	Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error
}

type PayoutRepository interface {
	Create(ctx context.Context, payout *domain.Payout) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payout, error)
	GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Payout, error)
	Update(ctx context.Context, payout *domain.Payout) error
}

type WalletRepository interface {
	GetByArtistID(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error)
	GetByArtistIDForUpdate(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error)
	Update(ctx context.Context, wallet *domain.Wallet) error
}

type LedgerRepository interface {
	Append(ctx context.Context, entry *domain.LedgerEntry) error
}

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}
