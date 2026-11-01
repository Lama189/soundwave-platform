package artist

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/google/uuid"
)

type CatalogArtistClient interface {
	Create(ctx context.Context, dto grpcclient.CreateArtistRequest) (*grpcclient.ArtistResponse, error)
	Delete(ctx context.Context, artistID, userID uuid.UUID) error
}

type UserRoleClient interface {
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) (*grpcclient.UserResponse, error)
}
