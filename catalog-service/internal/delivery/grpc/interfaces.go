package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/album"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/artist"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
)

type ArtistService interface {
	Create(ctx context.Context, userID uuid.UUID, dto artist.CreateArtistDTO) (*artist.ArtistOutputDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*artist.ArtistOutputDTO, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*artist.ArtistOutputDTO, error)
	Update(ctx context.Context, currentUserID, artistID uuid.UUID, dto artist.UpdateArtistDTO) (*artist.ArtistOutputDTO, error)
	Delete(ctx context.Context, currentUserID, artistID uuid.UUID) error
}

type TrackService interface {
	UploadTrack(ctx context.Context, currentUserID uuid.UUID, dto track.UploadTrackDTO) (*track.UploadTrackOutputDTO, error)
	ConfirmUpload(ctx context.Context, currentUserID, trackID uuid.UUID) (*track.TrackOutputDTO, error)
	CompleteProcessing(ctx context.Context, trackID uuid.UUID, dto track.CompleteProcessingDTO) (*track.TrackOutputDTO, error)
	FailProcessing(ctx context.Context, trackID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*track.TrackOutputDTO, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*track.TrackOutputDTO, error)
}

type AlbumService interface {
	Create(ctx context.Context, currentUserID uuid.UUID, dto album.CreateAlbumDTO) (*album.AlbumOutputDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*album.AlbumOutputDTO, error)
	GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*album.AlbumOutputDTO, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*album.AlbumOutputDTO, error)
	Update(ctx context.Context, currentUserID, albumID uuid.UUID, dto album.UpdateAlbumDTO) (*album.AlbumOutputDTO, error)
	Delete(ctx context.Context, currentUserID, albumID uuid.UUID) error
}
