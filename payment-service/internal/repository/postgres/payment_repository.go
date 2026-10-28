package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentRepository struct {
	db DB
}

func NewPaymentRepository(db DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	const query = `
		INSERT INTO payments (
			id,
			order_id,
			payer_user_id,
			amount,
			currency,
			status,
			provider,
			external_id,
			client_secret,
			failure_reason,
			idempotency_key,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		payment.ID,
		payment.OrderID,
		payment.PayerUserID,
		payment.Amount,
		string(payment.Currency),
		string(payment.Status),
		string(payment.Provider),
		payment.ExternalID,
		payment.ClientSecret,
		payment.FailureReason,
		payment.IdempotencyKey,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create payment %s: %w", payment.ID, mapError(err))
	}

	return nil
}

const basePaymentSelectQuery = `
	SELECT
		id,
		order_id,
		payer_user_id,
		amount,
		currency,
		status,
		provider,
		external_id,
		client_secret,
		failure_reason,
		idempotency_key,
		created_at,
		updated_at
	FROM payments
`

func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE id = $1;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get payment by id %s: %w", id, err)
	}

	return payment, nil
}

func (r *PaymentRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE id = $1\nFOR UPDATE;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get payment by id %s for update: %w", id, err)
	}

	return payment, nil
}

func (r *PaymentRepository) GetByExternalID(ctx context.Context, externalID string) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE external_id = $1;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, externalID))
	if err != nil {
		return nil, fmt.Errorf("get payment by external id %s: %w", externalID, err)
	}

	return payment, nil
}

func (r *PaymentRepository) GetByExternalIDForUpdate(ctx context.Context, externalID string) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE external_id = $1\nFOR UPDATE;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, externalID))
	if err != nil {
		return nil, fmt.Errorf("get payment by external id %s for update: %w", externalID, err)
	}

	return payment, nil
}

func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE idempotency_key = $1;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, key))
	if err != nil {
		return nil, fmt.Errorf("get payment by idempotency key %s: %w", key, err)
	}

	return payment, nil
}

func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error) {
	const query = basePaymentSelectQuery + "\nWHERE order_id = $1\nORDER BY created_at DESC\nLIMIT 1;"

	payment, err := r.scanPayment(r.db.QueryRow(ctx, query, orderID))
	if err != nil {
		return nil, fmt.Errorf("get payment by order id %s: %w", orderID, err)
	}

	return payment, nil
}

func (r *PaymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	const query = `
		UPDATE payments
		SET
			status = $2,
			external_id = $3,
			client_secret = $4,
			failure_reason = $5,
			updated_at = $6
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		payment.ID,
		string(payment.Status),
		payment.ExternalID,
		payment.ClientSecret,
		payment.FailureReason,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update payment %s: %w", payment.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("update payment %s: %w", payment.ID, domain.ErrNotFound)
	}

	return nil
}

func (r *PaymentRepository) scanPayment(row pgx.Row) (*domain.Payment, error) {
	var m paymentModel
	err := row.Scan(
		&m.ID,
		&m.OrderID,
		&m.PayerUserID,
		&m.Amount,
		&m.Currency,
		&m.Status,
		&m.Provider,
		&m.ExternalID,
		&m.ClientSecret,
		&m.FailureReason,
		&m.IdempotencyKey,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	payment, err := m.toDomain()
	if err != nil {
		return nil, fmt.Errorf("restore payment domain: %w", err)
	}

	return payment, nil
}
