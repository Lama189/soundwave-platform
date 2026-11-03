package playlist

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type PlaylistRepository interface {
	Create(ctx context.Context, playlist *domain.Playlist) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Playlist, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Playlist, error)
	Update(ctx context.Context, playlist *domain.Playlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddTrack(ctx context.Context, playlistID, trackID uuid.UUID, position int) error
	RemoveTrack(ctx context.Context, playlistID, trackID uuid.UUID) error
	GetTracks(ctx context.Context, playlistID uuid.UUID) ([]*domain.PlaylistTrack, error)
	ChangeTrackPosition(ctx context.Context, playlistID, trackID uuid.UUID, newPosition int) error
}

type TrackRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error)
}

type AccessChecker interface {
	HasTrackAccess(ctx context.Context, userID *uuid.UUID, trackID uuid.UUID) (bool, error)
}
