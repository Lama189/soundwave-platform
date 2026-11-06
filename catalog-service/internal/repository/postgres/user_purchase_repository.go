package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type UserPurchaseRepository struct {
	db DB
}

func NewUserPurchaseRepository(db DB) *UserPurchaseRepository {
	return &UserPurchaseRepository{db: db}
}

func (r *UserPurchaseRepository) Create(ctx context.Context, purchase *domain.UserPurchase) error {
	const query = `
		INSERT INTO user_purchases (id, user_id, order_id, item_type, track_id, album_id, status, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT DO NOTHING;
	`

	_, err := r.db.Exec(
		ctx,
		query,
		purchase.ID,
		purchase.UserID,
		purchase.OrderID,
		string(purchase.ItemType),
		purchase.TrackID,
		purchase.AlbumID,
		string(purchase.Status),
		purchase.RevokedAt,
		purchase.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user purchase: %w", mapError(err))
	}

	return nil
}

func (r *UserPurchaseRepository) CreateBatch(ctx context.Context, purchases []*domain.UserPurchase) error {
	if len(purchases) == 0 {
		return nil
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		INSERT INTO user_purchases (id, user_id, order_id, item_type, track_id, album_id, status, revoked_at, created_at)
		VALUES
	`)

	args := make([]any, 0, len(purchases)*9)
	for i, p := range purchases {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		baseIdx := i * 9
		fmt.Fprintf(&queryBuilder, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			baseIdx+1, baseIdx+2, baseIdx+3, baseIdx+4, baseIdx+5, baseIdx+6, baseIdx+7, baseIdx+8, baseIdx+9,
		)
		args = append(args,
			p.ID,
			p.UserID,
			p.OrderID,
			string(p.ItemType),
			p.TrackID,
			p.AlbumID,
			string(p.Status),
			p.RevokedAt,
			p.CreatedAt,
		)
	}

	queryBuilder.WriteString(" ON CONFLICT DO NOTHING;")

	_, err := r.db.Exec(ctx, queryBuilder.String(), args...)
	if err != nil {
		return fmt.Errorf("create batch user purchases: %w", mapError(err))
	}

	return nil
}

func (r *UserPurchaseRepository) HasPurchasedTrack(ctx context.Context, userID, trackID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM user_purchases
			WHERE user_id = $1 AND track_id = $2 AND status = 'ACTIVE'
		);
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, userID, trackID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check track purchase exists: %w", mapError(err))
	}

	return exists, nil
}

func (r *UserPurchaseRepository) HasPurchasedAlbum(ctx context.Context, userID, albumID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM user_purchases
			WHERE user_id = $1 AND album_id = $2 AND status = 'ACTIVE'
		);
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, userID, albumID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check album purchase exists: %w", mapError(err))
	}

	return exists, nil
}

func (r *UserPurchaseRepository) GetPurchasedTrackIDsIn(ctx context.Context, userID uuid.UUID, trackIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(trackIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
	}

	const query = `
		SELECT track_id
		FROM user_purchases
		WHERE user_id = $1 AND track_id = ANY($2) AND status = 'ACTIVE';
	`

	rows, err := r.db.Query(ctx, query, userID, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("get purchased track ids in: %w", mapError(err))
	}
	defer rows.Close()

	purchased := make(map[uuid.UUID]bool, len(trackIDs))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan purchased track id: %w", err)
		}
		purchased[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate purchased track rows: %w", mapError(err))
	}

	return purchased, nil
}

func (r *UserPurchaseRepository) ListByUserID(ctx context.Context, userID uuid.UUID, itemType *domain.ItemType, limit, offset int) ([]*domain.UserPurchase, error) {
	var (
		query string
		args  []any
	)

	if itemType != nil {
		query = `
			SELECT id, user_id, order_id, item_type, track_id, album_id, status, revoked_at, created_at
			FROM user_purchases
			WHERE user_id = $1 AND item_type = $2 AND status = 'ACTIVE'
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4;
		`
		args = []any{userID, string(*itemType), limit, offset}
	} else {
		query = `
			SELECT id, user_id, order_id, item_type, track_id, album_id, status, revoked_at, created_at
			FROM user_purchases
			WHERE user_id = $1 AND status = 'ACTIVE'
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3;
		`
		args = []any{userID, limit, offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list user purchases: %w", mapError(err))
	}
	defer rows.Close()

	var purchases []*domain.UserPurchase
	for rows.Next() {
		var m userPurchaseModel
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.OrderID,
			&m.ItemType,
			&m.TrackID,
			&m.AlbumID,
			&m.Status,
			&m.RevokedAt,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user purchase: %w", mapError(err))
		}
		purchases = append(purchases, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user purchase rows: %w", mapError(err))
	}

	return purchases, nil
}

func (r *UserPurchaseRepository) ExistsByOrderID(ctx context.Context, orderID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM user_purchases
			WHERE order_id = $1
		);
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, orderID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check purchase exists by order id: %w", mapError(err))
	}

	return exists, nil
}

func (r *UserPurchaseRepository) RevokeByOrderID(ctx context.Context, orderID uuid.UUID) error {
	const query = `
		UPDATE user_purchases
		SET
			status = 'REVOKED',
			revoked_at = NOW()
		WHERE order_id = $1 AND status = 'ACTIVE';
	`

	_, err := r.db.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("revoke purchases for order %s: %w", orderID, mapError(err))
	}

	return nil
}
