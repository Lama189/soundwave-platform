package track

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/google/uuid"
)

type CatalogTrackClient interface {
	UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error)
	ConfirmUpload(ctx context.Context, req grpcclient.ConfirmUploadRequest) (*grpcclient.TrackResponse, error)
	CompleteProcessing(ctx context.Context, req grpcclient.CompleteProcessingRequest) (*grpcclient.TrackResponse, error)
	FailProcessing(ctx context.Context, req grpcclient.FailProcessingRequest) (*grpcclient.FailProcessingResponse, error)
	GetTrackStream(ctx context.Context, req grpcclient.GetTrackStreamRequest) (*grpcclient.TrackStreamResponse, error)
	GetTrack(ctx context.Context, trackID uuid.UUID) (*grpcclient.TrackResponse, error)
	LikeTrack(ctx context.Context, userID, trackID uuid.UUID) error
	UnlikeTrack(ctx context.Context, userID, trackID uuid.UUID) error
}
