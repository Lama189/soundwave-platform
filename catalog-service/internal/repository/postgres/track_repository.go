package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type TrackRepository struct {
	db DB
}

func NewTrackRepository(db DB) *TrackRepository {
	return &TrackRepository{db: db}
}

func (r *TrackRepository) Create(ctx context.Context, track *domain.Track) error {
	const query = `
		INSERT INTO tracks (
			id, artist_id, album_id, title, duration_seconds,
			cover_key, file_key, preview_key, raw_key, price, currency, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id;
	`

	err := r.db.QueryRow(
		ctx,
		query,
		track.ID,
		track.ArtistID,
		track.AlbumID,
		track.Title,
		int(track.Duration.Seconds()),
		track.CoverKey,
		track.FileKey,
		track.PreviewKey,
		track.RawKey,
		track.Price,
		string(track.Currency),
		string(track.Status),
		track.CreatedAt,
		track.UpdatedAt,
	).Scan(&track.ID)
	if err != nil {
		return fmt.Errorf("create track: %w", mapError(err))
	}

	return nil
}

func (r *TrackRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
	const query = `
		SELECT id, artist_id, album_id, title, duration_seconds,
		       cover_key, file_key, preview_key, raw_key, price, currency, status, created_at, updated_at
		FROM tracks
		WHERE id = $1;
	`

	var m trackModel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.ArtistID,
		&m.AlbumID,
		&m.Title,
		&m.DurationSeconds,
		&m.CoverKey,
		&m.FileKey,
		&m.PreviewKey,
		&m.RawKey,
		&m.Price,
		&m.Currency,
		&m.Status,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get track by id %s: %w", id, mapError(err))
	}

	return m.toDomain(), nil
}

func (r *TrackRepository) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Track, error) {
	const query = `
		SELECT id, artist_id, album_id, title, duration_seconds,
		       cover_key, file_key, preview_key, raw_key, price, currency, status, created_at, updated_at
		FROM tracks
		WHERE artist_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, artistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get tracks by artist id %s: %w", artistID, mapError(err))
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var m trackModel
		if err := rows.Scan(
			&m.ID,
			&m.ArtistID,
			&m.AlbumID,
			&m.Title,
			&m.DurationSeconds,
			&m.CoverKey,
			&m.FileKey,
			&m.PreviewKey,
			&m.RawKey,
			&m.Price,
			&m.Currency,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		tracks = append(tracks, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error) {
	const query = `
		SELECT id, artist_id, album_id, title, duration_seconds,
		       cover_key, file_key, preview_key, raw_key, price, currency, status, created_at, updated_at
		FROM tracks
		WHERE album_id = $1
		ORDER BY created_at ASC;
	`

	rows, err := r.db.Query(ctx, query, albumID)
	if err != nil {
		return nil, fmt.Errorf("get tracks by album id %s: %w", albumID, mapError(err))
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var m trackModel
		if err := rows.Scan(
			&m.ID,
			&m.ArtistID,
			&m.AlbumID,
			&m.Title,
			&m.DurationSeconds,
			&m.CoverKey,
			&m.FileKey,
			&m.PreviewKey,
			&m.RawKey,
			&m.Price,
			&m.Currency,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		tracks = append(tracks, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	const query = `
		SELECT id, artist_id, album_id, title, duration_seconds,
		       cover_key, file_key, preview_key, raw_key, price, currency, status, created_at, updated_at
		FROM tracks
		WHERE id = ANY($1);
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("get tracks by ids: %w", mapError(err))
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var m trackModel
		if err := rows.Scan(
			&m.ID,
			&m.ArtistID,
			&m.AlbumID,
			&m.Title,
			&m.DurationSeconds,
			&m.CoverKey,
			&m.FileKey,
			&m.PreviewKey,
			&m.RawKey,
			&m.Price,
			&m.Currency,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		tracks = append(tracks, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks: %w", err)
	}

	return tracks, nil
}

func (r *TrackRepository) Update(ctx context.Context, track *domain.Track) error {
	const query = `
		UPDATE tracks
		SET title = $1, album_id = $2, cover_key = $3, price = $4, currency = $5, updated_at = $6
		WHERE id = $7;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		track.Title,
		track.AlbumID,
		track.CoverKey,
		track.Price,
		string(track.Currency),
		track.UpdatedAt,
		track.ID,
	)
	if err != nil {
		return fmt.Errorf("update track %s: %w", track.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: track id %s", domain.ErrNotFound, track.ID)
	}

	return nil
}

func (r *TrackRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.TrackStatus,
	fileKey, previewKey string,
	duration time.Duration,
) error {
	const query = `
		UPDATE tracks
		SET status = $1, file_key = $2, preview_key = $3,
		    duration_seconds = $4, updated_at = $5
		WHERE id = $6;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		string(status),
		fileKey,
		previewKey,
		int(duration.Seconds()),
		time.Now().UTC(),
		id,
	)
	if err != nil {
		return fmt.Errorf("update track status %s: %w", id, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: track id %s", domain.ErrNotFound, id)
	}

	return nil
}

func (r *TrackRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `
		DELETE FROM tracks
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete track %s: %w", id, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: track id %s", domain.ErrNotFound, id)
	}

	return nil
}
