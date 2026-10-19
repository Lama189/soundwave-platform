package wallet

import (
	"context"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/Lama189/soundwave-platform/billing-service/internal/service/fx"
	"github.com/Lama189/soundwave-platform/billing-service/internal/uow"
	"github.com/google/uuid"
)

type UnitOfWorkManager interface {
	Do(ctx context.Context, fn func(work uow.UnitOfWork) error) error
}

type CurrencyConverter interface {
	Convert(ctx context.Context, input fx.ConvertCurrency) (fx.ConvertCurrencyOutputDTO, error)
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