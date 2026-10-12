package track

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
)

type UploadTrackUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewUploadTrackUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *UploadTrackUseCase {
	return &UploadTrackUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *UploadTrackUseCase) Execute(ctx context.Context, input UploadTrackInput) (*grpcclient.UploadTrackResponse, error) {
	res, err := uc.catalogClient.UploadTrack(ctx, grpcclient.UploadTrackRequest{
		UserID:   input.UserID,
		AlbumID:  input.AlbumID,
		Title:    input.Title,
		FileName: input.FileName,
	})
	if err != nil {
		uc.logger.Warn(
			"failed to initiate track upload",
			slog.String("user_id", input.UserID.String()),
			slog.String("title", input.Title),
			slog.String("file_name", input.FileName),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("upload track: %w", err)
	}

	return res, nil
}
