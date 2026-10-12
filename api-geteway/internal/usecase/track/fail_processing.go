package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type FailProcessingUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewFailProcessingUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *FailProcessingUseCase {
	return &FailProcessingUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *FailProcessingUseCase) Execute(ctx context.Context, input FailProcessingInput) error {
	_, err := uc.catalogClient.FailProcessing(ctx, grpcclient.FailProcessingRequest{
		TrackID: input.TrackID,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to fail track processing",
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("fail track processing: %w", err)
	}

	return nil
}
