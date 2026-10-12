package track

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type CatalogTrackClient interface {
	UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error)
	ConfirmUpload(ctx context.Context, req grpcclient.ConfirmUploadRequest) (*grpcclient.TrackResponse, error)
	CompleteProcessing(ctx context.Context, req grpcclient.CompleteProcessingRequest) (*grpcclient.TrackResponse, error)
	FailProcessing(ctx context.Context, req grpcclient.FailProcessingRequest) (*grpcclient.FailProcessingResponse, error)
}
