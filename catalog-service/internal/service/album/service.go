package album

import (
	"context"
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	albums  AlbumRepository
	artists ArtistRepository
}

func NewService(albums AlbumRepository, artists ArtistRepository) *Service {
	return &Service{albums: albums, artists: artists}
}

func (s *Service) Create(ctx context.Context, currentUserID uuid.UUID, dto CreateAlbumDTO) (*AlbumOutputDTO, error) {
	artist, err := s.artists.GetByID(ctx, dto.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("get artist by id %s: %w", dto.ArtistID, err)
	}

	if artist.UserID != currentUserID {
		return nil, domain.ErrNotArtistOwner
	}

	newAlbum, err := domain.NewAlbum(
		dto.ArtistID,
		dto.Title,
		dto.Description,
		dto.CoverKey,
		dto.Price,
		dto.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("create album: %w", err)
	}

	if err := s.albums.Create(ctx, newAlbum); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, fmt.Errorf("save album: %w", err)
	}

	return toOutputDTO(newAlbum), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*AlbumOutputDTO, error) {
	a, err := s.albums.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrAlbumNotFound
		}
		return nil, fmt.Errorf("get album by id: %s: %w", id, err)
	}

	return toOutputDTO(a), nil
}

func (s *Service) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*AlbumOutputDTO, error) {
	if len(ids) == 0 {
		return []*AlbumOutputDTO{}, nil
	}

	albums, err := s.albums.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get albums by ids: %w", err)
	}

	return toOutputDTOList(albums), nil
}

func (s *Service) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*AlbumOutputDTO, error) {
	if _, err := s.artists.GetByID(ctx, artistID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("check artist exists %s: %w", artistID, err)
	}

	albums, err := s.albums.GetByArtistID(ctx, artistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get albums by artist id %s: %w", artistID, err)
	}

	return toOutputDTOList(albums), nil
}

func (s *Service) Update(ctx context.Context, currentUserID, albumID uuid.UUID, dto UpdateAlbumDTO) (*AlbumOutputDTO, error) {
	album, err := s.albums.GetByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrAlbumNotFound
		}
		return nil, fmt.Errorf("find album for update: %w", err)
	}

	artist, err := s.artists.GetByID(ctx, album.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("find album artist: %w", err)
	}

	if artist.UserID != currentUserID {
		return nil, domain.ErrNotAlbumOwner
	}

	if err := album.Update(dto.Title, dto.Description, dto.CoverKey, dto.Price, dto.Currency); err != nil {
		return nil, fmt.Errorf("update album entity: %w", err)
	}

	if err := s.albums.Update(ctx, album); err != nil {
		return nil, fmt.Errorf("save updated album: %w", err)
	}

	return toOutputDTO(album), nil
}

func (s *Service) Delete(ctx context.Context, currentUserID, albumID uuid.UUID) error {
	album, err := s.albums.GetByID(ctx, albumID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrAlbumNotFound
		}
		return fmt.Errorf("find album for delete: %w", err)
	}

	artist, err := s.artists.GetByID(ctx, album.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrArtistNotFound
		}
		return fmt.Errorf("find album artist: %w", err)
	}

	if artist.UserID != currentUserID {
		return domain.ErrNotAlbumOwner
	}

	if err := s.albums.Delete(ctx, albumID); err != nil {
		return fmt.Errorf("delete album: %w", err)
	}

	return nil
}
