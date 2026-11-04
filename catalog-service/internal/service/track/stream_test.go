package track_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	"github.com/google/uuid"
)

type mockStorage struct {
	generateAudioDownloadURLFn func(ctx context.Context, key string, expiry time.Duration) (string, error)
}

func (m *mockStorage) GenerateUploadURL(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockStorage) GenerateAudioDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.generateAudioDownloadURLFn != nil {
		return m.generateAudioDownloadURLFn(ctx, key, expiry)
	}
	return fmt.Sprintf("https://minio.example.com/%s?expires=%s", key, expiry), nil
}

func (m *mockStorage) ObjectExists(ctx context.Context, key string) (bool, error) {
	return true, nil
}

type mockArtistRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Artist, error)
}

func (m *mockArtistRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artist, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockArtistRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Artist, error) {
	return nil, nil
}

func TestGetTrackStream(t *testing.T) {
	trackID := uuid.New()
	artistID := uuid.New()
	artistUserID := uuid.New()
	buyerUserID := uuid.New()
	otherUserID := uuid.New()

	artist := domain.RestoreArtist(artistID, artistUserID, "Artist Name", "bio", time.Now(), time.Now())

	readyTrackPaid := domain.RestoreTrack(
		trackID, artistID, nil, "Paid Track", 180*time.Second,
		"cover.jpg", "tracks/full.mp3", "tracks/preview.mp3", "raw.wav",
		150, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now(),
	)

	readyTrackFree := domain.RestoreTrack(
		trackID, artistID, nil, "Free Track", 180*time.Second,
		"cover.jpg", "tracks/full.mp3", "tracks/preview.mp3", "raw.wav",
		0, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now(),
	)

	readyTrackNoPreview := domain.RestoreTrack(
		trackID, artistID, nil, "No Preview Track", 180*time.Second,
		"cover.jpg", "tracks/full.mp3", "", "raw.wav",
		150, domain.CurrencyUSD, domain.TrackStatusReady, time.Now(), time.Now(),
	)

	draftTrack := domain.RestoreTrack(
		trackID, artistID, nil, "Draft Track", 180*time.Second,
		"", "", "", "raw.wav",
		0, domain.CurrencyUSD, domain.TrackStatusDraft, time.Now(), time.Now(),
	)

	tests := []struct {
		name             string
		track            *domain.Track
		trackErr         error
		userID           *uuid.UUID
		hasAccess        bool
		expiryConfig     time.Duration
		wantKey          string
		wantPreview      bool
		wantExpiryPassed time.Duration
		wantErr          error
	}{
		{
			name:             "purchased paid track gets full stream",
			track:            readyTrackPaid,
			userID:           &buyerUserID,
			hasAccess:        true,
			expiryConfig:     20 * time.Minute,
			wantKey:          "tracks/full.mp3",
			wantPreview:      false,
			wantExpiryPassed: 20 * time.Minute,
		},
		{
			name:             "free track gets full stream even for guest",
			track:            readyTrackFree,
			userID:           nil,
			hasAccess:        false,
			expiryConfig:     0, // should default to 15m
			wantKey:          "tracks/full.mp3",
			wantPreview:      false,
			wantExpiryPassed: 15 * time.Minute,
		},
		{
			name:             "unpurchased paid track gets preview stream",
			track:            readyTrackPaid,
			userID:           &otherUserID,
			hasAccess:        false,
			expiryConfig:     10 * time.Minute,
			wantKey:          "tracks/preview.mp3",
			wantPreview:      true,
			wantExpiryPassed: 10 * time.Minute,
		},
		{
			name:             "unpurchased paid track for guest gets preview stream",
			track:            readyTrackPaid,
			userID:           nil,
			hasAccess:        false,
			expiryConfig:     15 * time.Minute,
			wantKey:          "tracks/preview.mp3",
			wantPreview:      true,
			wantExpiryPassed: 15 * time.Minute,
		},
		{
			name:             "artist gets full stream for own track without purchasing",
			track:            readyTrackPaid,
			userID:           &artistUserID,
			hasAccess:        false,
			expiryConfig:     15 * time.Minute,
			wantKey:          "tracks/full.mp3",
			wantPreview:      false,
			wantExpiryPassed: 15 * time.Minute,
		},
		{
			name:             "unpurchased track with no preview key falls back to file key",
			track:            readyTrackNoPreview,
			userID:           &otherUserID,
			hasAccess:        false,
			expiryConfig:     15 * time.Minute,
			wantKey:          "tracks/full.mp3",
			wantPreview:      false,
			wantExpiryPassed: 15 * time.Minute,
		},
		{
			name:         "not ready track returns ErrTrackNotReady",
			track:        draftTrack,
			userID:       &buyerUserID,
			expiryConfig: 15 * time.Minute,
			wantErr:      domain.ErrTrackNotReady,
		},
		{
			name:         "non-existent track returns ErrTrackNotFound",
			track:        nil,
			trackErr:     domain.ErrNotFound,
			userID:       &buyerUserID,
			expiryConfig: 15 * time.Minute,
			wantErr:      domain.ErrTrackNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackRepo := &mockTrackRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Track, error) {
					if tt.trackErr != nil {
						return nil, tt.trackErr
					}
					return tt.track, nil
				},
			}

			artistRepo := &mockArtistRepo{
				getByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Artist, error) {
					return artist, nil
				},
			}

			var capturedKey string
			var capturedExpiry time.Duration
			storage := &mockStorage{
				generateAudioDownloadURLFn: func(ctx context.Context, key string, expiry time.Duration) (string, error) {
					capturedKey = key
					capturedExpiry = expiry
					return fmt.Sprintf("https://minio.example.com/%s", key), nil
				},
			}

			accessChecker := &mockAccessChecker{
				hasAccessFn: func(ctx context.Context, u *uuid.UUID, tid uuid.UUID) (bool, error) {
					return tt.hasAccess, nil
				},
			}

			svc := track.NewTrackService(
				nil,
				trackRepo,
				nil,
				artistRepo,
				nil,
				storage,
				accessChecker,
				tt.expiryConfig,
			)

			dto := track.GetTrackStreamDTO{
				TrackID: trackID,
				UserID:  tt.userID,
			}

			res, err := svc.GetTrackStream(context.Background(), dto)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("GetTrackStream() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if res.IsPreview != tt.wantPreview {
				t.Errorf("IsPreview = %v, want %v", res.IsPreview, tt.wantPreview)
			}
			if capturedKey != tt.wantKey {
				t.Errorf("storage key = %v, want %v", capturedKey, tt.wantKey)
			}
			if capturedExpiry != tt.wantExpiryPassed {
				t.Errorf("expiry = %v, want %v", capturedExpiry, tt.wantExpiryPassed)
			}
			if res.StreamURL == "" {
				t.Error("StreamURL should not be empty")
			}
			if res.ExpiresAt.Before(time.Now()) {
				t.Error("ExpiresAt should be in the future")
			}
		})
	}
}

func TestGetTrackStream_DTOValidation(t *testing.T) {
	svc := track.NewTrackService(nil, nil, nil, nil, nil, nil, nil, 0)

	// nil track ID
	_, err := svc.GetTrackStream(context.Background(), track.GetTrackStreamDTO{
		TrackID: uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for nil TrackID, got %v", err)
	}

	// nil uuid UserID pointer value
	nilUUID := uuid.Nil
	_, err = svc.GetTrackStream(context.Background(), track.GetTrackStreamDTO{
		TrackID: uuid.New(),
		UserID:  &nilUUID,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for empty UserID, got %v", err)
	}
}
