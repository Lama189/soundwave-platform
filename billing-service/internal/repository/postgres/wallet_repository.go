package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

const baseWalletSelectQuery = `
    SELECT
        id,
        artist_id,
        currency,
        balance,
        locked_balance,
        created_at,
        updated_at
    FROM wallets
    WHERE artist_id = $1
`

type WalletRepository struct {
	db DB
}

func NewWalletRepository(db DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	const query = `
		INSERT INTO wallets (
			id,
			artist_id,
			currency,
			balance,
			locked_balance,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		wallet.ID,
		wallet.ArtistID,
		string(wallet.Currency),
		wallet.Balance.Amount,
		wallet.LockedBalance.Amount,
		wallet.CreatedAt,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create wallet %s for artist %s: %w", wallet.ID, wallet.ArtistID, mapError(err))
	}

	return nil
}

func (r *WalletRepository) GetByArtistID(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error) {
	return r.getByArtistID(ctx, artistID, false)
}

func (r *WalletRepository) GetByArtistIDForUpdate(ctx context.Context, artistID uuid.UUID) (*domain.Wallet, error) {
	return r.getByArtistID(ctx, artistID, true)
}

func (r *WalletRepository) Update(ctx context.Context, wallet *domain.Wallet) error {
	const query = `
		UPDATE wallets
		SET
			balance = $2,
			locked_balance = $3,
			updated_at = $4
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		wallet.ID,
		wallet.Balance.Amount,
		wallet.LockedBalance.Amount,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update wallet %s: %w", wallet.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("update wallet %s: %w", wallet.ID, domain.ErrNotFound)
	}

	return nil
}

func (r *WalletRepository) getByArtistID(ctx context.Context, artistID uuid.UUID, forUpdate bool) (*domain.Wallet, error) {
	query := baseWalletSelectQuery
	if forUpdate {
		query += "\nFOR UPDATE;"
	} else {
		query += ";"
	}

	var m walletModel
	err := r.db.QueryRow(ctx, query, artistID).Scan(
		&m.ID,
		&m.ArtistID,
		&m.Currency,
		&m.Balance,
		&m.LockedBalance,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get wallet by artist id %s: %w", artistID, mapError(err))
	}

	return m.toDomain(), nil
}
