package track

import (
	"context"
	"fmt"
	"log/slog"
)

type LikeTrackUseCase struct {
	catalogClient CatalogTrackClient
	logger        *slog.Logger
}

func NewLikeTrackUseCase(catalogClient CatalogTrackClient, logger *slog.Logger) *LikeTrackUseCase {
	return &LikeTrackUseCase{
		catalogClient: catalogClient,
		logger:        logger,
	}
}

func (uc *LikeTrackUseCase) Execute(ctx context.Context, input LikeTrackInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := uc.catalogClient.LikeTrack(ctx, input.UserID, input.TrackID); err != nil {
		uc.logger.Warn(
			"failed to like track",
			slog.String("user_id", input.UserID.String()),
			slog.String("track_id", input.TrackID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("like track: %w", err)
	}

	return nil
}
