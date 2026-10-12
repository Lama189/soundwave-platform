package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type CompleteProcessingUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewCompleteProcessingUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *CompleteProcessingUseCase {
	return &CompleteProcessingUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *CompleteProcessingUseCase) Execute(ctx context.Context, input CompleteProcessingInput) (*grpcclient.TrackResponse, error) {
	res, err := uc.catalogClient.CompleteProcessing(ctx, grpcclient.CompleteProcessingRequest{
		TrackID:         input.TrackID,
		DurationSeconds: input.DurationSeconds,
		FileKey:         input.FileKey,
		PreviewKey:      input.PreviewKey,
		CoverKey:        input.CoverKey,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to complete track processing",
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("complete track processing: %w", err)
	}

	return res, nil
}
