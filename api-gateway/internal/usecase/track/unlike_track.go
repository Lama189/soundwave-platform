package track

import (
	"context"
	"fmt"
	"log/slog"
)

type UnlikeTrackUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewUnlikeTrackUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *UnlikeTrackUseCase {
	return &UnlikeTrackUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *UnlikeTrackUseCase) Execute(ctx context.Context, input UnlikeTrackInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := uc.catalogClient.UnlikeTrack(ctx, input.UserID, input.TrackID); err != nil {
		uc.logger.Warn(
			"failed to unlike track",
			slog.String("user_id", input.UserID.String()),
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("unlike track: %w", err)
	}

	return nil
}
