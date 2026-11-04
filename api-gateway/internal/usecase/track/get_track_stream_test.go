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

type mockCatalogTrackClient struct {
	getTrackStreamFunc func(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error)
}

func (m *mockCatalogTrackClient) UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error) {
	return nil, nil
}

func (m *mockCatalogTrackClient) ConfirmUpload(ctx context.Context, req grpcclient.ConfirmUploadRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}

func (m *mockCatalogTrackClient) CompleteProcessing(ctx context.Context, req grpcclient.CompleteProcessingRequest) (*grpcclient.TrackResponse, error) {
	return nil, nil
}

func (m *mockCatalogTrackClient) FailProcessing(ctx context.Context, req grpcclient.FailProcessingRequest) (*grpcclient.FailProcessingResponse, error) {
	return nil, nil
}

func (m *mockCatalogTrackClient) GetTrackStream(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error) {
	if m.getTrackStreamFunc != nil {
		return m.getTrackStreamFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockCatalogTrackClient) GetTrack(ctx context.Context, trackID uuid.UUID) (*grpcclient.TrackResponse, error) {
	return nil, nil
}

func TestGetTrackStreamInput_Validate(t *testing.T) {
	validTrackID := uuid.New()
	validUserID := uuid.New()
	nilUUID := uuid.Nil

	tests := []struct {
		name    string
		input   track.GetTrackStreamInput
		wantErr bool
	}{
		{
			name: "valid with user id",
			input: track.GetTrackStreamInput{
				TrackID: validTrackID,
				UserID:  &validUserID,
			},
			wantErr: false,
		},
		{
			name: "valid without user id (anonymous)",
			input: track.GetTrackStreamInput{
				TrackID: validTrackID,
				UserID:  nil,
			},
			wantErr: false,
		},
		{
			name: "missing track id",
			input: track.GetTrackStreamInput{
				TrackID: uuid.Nil,
				UserID:  &validUserID,
			},
			wantErr: true,
		},
		{
			name: "nil user id pointer value",
			input: track.GetTrackStreamInput{
				TrackID: validTrackID,
				UserID:  &nilUUID,
			},
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

func TestGetTrackStreamUseCase_Execute(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	trackID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockClient := &mockCatalogTrackClient{
			getTrackStreamFunc: func(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error) {
				if req.TrackID != trackID {
					t.Errorf("expected trackID %s, got %s", trackID, req.TrackID)
				}
				if req.UserID == nil || *req.UserID != userID {
					t.Errorf("expected userID %s, got %v", userID, req.UserID)
				}
				return &grpcclient.TrackStreamResponse{
					StreamURL: "http://example.com/audio.mp3",
					IsPreview: false,
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}, nil
			},
		}

		uc := track.NewGetTrackStreamUseCase(mockClient, discardLogger)
		output, err := uc.Execute(context.Background(), track.GetTrackStreamInput{
			TrackID: trackID,
			UserID:  &userID,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output.StreamURL != "http://example.com/audio.mp3" {
			t.Errorf("expected stream url http://example.com/audio.mp3, got %s", output.StreamURL)
		}
		if output.IsPreview {
			t.Errorf("expected is_preview = false, got true")
		}
	})

	t.Run("catalog client error", func(t *testing.T) {
		mockClient := &mockCatalogTrackClient{
			getTrackStreamFunc: func(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error) {
				return nil, errors.New("catalog gRPC error")
			},
		}

		uc := track.NewGetTrackStreamUseCase(mockClient, discardLogger)
		output, err := uc.Execute(context.Background(), track.GetTrackStreamInput{
			TrackID: trackID,
			UserID:  &userID,
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if output != nil {
			t.Errorf("expected nil output, got %v", output)
		}
	})
}
