package album

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type AlbumRepository interface {
	Create(ctx context.Context, album *domain.Album) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Album, error)
	GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Album, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Album, error)
	Update(ctx context.Context, album *domain.Album) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ArtistRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Artist, error)
}
