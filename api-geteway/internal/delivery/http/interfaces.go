package http

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/grpcclient"
	usecaseArtist "github.com/Lama189/soundwave-platform/api-geteway/internal/usecase/artist"
	usecaseTrack "github.com/Lama189/soundwave-platform/api-geteway/internal/usecase/track"
	"github.com/google/uuid"
)

type UserClient interface {
	Register(ctx context.Context, req grpcclient.RegisterRequest) (*grpcclient.UserResponse, error)
	Login(ctx context.Context, req grpcclient.LoginRequest) (*grpcclient.AuthResponse, error)
	Refresh(ctx context.Context, req grpcclient.RefreshRequest) (*grpcclient.RefreshResponse, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*grpcclient.UserResponse, error)
}

type ArtistClient interface {
	GetArtist(ctx context.Context, artistID uuid.UUID) (*grpcclient.ArtistResponse, error)
	GetArtistByUserId(ctx context.Context, userID uuid.UUID) (*grpcclient.ArtistResponse, error)
}

type BecomeArtistUseCase interface {
	Execute(ctx context.Context, input usecaseArtist.BecomeArtistInput) (*grpcclient.ArtistResponse, error)
}

type UploadTrackUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.UploadTrackInput) (*grpcclient.UploadTrackResponse, error)
}

type ConfirmUploadUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.ConfirmUploadInput) (*grpcclient.TrackResponse, error)
}

type CompleteProcessingUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.CompleteProcessingInput) (*grpcclient.TrackResponse, error)
}

type FailProcessingUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.FailProcessingInput) error
}
