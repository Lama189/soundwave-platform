package album

import (
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type CreateAlbumDTO struct {
	ArtistID    uuid.UUID
	Title       string
	Description string
	CoverKey    string
	Price       int64
	Currency    domain.Currency
}

type UpdateAlbumDTO struct {
	Title       string
	Description string
	CoverKey    string
	Price       int64
	Currency    domain.Currency
}

type AlbumOutputDTO struct {
	ID          uuid.UUID
	ArtistID    uuid.UUID
	Title       string
	Description string
	CoverKey    string
	Price       int64
	Currency    domain.Currency
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func toOutputDTO(a *domain.Album) *AlbumOutputDTO {
	if a == nil {
		return nil
	}

	return &AlbumOutputDTO{
		ID:          a.ID,
		ArtistID:    a.ArtistID,
		Title:       a.Title,
		Description: a.Description,
		CoverKey:    a.CoverKey,
		Price:       a.Price,
		Currency:    a.Currency,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func toOutputDTOList(albums []*domain.Album) []*AlbumOutputDTO {
	if albums == nil {
		return nil
	}

	list := make([]*AlbumOutputDTO, 0, len(albums))
	for _, a := range albums {
		list = append(list, toOutputDTO(a))
	}

	return list
}
