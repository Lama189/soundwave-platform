package track_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/track"
	"github.com/google/uuid"
)

type mockGetTrackClient struct {
	getTrackFunc func(ctx context.Context, trackID uuid.UUID) (*grpcclient.TrackResponse, error)
}

func (m *mockGetTrackClient) UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error) {
	return nil, nil
}
func (m *mockGetTrackClient) ConfirmUpload(ctx context.Context, req grpcclient.ConfirmUploadRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}
func (m *mockGetTrackClient) CompleteProcessing(ctx context.Context, req grpcclient.CompleteProcessingRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}
func (m *mockGetTrackClient) FailProcessing(ctx context.Context, req grpcclient.FailProcessingRequest) (*grpcclient.FailProcessingResponse, error) {
	return nil, nil
}
func (m *mockGetTrackClient) GetTrackStream(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error) {
	return nil, nil
}
func (m *mockGetTrackClient) GetTrack(ctx context.Context, trackID uuid.UUID) (*grpcclient.TrackResponse, error) {
	if m.getTrackFunc != nil {
		return m.getTrackFunc(ctx, trackID)
	}
	return nil, nil
}
func (m *mockGetTrackClient) LikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	return nil
}
func (m *mockGetTrackClient) UnlikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	return nil
}

func TestGetTrackInput_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name    string
		input   track.GetTrackInput
		wantErr bool
	}{
		{
			name:    "valid track id",
			input:   track.GetTrackInput{TrackID: validID},
			wantErr: false,
		},
		{
			name:    "missing track id",
			input:   track.GetTrackInput{TrackID: uuid.Nil},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTrackUseCase_Execute(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	trackID := uuid.New()
	artistID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockClient := &mockGetTrackClient{
			getTrackFunc: func(ctx context.Context, id uuid.UUID) (*grpcclient.TrackResponse, error) {
				if id != trackID {
					t.Errorf("expected trackID %s, got %s", trackID, id)
				}
				return &grpcclient.TrackResponse{
					ID:              trackID,
					ArtistID:        artistID,
					Title:           "Song Title",
					DurationSeconds: 210,
					Status:          "ready",
					Price:           100,
					Currency:        "USD",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}, nil
			},
		}

		uc := track.NewGetTrackUseCase(mockClient, discardLogger)
		output, err := uc.Execute(context.Background(), track.GetTrackInput{
			TrackID: trackID,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.ID != trackID {
			t.Errorf("expected track ID %s, got %s", trackID, output.ID)
		}
		if output.Title != "Song Title" {
			t.Errorf("expected title 'Song Title', got %s", output.Title)
		}
	})

	t.Run("client error", func(t *testing.T) {
		mockClient := &mockGetTrackClient{
			getTrackFunc: func(ctx context.Context, id uuid.UUID) (*grpcclient.TrackResponse, error) {
				return nil, errors.New("gRPC not found")
			},
		}

		uc := track.NewGetTrackUseCase(mockClient, discardLogger)
		output, err := uc.Execute(context.Background(), track.GetTrackInput{
			TrackID: trackID,
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if output != nil {
			t.Errorf("expected nil output, got %v", output)
		}
	})
}
