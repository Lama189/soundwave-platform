package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

type LedgerRepository struct {
	db DB
}

func NewLedgerRepository(db DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) Append(ctx context.Context, entry *domain.LedgerEntry) error {
	const query = `
		INSERT INTO ledger_entries (
			id,
			wallet_id,
			operation_type,
			amount,
			balance_after,
			currency,
			reference_type,
			reference_id,
			description,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		entry.ID,
		entry.WalletID,
		string(entry.OperationType),
		entry.Amount.Amount,
		entry.BalanceAfter.Amount,
		string(entry.Amount.Currency),
		string(entry.ReferenceType),
		entry.ReferenceID,
		entry.Description,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("append ledger entry: %w", mapError(err))
	}

	return nil
}

func (r *LedgerRepository) GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.LedgerEntry, error) {
	const query = `
		SELECT 
			id,
			wallet_id,
			operation_type,
			amount,
			balance_after,
			currency,
			reference_type,
			reference_id,
			description,
			created_at
		FROM ledger_entries
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get ledger entries for wallet %s: %w", walletID, mapError(err))
	}
	defer rows.Close()

	entries := make([]*domain.LedgerEntry, 0)
	for rows.Next() {
		var m ledgerEntryModel
		if err := rows.Scan(
			&m.ID,
			&m.WalletID,
			&m.OperationType,
			&m.Amount,
			&m.BalanceAfter,
			&m.Currency,
			&m.ReferenceType,
			&m.ReferenceID,
			&m.Description,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan ledger entry record: %w", err)
		}
		entries = append(entries, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ledger entry records: %w", err)
	}

	return entries, nil
}
