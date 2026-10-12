package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type ConfirmUploadUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewConfirmUploadUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *ConfirmUploadUseCase {
	return &ConfirmUploadUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *ConfirmUploadUseCase) Execute(ctx context.Context, input ConfirmUploadInput) (*grpcclient.TrackResponse, error) {
	res, err := uc.catalogClient.ConfirmUpload(ctx, grpcclient.ConfirmUploadRequest{
		TrackID: input.TrackID,
		UserID:  input.UserID,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to confirm track upload",
			slog.String("track_id", input.TrackID.String()),
			slog.String("user_id", input.UserID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("confirm track upload: %w", err)
	}

	return res, nil
}
