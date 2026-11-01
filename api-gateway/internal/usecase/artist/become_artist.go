package artist

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type BecomeArtistUseCase struct {
	catalogClient CatalogArtistClient
	userClient    UserRoleClient
	logger        *slog.Logger
}

func NewBecomeArtistUseCase(catalogClient CatalogArtistClient, userClient UserRoleClient, logger *slog.Logger) *BecomeArtistUseCase {
	return &BecomeArtistUseCase{
		catalogClient: catalogClient,
		userClient:    userClient,
		logger:        logger,
	}
}

func (uc *BecomeArtistUseCase) Execute(ctx context.Context, input BecomeArtistInput) (*grpcclient.ArtistResponse, error) {
	artist, err := uc.catalogClient.Create(ctx, grpcclient.CreateArtistRequest{
		UserID: input.UserID,
		Name:   input.Name,
		Bio:    input.Bio,
	})
	if err != nil {
		return nil, fmt.Errorf("create artist profile: %w", err)
	}

	_, err = uc.userClient.UpdateUserRole(ctx, input.UserID, "artist")
	if err != nil {
		uc.logger.Warn(
			"failed to update role to artist, compensating by deleting artist profile",
			slog.String("user_id", input.UserID.String()),
			slog.String("artist_id", artist.ID.String()),
			slog.String("error", err.Error()),
		)

		compCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if compErr := uc.catalogClient.Delete(compCtx, artist.ID, input.UserID); compErr != nil {
			uc.logger.Error(
				"CRITICAL: compensation failed, orphaned artist profile remains",
				slog.String("user_id", input.UserID.String()),
				slog.String("artist_id", artist.ID.String()),
				slog.String("compensation_error", compErr.Error()),
			)
		}

		return nil, fmt.Errorf("update user role: %w", err)
	}

	return artist, nil
}
