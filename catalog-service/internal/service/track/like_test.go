package track_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	"github.com/google/uuid"
)

type mockTrackRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Track, error)
}

func (m *mockTrackRepo) Create(ctx context.Context, t *domain.Track) error { return nil }
func (m *mockTrackRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTrackRepo) GetByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int) ([]*domain.Track, error) {
	return nil, nil
}
func (m *mockTrackRepo) GetByAlbumID(ctx context.Context, albumID uuid.UUID) ([]*domain.Track, error) {
	return nil, nil
}
func (m *mockTrackRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Track, error) {
	return nil, nil
}
func (m *mockTrackRepo) Update(ctx context.Context, t *domain.Track) error { return nil }
func (m *mockTrackRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TrackStatus, fileKey, previewKey string, duration time.Duration) error {
	return nil
}
func (m *mockTrackRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type mockTrackLikeRepo struct {
	addedLikes []*domain.TrackLike
}

func (m *mockTrackLikeRepo) Add(ctx context.Context, like *domain.TrackLike) error {
	m.addedLikes = append(m.addedLikes, like)
	return nil
}
func (m *mockTrackLikeRepo) Remove(ctx context.Context, userID, trackID uuid.UUID) error { return nil }
func (m *mockTrackLikeRepo) IsLiked(ctx context.Context, userID, trackID uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockTrackLikeRepo) GetLikedTrackIDs(ctx context.Context, userID uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockTrackLikeRepo) CountByTrackID(ctx context.Context, trackID uuid.UUID) (int64, error) {
	return 0, nil
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

func TestLikeTrack_AccessCheck(t *testing.T) {
	userID := uuid.New()
	trackID := uuid.New()
	artistID := uuid.New()

	tests := []struct {
		name      string
		track     *domain.Track
		hasAccess bool
		wantErr   error
	}{
		{
			name: "purchased track can be liked",
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
		{
			name: "not ready track returns ErrTrackNotReady",
			track: domain.RestoreTrack(
				trackID, artistID, nil, "Title", 180*time.Second, "", "", "", "", 0, domain.CurrencyUSD, domain.TrackStatusDraft, time.Now(), time.Now(),
			),
			hasAccess: true,
			wantErr:   domain.ErrTrackNotReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackRepo := &mockTrackRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
					return tt.track, nil
				},
			}
			likeRepo := &mockTrackLikeRepo{}
			accessChecker := &mockAccessChecker{
				hasAccessFn: func(ctx context.Context, u *uuid.UUID, tid uuid.UUID) (bool, error) {
					return tt.hasAccess, nil
				},
			}

			svc := track.NewTrackService(nil, trackRepo, likeRepo, nil, nil, nil, accessChecker)

			err := svc.LikeTrack(context.Background(), userID, trackID)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("LikeTrack() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
