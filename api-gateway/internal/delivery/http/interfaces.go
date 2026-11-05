package http

import (
	"context"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	usecaseArtist "github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/artist"
	usecaseOrder "github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/order"
	usecaseTrack "github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/track"
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

type GetTrackStreamUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.GetTrackStreamInput) (*usecaseTrack.GetTrackStreamOutput, error)
}

type GetTrackUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.GetTrackInput) (*grpcclient.TrackResponse, error)
}

type LikeTrackUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.LikeTrackInput) error
}

type UnlikeTrackUseCase interface {
	Execute(ctx context.Context, input usecaseTrack.UnlikeTrackInput) error
}

type AlbumClient interface {
	CreateAlbum(ctx context.Context, req grpcclient.CreateAlbumRequest) (*grpcclient.AlbumResponse, error)
	GetAlbum(ctx context.Context, albumID uuid.UUID) (*grpcclient.AlbumResponse, error)
	GetAlbumsByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int32) ([]*grpcclient.AlbumResponse, error)
	UpdateAlbum(ctx context.Context, req grpcclient.UpdateAlbumRequest) (*grpcclient.AlbumResponse, error)
	DeleteAlbum(ctx context.Context, albumID, userID uuid.UUID) error
}

type PlaylistClient interface {
	CreatePlaylist(ctx context.Context, req grpcclient.CreatePlaylistRequest) (*grpcclient.PlaylistResponse, error)
	GetPlaylist(ctx context.Context, playlistID uuid.UUID, userID *uuid.UUID) (*grpcclient.PlaylistWithTracksResponse, error)
	GetMyPlaylists(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*grpcclient.PlaylistResponse, error)
	AddTrackToPlaylist(ctx context.Context, req grpcclient.AddTrackToPlaylistRequest) error
	RemoveTrackFromPlaylist(ctx context.Context, userID, playlistID, trackID uuid.UUID) error
	ChangeTrackPosition(ctx context.Context, req grpcclient.ChangePlaylistTrackPositionRequest) error
	UpdatePlaylist(ctx context.Context, req grpcclient.UpdatePlaylistRequest) (*grpcclient.PlaylistResponse, error)
	DeletePlaylist(ctx context.Context, userID, playlistID uuid.UUID) error
}

type CreateOrderUseCase interface {
	Execute(ctx context.Context, input usecaseOrder.CreateOrderInput) (*usecaseOrder.OrderOutput, error)
}
