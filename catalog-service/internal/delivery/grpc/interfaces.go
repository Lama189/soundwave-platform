package v1

import (
	"context"

	"github.com/google/uuid"

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
}


