package track_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/track"
	"github.com/google/uuid"
)

type mockLikeClient struct {
	likeFunc   func(ctx context.Context, userID, trackID uuid.UUID) error
	unlikeFunc func(ctx context.Context, userID, trackID uuid.UUID) error
}

func (m *mockLikeClient) UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) ConfirmUpload(ctx context.Context, req grpcclient.ConfirmUploadRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) CompleteProcessing(ctx context.Context, req grpcclient.CompleteProcessingRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) FailProcessing(ctx context.Context, req grpcclient.FailProcessingRequest) (*grpcclient.FailProcessingResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) GetTrackStream(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) GetTrack(ctx context.Context, trackID uuid.UUID) (*grpcclient.TrackResponse, error) {
	return nil, nil
}
func (m *mockLikeClient) LikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	if m.likeFunc != nil {
		return m.likeFunc(ctx, userID, trackID)
	}
	return nil
}
func (m *mockLikeClient) UnlikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	if m.unlikeFunc != nil {
		return m.unlikeFunc(ctx, userID, trackID)
	}
	return nil
}

func TestLikeTrackUseCase(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	trackID := uuid.New()

	t.Run("success", func(t *testing.T) {
		client := &mockLikeClient{
			likeFunc: func(ctx context.Context, u, tr uuid.UUID) error {
				if u != userID || tr != trackID {
					t.Errorf("got u=%v, tr=%v", u, tr)
				}
				return nil
			},
		}

		uc := track.NewLikeTrackUseCase(client, discardLogger)
		err := uc.Execute(context.Background(), track.LikeTrackInput{
			UserID:  userID,
			TrackID: trackID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		uc := track.NewLikeTrackUseCase(&mockLikeClient{}, discardLogger)
		err := uc.Execute(context.Background(), track.LikeTrackInput{
			UserID:  uuid.Nil,
			TrackID: trackID,
		})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})

	t.Run("client error", func(t *testing.T) {
		client := &mockLikeClient{
			likeFunc: func(ctx context.Context, u, tr uuid.UUID) error {
				return errors.New("gRPC error")
			},
		}

		uc := track.NewLikeTrackUseCase(client, discardLogger)
		err := uc.Execute(context.Background(), track.LikeTrackInput{
			UserID:  userID,
			TrackID: trackID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUnlikeTrackUseCase(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userID := uuid.New()
	trackID := uuid.New()

	t.Run("success", func(t *testing.T) {
		client := &mockLikeClient{
			unlikeFunc: func(ctx context.Context, u, tr uuid.UUID) error {
				if u != userID || tr != trackID {
					t.Errorf("got u=%v, tr=%v", u, tr)
				}
				return nil
			},
		}

		uc := track.NewUnlikeTrackUseCase(client, discardLogger)
		err := uc.Execute(context.Background(), track.UnlikeTrackInput{
			UserID:  userID,
			TrackID: trackID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		uc := track.NewUnlikeTrackUseCase(&mockLikeClient{}, discardLogger)
		err := uc.Execute(context.Background(), track.UnlikeTrackInput{
			UserID:  userID,
			TrackID: uuid.Nil,
		})
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}
