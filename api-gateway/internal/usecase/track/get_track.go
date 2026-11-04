package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type GetTrackUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewGetTrackUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *GetTrackUseCase {
	return &GetTrackUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *GetTrackUseCase) Execute(ctx context.Context, input GetTrackInput) (*grpcclient.TrackResponse, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	track, err := uc.catalogClient.GetTrack(ctx, input.TrackID)
	if err != nil {
		uc.logger.Warn(
			"failed to get track",
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("get track: %w", err)
	}

	return track, nil
}
