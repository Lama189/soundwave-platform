package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type AlbumRepository struct {
	db DB
}

func NewAlbumRepository(db DB) *AlbumRepository {
	return &AlbumRepository{db: db}
}

func (r *AlbumRepository) Create(ctx context.Context, album *domain.Album) error {
	const query = `
		INSERT INTO albums (
			artist_id, title, description, cover_key, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`

	err := r.db.QueryRow(
		ctx,
		query,
		album.ArtistID,
		album.Title,
		album.Description,
		album.CoverKey,
		album.CreatedAt,
		album.UpdatedAt,
	).Scan(&album.ID)
	if err != nil {
		return fmt.Errorf("create album: %w", mapError(err))
	}

	return nil
}

func (r *AlbumRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error) {
	const query = `
		SELECT id, artist_id, title, description, cover_key, created_at, updated_at
		FROM albums
		WHERE id = $1;
	`

	var m albumModel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.ArtistID,
		&m.Title,
		&m.Description,
		&m.CoverKey,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get album by id %s: %w", id, mapError(err))
	}

	return m.toDomain(), nil
}

func (r *AlbumRepository) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Album, error) {
	const query = `
		SELECT id, artist_id, title, description, cover_key, created_at, updated_at
		FROM albums
		WHERE artist_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, artistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get albums by artist id %s: %w", artistID, mapError(err))
	}
	defer rows.Close()

	var albums []*domain.Album
	for rows.Next() {
		var m albumModel
		if err := rows.Scan(
			&m.ID,
			&m.ArtistID,
			&m.Title,
			&m.Description,
			&m.CoverKey,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, m.toDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate albums: %w", err)
	}

	return albums, nil
}

func (r *AlbumRepository) Update(ctx context.Context, album *domain.Album) error {
	const query = `
		UPDATE albums
		SET title = $1, description = $2, cover_key = $3, updated_at = $4
		WHERE id = $5;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		album.Title,
		album.Description,
		album.CoverKey,
		album.UpdatedAt,
		album.ID,
	)
	if err != nil {
		return fmt.Errorf("update album %s: %w", album.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: album id %s", domain.ErrNotFound, album.ID)
	}

	return nil
}

func (r *AlbumRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `
		DELETE FROM albums
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete album %s: %w", id, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: album id %s", domain.ErrNotFound, id)
	}

	return nil
}
