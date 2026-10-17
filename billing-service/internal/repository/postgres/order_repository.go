package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/billing-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OrderRepository struct {
	db DB
}

func NewOrderRepository(db DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	const queryOrder = `
		INSERT INTO orders (
			id,
			buyer_user_id,
			status,
			total_amount,
			currency,
			payment_id,
			failure_reason,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		);
	`

	_, err := r.db.Exec(
		ctx,
		queryOrder,
		order.ID,
		order.BuyerUserID,
		string(order.Status),
		order.TotalAmount.Amount,
		string(order.TotalAmount.Currency),
		order.PaymentID,
		order.FailureReason,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order %s: %w", order.ID, mapError(err))
	}

	if len(order.Items) == 0 {
		return nil
	}

	rows := make([][]any, len(order.Items))
	for i, item := range order.Items {
		rows[i] = []any{
			item.ID,
			order.ID,
			string(item.ItemType),
			item.ItemID,
			item.ArtistID,
			item.Price.Amount,
			item.PlatformFee.Amount,
			item.ArtistShare.Amount,
			string(item.Price.Currency),
		}
	}

	columns := []string{
		"id", "order_id", "item_type", "item_id", "artist_id",
		"price", "platform_fee", "artist_share", "currency",
	}

	_, err = r.db.CopyFrom(ctx, pgx.Identifier{"order_items"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("bulk insert order items for order %s: %w", order.ID, mapError(err))
	}

	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	const queryOrder = `
		SELECT 
			id,
			buyer_user_id,
			status,
			total_amount,
			currency,
			payment_id,
			failure_reason,
			created_at,
			updated_at
		FROM orders
		WHERE id = $1;
	`

	var om orderModel
	err := r.db.QueryRow(ctx, queryOrder, id).Scan(
		&om.ID,
		&om.BuyerUserID,
		&om.Status,
		&om.TotalAmount,
		&om.Currency,
		&om.PaymentID,
		&om.FailureReason,
		&om.CreatedAt,
		&om.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get order by id %s: %w", id, mapError(err))
	}

	const queryItems = `
		SELECT 
			id,
			order_id,
			item_type,
			item_id,
			artist_id,
			price,
			platform_fee,
			artist_share,
			currency
		FROM order_items
		WHERE order_id = $1;
	`

	rows, err := r.db.Query(ctx, queryItems, id)
	if err != nil {
		return nil, fmt.Errorf("get items for order %s: %w", id, mapError(err))
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var im orderItemModel
		if err := rows.Scan(
			&im.ID,
			&im.OrderID,
			&im.ItemType,
			&im.ItemID,
			&im.ArtistID,
			&im.Price,
			&im.PlatformFee,
			&im.ArtistShare,
			&im.Currency,
		); err != nil {
			return nil, fmt.Errorf("scan order item record for order %s: %w", id, err)
		}
		items = append(items, im.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items for order %s: %w", id, err)
	}

	return om.toDomain(items), nil
}

func (r *OrderRepository) GetByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	const queryOrders = `
		SELECT 
			id,
			buyer_user_id,
			status,
			total_amount,
			currency,
			payment_id,
			failure_reason,
			created_at,
			updated_at
		FROM orders
		WHERE buyer_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, queryOrders, buyerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get orders for buyer %s: %w", buyerID, mapError(err))
	}
	defer rows.Close()

	var orderModels []orderModel
	orderIDs := make([]uuid.UUID, 0)

	for rows.Next() {
		var om orderModel
		if err := rows.Scan(
			&om.ID,
			&om.BuyerUserID,
			&om.Status,
			&om.TotalAmount,
			&om.Currency,
			&om.PaymentID,
			&om.FailureReason,
			&om.CreatedAt,
			&om.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order record for buyer %s: %w", buyerID, err)
		}
		orderModels = append(orderModels, om)
		orderIDs = append(orderIDs, om.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders for buyer %s: %w", buyerID, err)
	}

	if len(orderModels) == 0 {
		return make([]*domain.Order, 0), nil
	}

	const queryItems = `
		SELECT 
			id,
			order_id,
			item_type,
			item_id,
			artist_id,
			price,
			platform_fee,
			artist_share,
			currency
		FROM order_items
		WHERE order_id = ANY($1);
	`

	itemRows, err := r.db.Query(ctx, queryItems, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("get items for buyer %s orders: %w", buyerID, mapError(err))
	}
	defer itemRows.Close()

	itemsByOrderID := make(map[uuid.UUID][]domain.OrderItem)
	for itemRows.Next() {
		var im orderItemModel
		if err := itemRows.Scan(
			&im.ID,
			&im.OrderID,
			&im.ItemType,
			&im.ItemID,
			&im.ArtistID,
			&im.Price,
			&im.PlatformFee,
			&im.ArtistShare,
			&im.Currency,
		); err != nil {
			return nil, fmt.Errorf("scan order item record: %w", err)
		}
		itemsByOrderID[im.OrderID] = append(itemsByOrderID[im.OrderID], im.toDomain())
	}

	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items for buyer %s orders: %w", buyerID, err)
	}

	orders := make([]*domain.Order, 0, len(orderModels))
	for _, om := range orderModels {
		items := itemsByOrderID[om.ID]
		if items == nil {
			items = make([]domain.OrderItem, 0)
		}
		orders = append(orders, om.toDomain(items))
	}

	return orders, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	const query = `
		UPDATE orders
		SET 
			status = $2,
			payment_id = $3,
			failure_reason = $4,
			updated_at = $5
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		order.ID,
		string(order.Status),
		order.PaymentID,
		order.FailureReason,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update order %s: %w", order.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("update order %s: %w", order.ID, domain.ErrNotFound)
	}

	return nil
}
