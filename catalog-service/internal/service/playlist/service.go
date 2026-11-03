package playlist

import (
	"context"
	"errors"
	"fmt"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/google/uuid"
)

type Service struct {
	playlists PlaylistRepository
	tracks    TrackRepository
	access    AccessChecker
}

func NewService(playlists PlaylistRepository, tracks TrackRepository, access AccessChecker) *Service {
	return &Service{
		playlists: playlists,
		tracks:    tracks,
		access:    access,
	}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, dto CreatePlaylistDTO) (*PlaylistOutputDTO, error) {
	newPlaylist, err := domain.NewPlaylist(userID, dto.Title, dto.Description, dto.IsPrivate)
	if err != nil {
		return nil, fmt.Errorf("create playlist: %w", err)
	}

	if err := s.playlists.Create(ctx, newPlaylist); err != nil {
		return nil, fmt.Errorf("save playlist: %w", err)
	}

	return toOutputDTO(newPlaylist), nil
}

func (s *Service) GetByID(ctx context.Context, currentUserID *uuid.UUID, playlistID uuid.UUID) (*PlaylistWithTracksDTO, error) {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("get playlist by id %s: %w", playlistID, err)
	}

	if p.IsPrivate {
		if currentUserID == nil || *currentUserID != p.UserID {
			return nil, domain.ErrPrivatePlaylistAccess
		}
	}

	playlistTracks, err := s.playlists.GetTracks(ctx, playlistID)
	if err != nil {
		return nil, fmt.Errorf("get playlist tracks for %s: %w", playlistID, err)
	}

	if len(playlistTracks) == 0 {
		return &PlaylistWithTracksDTO{
			Playlist: *toOutputDTO(p),
			Tracks:   []PlaylistTrackItemDTO{},
		}, nil
	}

	trackIDs := make([]uuid.UUID, 0, len(playlistTracks))
	for _, pt := range playlistTracks {
		trackIDs = append(trackIDs, pt.TrackID)
	}

	tracks, err := s.tracks.GetByIDs(ctx, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("get tracks by ids: %w", err)
	}

	trackMap := make(map[uuid.UUID]*domain.Track, len(tracks))
	for _, t := range tracks {
		trackMap[t.ID] = t
	}

	items := make([]PlaylistTrackItemDTO, 0, len(playlistTracks))
	for _, pt := range playlistTracks {
		t, ok := trackMap[pt.TrackID]
		if !ok {
			continue
		}
		items = append(items, PlaylistTrackItemDTO{
			TrackID:  pt.TrackID,
			Position: pt.Position,
			AddedAt:  pt.AddedAt,
			Title:    t.Title,
			ArtistID: t.ArtistID,
			Duration: t.Duration,
			CoverKey: t.CoverKey,
			Status:   string(t.Status),
		})
	}

	return &PlaylistWithTracksDTO{
		Playlist: *toOutputDTO(p),
		Tracks:   items,
	}, nil
}

func (s *Service) GetMyPlaylists(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PlaylistOutputDTO, error) {
	playlists, err := s.playlists.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get playlists for user %s: %w", userID, err)
	}

	return toOutputDTOList(playlists), nil
}

func (s *Service) AddTrack(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, position int) error {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrPlaylistNotFound
		}
		return fmt.Errorf("find playlist %s: %w", playlistID, err)
	}

	if p.UserID != currentUserID {
		return domain.ErrNotPlaylistOwner
	}

	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTrackNotFound
		}
		return fmt.Errorf("find track %s: %w", trackID, err)
	}

	if t.Status != domain.TrackStatusReady {
		return domain.ErrTrackNotReady
	}

	if s.access != nil {
		hasAccess, err := s.access.HasTrackAccess(ctx, &currentUserID, trackID)
		if err != nil {
			return fmt.Errorf("check track access for playlist: %w", err)
		}
		if !hasAccess {
			return domain.ErrTrackNotPurchased
		}
	}

	if err := s.playlists.AddTrack(ctx, playlistID, trackID, position); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.ErrTrackAlreadyInPlaylist
		}
		return fmt.Errorf("add track to playlist: %w", err)
	}

	return nil
}

func (s *Service) RemoveTrack(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID) error {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrPlaylistNotFound
		}
		return fmt.Errorf("find playlist %s: %w", playlistID, err)
	}

	if p.UserID != currentUserID {
		return domain.ErrNotPlaylistOwner
	}

	if err := s.playlists.RemoveTrack(ctx, playlistID, trackID); err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrTrackNotInPlaylist) {
			return domain.ErrTrackNotInPlaylist
		}
		return fmt.Errorf("remove track from playlist: %w", err)
	}

	return nil
}

func (s *Service) ChangeTrackPosition(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, newPosition int) error {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrPlaylistNotFound
		}
		return fmt.Errorf("find playlist %s: %w", playlistID, err)
	}

	if p.UserID != currentUserID {
		return domain.ErrNotPlaylistOwner
	}

	if err := s.playlists.ChangeTrackPosition(ctx, playlistID, trackID, newPosition); err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrTrackNotInPlaylist) {
			return domain.ErrTrackNotInPlaylist
		}
		return fmt.Errorf("change track position in playlist: %w", err)
	}

	return nil
}

func (s *Service) Update(ctx context.Context, currentUserID, playlistID uuid.UUID, dto UpdatePlaylistDTO) (*PlaylistOutputDTO, error) {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrPlaylistNotFound
		}
		return nil, fmt.Errorf("find playlist for update %s: %w", playlistID, err)
	}

	if p.UserID != currentUserID {
		return nil, domain.ErrNotPlaylistOwner
	}

	if err := p.Update(dto.Title, dto.Description, dto.IsPrivate); err != nil {
		return nil, fmt.Errorf("update playlist entity: %w", err)
	}

	if err := s.playlists.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("save updated playlist: %w", err)
	}

	return toOutputDTO(p), nil
}

func (s *Service) Delete(ctx context.Context, currentUserID, playlistID uuid.UUID) error {
	p, err := s.playlists.GetByID(ctx, playlistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrPlaylistNotFound
		}
		return fmt.Errorf("find playlist for delete %s: %w", playlistID, err)
	}

	if p.UserID != currentUserID {
		return domain.ErrNotPlaylistOwner
	}

	if err := s.playlists.Delete(ctx, playlistID); err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}

	return nil
}
