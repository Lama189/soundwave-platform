package playlist

import (
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type CreatePlaylistDTO struct {
	Title       string
	Description string
	IsPrivate   bool
}

type UpdatePlaylistDTO struct {
	Title       string
	Description string
	IsPrivate   bool
}

type PlaylistOutputDTO struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	IsPrivate   bool
	CreatedAt   time.Time
}

type PlaylistTrackItemDTO struct {
	TrackID  uuid.UUID
	Position int
	AddedAt  time.Time
	Title    string
	ArtistID uuid.UUID
	Duration time.Duration
	CoverKey string
	Status   string
}

type PlaylistWithTracksDTO struct {
	Playlist PlaylistOutputDTO
	Tracks   []PlaylistTrackItemDTO
}

func toOutputDTO(p *domain.Playlist) *PlaylistOutputDTO {
	if p == nil {
		return nil
	}

	return &PlaylistOutputDTO{
		ID:          p.ID,
		UserID:      p.UserID,
		Title:       p.Title,
		Description: p.Description,
		IsPrivate:   p.IsPrivate,
		CreatedAt:   p.CreatedAt,
	}
}

func toOutputDTOList(playlists []*domain.Playlist) []*PlaylistOutputDTO {
	if playlists == nil {
		return nil
	}

	list := make([]*PlaylistOutputDTO, 0, len(playlists))
	for _, p := range playlists {
		list = append(list, toOutputDTO(p))
	}

	return list
}
