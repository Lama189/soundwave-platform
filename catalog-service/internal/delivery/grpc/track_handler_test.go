package v1_test

import (
	"context"
	"testing"
	"time"

	deliveryGrpc "github.com/Lama189/soundwave-platform/catalog-service/internal/delivery/grpc"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockTrackService struct {
	getTrackStreamFn func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error)
}

func (m *mockTrackService) UploadTrack(ctx context.Context, userID uuid.UUID, dto track.UploadTrackDTO) (*track.UploadTrackOutputDTO, error) {
	return nil, nil
}
func (m *mockTrackService) ConfirmUpload(ctx context.Context, userID, trackID uuid.UUID) (*track.TrackOutputDTO, error) {
	return nil, nil
}
func (m *mockTrackService) CompleteProcessing(ctx context.Context, trackID uuid.UUID, dto track.CompleteProcessingDTO) (*track.TrackOutputDTO, error) {
	return nil, nil
}
func (m *mockTrackService) FailProcessing(ctx context.Context, trackID uuid.UUID) error {
	return nil
}
func (m *mockTrackService) GetByID(ctx context.Context, id uuid.UUID) (*track.TrackOutputDTO, error) {
	return nil, nil
}
func (m *mockTrackService) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*track.TrackOutputDTO, error) {
	return nil, nil
}
func (m *mockTrackService) GetTrackStream(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error) {
	if m.getTrackStreamFn != nil {
		return m.getTrackStreamFn(ctx, dto)
	}
	return nil, nil
}

func TestTrackGRPCServer_GetTrackStream(t *testing.T) {
	trackID := uuid.New()
	userID := uuid.New()
	expiresAt := time.Now().Add(15 * time.Minute)

	tests := []struct {
		name        string
		req         *catalogpb.GetTrackStreamRequest
		serviceFn   func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error)
		wantCode    codes.Code
		wantURL     string
		wantPreview bool
	}{
		{
			name: "success with user id",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: trackID.String(),
				UserId:  func() *string { s := userID.String(); return &s }(),
			},
			serviceFn: func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error) {
				if dto.TrackID != trackID {
					t.Errorf("got trackID %v, want %v", dto.TrackID, trackID)
				}
				if dto.UserID == nil || *dto.UserID != userID {
					t.Errorf("got userID %v, want %v", dto.UserID, userID)
				}
				return &track.TrackStreamOutputDTO{
					StreamURL: "https://minio.example.com/audio/full.mp3",
					IsPreview: false,
					ExpiresAt: expiresAt,
				}, nil
			},
			wantCode:    codes.OK,
			wantURL:     "https://minio.example.com/audio/full.mp3",
			wantPreview: false,
		},
		{
			name: "success without user id (guest)",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: trackID.String(),
			},
			serviceFn: func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error) {
				if dto.UserID != nil {
					t.Errorf("expected nil userID, got %v", dto.UserID)
				}
				return &track.TrackStreamOutputDTO{
					StreamURL: "https://minio.example.com/audio/preview.mp3",
					IsPreview: true,
					ExpiresAt: expiresAt,
				}, nil
			},
			wantCode:    codes.OK,
			wantURL:     "https://minio.example.com/audio/preview.mp3",
			wantPreview: true,
		},
		{
			name: "invalid track id format",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: "invalid-uuid",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid user id format",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: trackID.String(),
				UserId:  func() *string { s := "invalid-uuid"; return &s }(),
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "track not found domain error maps to NotFound code",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: trackID.String(),
			},
			serviceFn: func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error) {
				return nil, domain.ErrTrackNotFound
			},
			wantCode: codes.NotFound,
		},
		{
			name: "track not ready domain error maps to InvalidArgument code",
			req: &catalogpb.GetTrackStreamRequest{
				TrackId: trackID.String(),
			},
			serviceFn: func(ctx context.Context, dto track.GetTrackStreamDTO) (*track.TrackStreamOutputDTO, error) {
				return nil, domain.ErrTrackNotReady
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTrackService{getTrackStreamFn: tt.serviceFn}
			server := deliveryGrpc.NewTrackGRPCServer(svc)

			res, err := server.GetTrackStream(context.Background(), tt.req)
			st, _ := status.FromError(err)

			if st.Code() != tt.wantCode {
				t.Fatalf("status code = %v, want %v (err = %v)", st.Code(), tt.wantCode, err)
			}

			if tt.wantCode == codes.OK {
				if res == nil {
					t.Fatal("response should not be nil")
				}
				if res.GetStreamUrl() != tt.wantURL {
					t.Errorf("StreamUrl = %v, want %v", res.GetStreamUrl(), tt.wantURL)
				}
				if res.GetIsPreview() != tt.wantPreview {
					t.Errorf("IsPreview = %v, want %v", res.GetIsPreview(), tt.wantPreview)
				}
				if res.GetExpiresAt() == nil {
					t.Error("ExpiresAt should not be nil")
				}
			}
		})
	}
}
