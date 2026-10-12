package track

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type CatalogTrackClient interface {
	UploadTrack(ctx context.Context, req grpcclient.UploadTrackRequest) (*grpcclient.UploadTrackResponse, error)
}
