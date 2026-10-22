package track

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/uow"
	"github.com/google/uuid"
)

type Service struct {
	uow     UnitOfWorkManager
	tracks  TrackRepository
	likes   TrackLikeRepository
	artists ArtistRepository
	albums  AlbumRepository
	storage Storage
}

func NewTrackService(
	uowManager UnitOfWorkManager,
	tracks TrackRepository,
	likes TrackLikeRepository,
	artists ArtistRepository,
	albums AlbumRepository,
	storage Storage,
) *Service {
	return &Service{
		uow:     uowManager,
		tracks:  tracks,
		likes:   likes,
		artists: artists,
		albums:  albums,
		storage: storage,
	}
}

func (s *Service) LikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTrackNotFound
		}
		return fmt.Errorf("find track for like: %w", err)
	}

	if t.Status != domain.TrackStatusReady {
		return domain.ErrTrackNotReady
	}

	like, err := domain.NewTrackLike(userID, trackID)
	if err != nil {
		return fmt.Errorf("create track like entity: %w", err)
	}

	if err := s.likes.Add(ctx, like); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.ErrTrackAlreadyLiked
		}
		return fmt.Errorf("add track like: %w", err)
	}

	return nil
}

func (s *Service) UnlikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	if err := s.likes.Remove(ctx, userID, trackID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTrackNotLiked
		}
		return fmt.Errorf("remove track like: %w", err)
	}

	return nil
}

func (s *Service) GetLikedTracks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TrackOutputDTO, error) {
	trackIDs, err := s.likes.GetLikedTrackIDs(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get liked track ids for user %s: %w", userID, err)
	}

	if len(trackIDs) == 0 {
		return []*TrackOutputDTO{}, nil
	}

	tracks, err := s.tracks.GetByIDs(ctx, trackIDs)
	if err != nil {
		return nil, fmt.Errorf("get tracks by ids: %w", err)
	}

	trackMap := make(map[uuid.UUID]*domain.Track, len(tracks))
	for _, t := range tracks {
		trackMap[t.ID] = t
	}

	orderedTracks := make([]*domain.Track, 0, len(trackIDs))
	for _, id := range trackIDs {
		if t, ok := trackMap[id]; ok {
			orderedTracks = append(orderedTracks, t)
		}
	}

	return toOutputDTOList(orderedTracks), nil
}

func (s *Service) IsLiked(ctx context.Context, userID, trackID uuid.UUID) (bool, error) {
	isLiked, err := s.likes.IsLiked(ctx, userID, trackID)
	if err != nil {
		return false, fmt.Errorf("check track is liked: %w", err)
	}

	return isLiked, nil
}

func (s *Service) UploadTrack(ctx context.Context, currentUserID uuid.UUID, dto UploadTrackDTO) (*UploadTrackOutputDTO, error) {
	artist, err := s.artists.GetByUserID(ctx, currentUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("get artist by user id %s: %w", currentUserID, err)
	}

	if artist.UserID != currentUserID {
		return nil, domain.ErrNotArtistOwner
	}

	var albumID *uuid.UUID
	if dto.AlbumID != nil && *dto.AlbumID != uuid.Nil {
		album, err := s.albums.GetByID(ctx, *dto.AlbumID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrAlbumNotFound
			}
			return nil, fmt.Errorf("get album: %w", err)
		}

		if album.ArtistID != artist.ID {
			return nil, domain.ErrNotAlbumOwner
		}

		albumID = dto.AlbumID
	}

	ext := strings.ToLower(filepath.Ext(dto.FileName))
	if !domain.IsSupportedAudioExtension(ext) {
		return nil, domain.ErrUnsupportedAudioFormat
	}

	newTrack, err := domain.NewTrack(artist.ID, dto.Title, dto.Price, dto.Currency)
	if err != nil {
		return nil, fmt.Errorf("create track draft: %w", err)
	}

	if albumID != nil {
		newTrack.SetAlbum(albumID)
	}

	rawKey := fmt.Sprintf("raw/audio/%s%s", newTrack.ID, ext)
	newTrack.SetRawKey(rawKey)

	if err := s.tracks.Create(ctx, newTrack); err != nil {
		return nil, fmt.Errorf("save track draft: %w", err)
	}

	uploadURL, err := s.storage.GenerateUploadURL(ctx, rawKey)
	if err != nil {
		return nil, fmt.Errorf("generate upload url for track %s: %w", newTrack.ID, err)
	}

	return &UploadTrackOutputDTO{
		Track:     *toOutputDTO(newTrack),
		UploadURL: uploadURL,
	}, nil
}

func (s *Service) ConfirmUpload(ctx context.Context, currentUserID, trackID uuid.UUID) (*TrackOutputDTO, error) {
	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("find track for confirm upload: %w", err)
	}

	artist, err := s.artists.GetByID(ctx, t.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("find track artist: %w", err)
	}

	if artist.UserID != currentUserID {
		return nil, domain.ErrNotArtistOwner
	}

	if t.RawKey == "" {
		return nil, fmt.Errorf("%w: track has no raw file key", domain.ErrInvalidInput)
	}

	exists, err := s.storage.ObjectExists(ctx, t.RawKey)
	if err != nil {
		return nil, fmt.Errorf("check raw object exists for track %s: %w", trackID, err)
	}
	if !exists {
		return nil, domain.ErrRawFileNotFound
	}

	if err := t.MarkAsProcessing(); err != nil {
		return nil, fmt.Errorf("mark track processing: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"track_id":  t.ID,
		"artist_id": t.ArtistID,
		"raw_key":   t.RawKey,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}

	event, err := domain.NewEvent(uuid.New(), domain.AggregateTypeTrack, t.ID, domain.EventTypeTrackUploaded, payload)
	if err != nil {
		return nil, fmt.Errorf("create outbox event: %w", err)
	}

	if err := s.uow.Do(ctx, func(txUOW uow.UnitOfWork) error {
		if err := txUOW.Tracks().UpdateStatus(
			ctx,
			t.ID,
			t.Status,
			t.FileKey,
			t.PreviewKey,
			t.Duration,
		); err != nil {
			return fmt.Errorf("update track status to processing: %w", err)
		}

		if err := txUOW.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("confirm upload transaction: %w", err)
	}

	return toOutputDTO(t), nil
}

func (s *Service) CompleteProcessing(ctx context.Context, trackID uuid.UUID, dto CompleteProcessingDTO) (*TrackOutputDTO, error) {
	track, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("get track by id: %w", err)
	}

	if err := track.MarkAsReady(dto.CoverKey, dto.FileKey, dto.PreviewKey, dto.Duration); err != nil {
		return nil, fmt.Errorf("mark track ready: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"track_id":         track.ID,
		"artist_id":        track.ArtistID,
		"file_key":         track.FileKey,
		"preview_key":      track.PreviewKey,
		"duration_seconds": int64(track.Duration.Seconds()),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}

	event, err := domain.NewEvent(
		uuid.New(),
		domain.AggregateTypeTrack,
		track.ID,
		domain.EventTypeTrackReady,
		payload,
	)
	if err != nil {
		return nil, fmt.Errorf("create track ready outbox event: %w", err)
	}

	if err := s.uow.Do(ctx, func(txUOW uow.UnitOfWork) error {
		if err := txUOW.Tracks().UpdateStatus(
			ctx,
			track.ID,
			track.Status,
			track.FileKey,
			track.PreviewKey,
			track.Duration,
		); err != nil {
			return fmt.Errorf("update track status: %w", err)
		}

		if track.CoverKey != "" {
			if err := txUOW.Tracks().Update(ctx, track); err != nil {
				return fmt.Errorf("update track cover: %w", err)
			}
		}

		if err := txUOW.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("complete processing transaction: %w", err)
	}

	return toOutputDTO(track), nil
}

func (s *Service) FailProcessing(ctx context.Context, trackID uuid.UUID) error {
	track, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTrackNotFound
		}
		return fmt.Errorf("get track by id %s: %w", trackID, err)
	}

	if err := track.MarkAsFailed(); err != nil {
		return fmt.Errorf("mark track failed: %w", err)
	}

	if err := s.tracks.UpdateStatus(
		ctx,
		track.ID,
		track.Status,
		track.FileKey,
		track.PreviewKey,
		track.Duration,
	); err != nil {
		return fmt.Errorf("update track %s status to failed: %w", trackID, err)
	}

	return nil
}

func (s *Service) MarkProcessing(ctx context.Context, trackID uuid.UUID) error {
	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		return fmt.Errorf("get track by id %s: %w", trackID, err)
	}

	if t.Status == domain.TrackStatusProcessing || t.Status == domain.TrackStatusReady {
		return nil
	}

	if t.Status != domain.TrackStatusDraft {
		return fmt.Errorf("%w: track %s status is %s, expected %s",
			domain.ErrInvalidStateTransition, trackID, t.Status, domain.TrackStatusDraft)
	}

	if t.RawKey == "" {
		return fmt.Errorf("%w: track has no raw file key", domain.ErrInvalidInput)
	}

	if err := t.MarkAsProcessing(); err != nil {
		return fmt.Errorf("mark track processing: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"track_id":  t.ID,
		"artist_id": t.ArtistID,
		"raw_key":   t.RawKey,
	})
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	event, err := domain.NewEvent(
		uuid.New(),
		domain.AggregateTypeTrack,
		t.ID,
		domain.EventTypeTrackUploaded,
		payload,
	)
	if err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}

	if err := s.uow.Do(ctx, func(txUOW uow.UnitOfWork) error {
		if err := txUOW.Tracks().UpdateStatus(
			ctx,
			t.ID,
			t.Status,
			t.FileKey,
			t.PreviewKey,
			t.Duration,
		); err != nil {
			return fmt.Errorf("update track status to processing: %w", err)
		}

		if err := txUOW.Outbox().Record(ctx, event); err != nil {
			return fmt.Errorf("record outbox event: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("mark processing transaction: %w", err)
	}

	return nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*TrackOutputDTO, error) {
	t, err := s.tracks.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("get track by id %s: %w", id, err)
	}

	return toOutputDTO(t), nil
}

func (s *Service) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*TrackOutputDTO, error) {
	if len(ids) == 0 {
		return []*TrackOutputDTO{}, nil
	}

	tracks, err := s.tracks.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get tracks by ids: %w", err)
	}

	return toOutputDTOList(tracks), nil
}

func (s *Service) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*TrackOutputDTO, error) {
	if _, err := s.artists.GetByID(ctx, artistID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("check artist exists %s: %w", artistID, err)
	}

	tracks, err := s.tracks.GetByArtistID(ctx, artistID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get tracks by artist id %s: %w", artistID, err)
	}

	return toOutputDTOList(tracks), nil
}

func (s *Service) Update(ctx context.Context, currentUserID, trackID uuid.UUID, dto UpdateTrackDTO) (*TrackOutputDTO, error) {
	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("find track for update: %w", err)
	}

	artist, err := s.artists.GetByID(ctx, t.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrArtistNotFound
		}
		return nil, fmt.Errorf("find track artist: %w", err)
	}

	if artist.UserID != currentUserID {
		return nil, domain.ErrNotArtistOwner
	}

	if dto.AlbumID != nil && *dto.AlbumID != uuid.Nil {
		album, err := s.albums.GetByID(ctx, *dto.AlbumID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrAlbumNotFound
			}
			return nil, fmt.Errorf("get album: %w", err)
		}

		if album.ArtistID != artist.ID {
			return nil, domain.ErrNotAlbumOwner
		}
	}

	if err := t.Update(dto.Title, dto.AlbumID, dto.CoverKey, dto.Price, dto.Currency); err != nil {
		return nil, fmt.Errorf("update track entity: %w", err)
	}

	if err := s.tracks.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("save updated track: %w", err)
	}

	return toOutputDTO(t), nil
}

func (s *Service) Delete(ctx context.Context, currentUserID, trackID uuid.UUID) error {
	t, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrTrackNotFound
		}
		return fmt.Errorf("find track for delete: %w", err)
	}

	artist, err := s.artists.GetByID(ctx, t.ArtistID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrArtistNotFound
		}
		return fmt.Errorf("find track artist: %w", err)
	}

	if artist.UserID != currentUserID {
		return domain.ErrNotArtistOwner
	}

	if err := s.tracks.Delete(ctx, trackID); err != nil {
		return fmt.Errorf("delete track: %w", err)
	}

	return nil
}

func (s *Service) GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*TrackOutputDTO, error) {
	if _, err := s.albums.GetByID(ctx, albumID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrAlbumNotFound
		}
		return nil, fmt.Errorf("check album exists %s: %w", albumID, err)
	}

	tracks, err := s.tracks.GetByAlbumID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("get tracks by album id %s: %w", albumID, err)
	}

	return toOutputDTOList(tracks), nil
}
