package playlist_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/playlist"
	"github.com/google/uuid"
)

type mockPlaylistRepo struct {
	getByIDFn  func(ctx context.Context, id uuid.UUID) (*domain.Playlist, error)
	addTrackFn func(ctx context.Context, playlistID, trackID uuid.UUID, position int) error
}

func (m *mockPlaylistRepo) Create(ctx context.Context, p *domain.Playlist) error { return nil }
func (m *mockPlaylistRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Playlist, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPlaylistRepo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Playlist, error) {
	return nil, nil
}
func (m *mockPlaylistRepo) Update(ctx context.Context, p *domain.Playlist) error { return nil }
func (m *mockPlaylistRepo) Delete(ctx context.Context, id uuid.UUID) error       { return nil }
func (m *mockPlaylistRepo) AddTrack(ctx context.Context, playlistID, trackID uuid.UUID, position int) error {
	if m.addTrackFn != nil {
		return m.addTrackFn(ctx, playlistID, trackID, position)
	}
	return nil
}
func (m *mockPlaylistRepo) RemoveTrack(ctx context.Context, playlistID, trackID uuid.UUID) error {
	return nil
}
func (m *mockPlaylistRepo) GetTracks(ctx context.Context, playlistID uuid.UUID) ([]*domain.PlaylistTrack, error) {
	return nil, nil
}
func (m *mockPlaylistRepo) ChangeTrackPosition(ctx context.Context, playlistID, trackID uuid.UUID, newPosition int) error {
	return nil
}

type mockTrackRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Track, error)
}

func (m *mockTrackRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTrackRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error) {
	return nil, nil
}

type mockAccessChecker struct {
	hasAccessFn func(ctx context.Context, userID *uuid.UUID, trackID uuid.UUID) (bool, error)
}

func (m *mockAccessChecker) HasTrackAccess(ctx context.Context, userID *uuid.UUID, trackID uuid.UUID) (bool, error) {
	if m.hasAccessFn != nil {
		return m.hasAccessFn(ctx, userID, trackID)
	}
	return false, nil
}

func TestAddTrack_AccessCheck(t *testing.T) {
	userID := uuid.New()
	playlistID := uuid.New()
	trackID := uuid.New()
	artistID := uuid.New()

	playlistObj := domain.RestorePlaylist(playlistID, userID, "My Playlist", "", false, time.Now())

	tests := []struct {
		name      string
		track     *domain.Track
		hasAccess bool
		wantErr   error
	}{
		{
			name: "purchased track can be added to playlist",
			track: domain.RestoreTrack(
				trackID, artistID, nil, "Title", 180*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now(),
			),
			hasAccess: true,
			wantErr:   nil,
		},
		{
			name: "unpurchased paid track returns ErrTrackNotPurchased",
			track: domain.RestoreTrack(
				trackID, artistID, nil, "Title", 180*time.Second, "", "", "", "", 100, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now(),
			),
			hasAccess: false,
			wantErr:   domain.ErrTrackNotPurchased,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			playlistRepo := &mockPlaylistRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Playlist, error) {
					return playlistObj, nil
				},
			}
			trackRepo := &mockTrackRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
					return tt.track, nil
				},
			}
			accessChecker := &mockAccessChecker{
				hasAccessFn: func(ctx context.Context, u *uuid.UUID, tid uuid.UUID) (bool, error) {
					return tt.hasAccess, nil
				},
			}

			svc := playlist.NewService(playlistRepo, trackRepo, accessChecker)

			err := svc.AddTrack(context.Background(), userID, playlistID, trackID, 0)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("AddTrack() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
