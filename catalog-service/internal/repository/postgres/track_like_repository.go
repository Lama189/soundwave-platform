package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type TrackLikeRepository struct {
	db DB
}

func NewTrackLikeRepository(db DB) *TrackLikeRepository {
	return &TrackLikeRepository{db: db}
}

func (r *TrackLikeRepository) Add(ctx context.Context, like *domain.TrackLike) error {
	const query = `
		INSERT INTO track_likes (user_id, track_id, created_at)
		VALUES ($1, $2, $3);
	`

	_, err := r.db.Exec(
		ctx,
		query,
		like.UserID,
		like.TrackID,
		like.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add track like: %w", mapError(err))
	}

	return nil
}

func (r *TrackLikeRepository) Remove(ctx context.Context, userID, trackID uuid.UUID) error {
	const query = `
		DELETE FROM track_likes
		WHERE user_id = $1 AND track_id = $2;
	`

	cmdTag, err := r.db.Exec(ctx, query, userID, trackID)
	if err != nil {
		return fmt.Errorf("remove track like: %w", mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: like not found", domain.ErrNotFound)
	}

	return nil
}

func (r *TrackLikeRepository) IsLiked(ctx context.Context, userID, trackID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM track_likes
			WHERE user_id = $1 AND track_id = $2
		);
	`

	var isLiked bool
	err := r.db.QueryRow(ctx, query, userID, trackID).Scan(&isLiked)
	if err != nil {
		return false, fmt.Errorf("check track is liked: %w", mapError(err))
	}

	return isLiked, nil
}

func (r *TrackLikeRepository) GetLikedTrackIDs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	const query = `
		SELECT track_id
		FROM track_likes
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get liked track ids for user %s: %w", userID, mapError(err))
	}
	defer rows.Close()

	var trackIDs []uuid.UUID
	for rows.Next() {
		var trackID uuid.UUID
		if err := rows.Scan(&trackID); err != nil {
			return nil, fmt.Errorf("scan liked track id: %w", err)
		}
		trackIDs = append(trackIDs, trackID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate liked track ids: %w", err)
	}

	return trackIDs, nil
}

func (r *TrackLikeRepository) CountByTrackID(ctx context.Context, trackID uuid.UUID) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM track_likes
		WHERE track_id = $1;
	`

	var count int64
	err := r.db.QueryRow(ctx, query, trackID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count likes for track %s: %w", trackID, mapError(err))
	}

	return count, nil
}
