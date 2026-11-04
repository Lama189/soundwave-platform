package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type GetTrackStreamUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewGetTrackStreamUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *GetTrackStreamUseCase {
	return &GetTrackStreamUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *GetTrackStreamUseCase) Execute(ctx context.Context, input GetTrackStreamInput) (*GetTrackStreamOutput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	res, err := uc.catalogClient.GetTrackStream(ctx, grpcclient.GetTrackStreamRequest{
		TrackID: input.TrackID,
		UserID:  input.UserID,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to get track stream",
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("get track stream: %w", err)
	}

	return toTrackStreamOutput(res), nil
}
