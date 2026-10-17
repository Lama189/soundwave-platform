package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
)

type PayoutRepository struct {
	db DB
}

func NewPayoutRepository(db DB) *PayoutRepository {
	return &PayoutRepository{db: db}
}

func (r *PayoutRepository) Create(ctx context.Context, payout *domain.Payout) error {
	const query = `
		INSERT INTO payouts (
			id,
			artist_id,
			wallet_id,
			amount,
			currency,
			destination_card,
			status,
			failure_reason,
			requested_at,
			processed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		payout.ID,
		payout.ArtistID,
		payout.WalletID,
		payout.Amount.Amount,
		string(payout.Amount.Currency),
		payout.DestinationCard,
		string(payout.Status),
		payout.FailureReason,
		payout.RequestedAt,
		payout.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("create payout %s: %w", payout.ID, mapError(err))
	}

	return nil
}

func (r *PayoutRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payout, error) {
	const query = `
		SELECT 
			id,
			artist_id,
			wallet_id,
			amount,
			currency,
			destination_card,
			status,
			failure_reason,
			requested_at,
			processed_at
		FROM payouts
		WHERE id = $1;
	`

	var m payoutModel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.ArtistID,
		&m.WalletID,
		&m.Amount,
		&m.Currency,
		&m.DestinationCard,
		&m.Status,
		&m.FailureReason,
		&m.RequestedAt,
		&m.ProcessedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get payout by id %s: %w", id, mapError(err))
	}

	return m.toDomain(), nil
}

func (r *PayoutRepository) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Payout, error) {
	const query = `
		SELECT 
			id,
			artist_id,
			wallet_id,
			amount,
			currency,
			destination_card,
			status,
			failure_reason,
			requested_at,
			processed_at
		FROM payouts
		WHERE artist_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, artistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get payouts for artist %s: %w", artistID, mapError(err))
	}
	defer rows.Close()

	payouts := make([]*domain.Payout, 0)
	for rows.Next() {
		var m payoutModel
		if err := rows.Scan(
			&m.ID,
			&m.ArtistID,
			&m.WalletID,
			&m.Amount,
			&m.Currency,
			&m.DestinationCard,
			&m.Status,
			&m.FailureReason,
			&m.RequestedAt,
			&m.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payout record: %w", err)
		}
		payouts = append(payouts, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payout records: %w", err)
	}

	return payouts, nil
}

func (r *PayoutRepository) Update(ctx context.Context, payout *domain.Payout) error {
	const query = `
		UPDATE payouts
		SET 
			status = $2,
			failure_reason = $3,
			processed_at = $4
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		payout.ID,
		string(payout.Status),
		payout.FailureReason,
		payout.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("update payout %s: %w", payout.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("update payout %s: %w", payout.ID, domain.ErrNotFound)
	}

	return nil
}
