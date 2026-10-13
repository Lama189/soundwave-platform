package postgres

import (
	"context"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type ArtistRepository struct {
	db DB
}

func NewArtistRepository(db DB) *ArtistRepository {
	return &ArtistRepository{db: db}
}

func (r *ArtistRepository) Create(ctx context.Context, artist *domain.Artist) error {
	const query = `
		INSERT INTO artists (
			user_id, name, bio, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
	`

	err := r.db.QueryRow(
		ctx,
		query,
		artist.UserID,
		artist.Name,
		artist.Bio,
		artist.CreatedAt,
		artist.UpdatedAt,
	).Scan(&artist.ID)
	if err != nil {
		return fmt.Errorf("create artist: %w", mapError(err))
	}

	return nil
}

func (r *ArtistRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artist, error) {
	const query = `
		SELECT id, user_id, name, bio, created_at, updated_at
		FROM artists
		WHERE id = $1;
	`

	var m artistModel
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.UserID,
		&m.Name,
		&m.Bio,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get artist by id %s: %w", id, mapError(err))
	}

	return m.toDomain(), nil
}

func (r *ArtistRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Artist, error) {
	const query = `
		SELECT id, user_id, name, bio, created_at, updated_at
		FROM artists
		WHERE user_id = $1;
	`

	var m artistModel
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&m.ID,
		&m.UserID,
		&m.Name,
		&m.Bio,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get artist by user_id %s: %w", userID, mapError(err))
	}

	return m.toDomain(), nil
}

func (r *ArtistRepository) Update(ctx context.Context, artist *domain.Artist) error {
	const query = `
		UPDATE artists
		SET user_id = $1, name = $2, bio = $3, updated_at = $4
		WHERE id = $5;
	`

	cmdTag, err := r.db.Exec(
		ctx,
		query,
		artist.UserID,
		artist.Name,
		artist.Bio,
		artist.UpdatedAt,
		artist.ID,
	)
	if err != nil {
		return fmt.Errorf("update artist %s: %w", artist.ID, mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: artist id %s", domain.ErrNotFound, artist.ID)
	}

	return nil
}

func (r *ArtistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `
		DELETE FROM artists
		WHERE id = $1;
	`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete artist: %w", mapError(err))
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: artist id %s", domain.ErrNotFound, id)
	}

	return nil
}
