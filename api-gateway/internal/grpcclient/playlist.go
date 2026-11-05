package grpcclient

import (
	"context"
	"time"

	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type PlaylistClient struct {
	grpcClient catalogpb.PlaylistServiceClient
	timeout    time.Duration
}

func NewPlaylistClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *PlaylistClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &PlaylistClient{
		grpcClient: catalogpb.NewPlaylistServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *PlaylistClient) CreatePlaylist(ctx context.Context, req CreatePlaylistRequest) (*PlaylistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.CreatePlaylist(ctx, &catalogpb.CreatePlaylistRequest{
		UserId:      req.UserID.String(),
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
	})
	if err != nil {
		return nil, err
	}

	return toPlaylistResponse(pbRes.GetPlaylist())
}

func (c *PlaylistClient) GetPlaylist(ctx context.Context, playlistID uuid.UUID, userID *uuid.UUID) (*PlaylistWithTracksResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var userIDStr *string
	if userID != nil {
		s := userID.String()
		userIDStr = &s
	}

	pbRes, err := c.grpcClient.GetPlaylist(ctx, &catalogpb.GetPlaylistRequest{
		PlaylistId: playlistID.String(),
		UserId:     userIDStr,
	})
	if err != nil {
		return nil, err
	}

	return toPlaylistWithTracksResponse(pbRes)
}

func (c *PlaylistClient) GetMyPlaylists(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*PlaylistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetMyPlaylists(ctx, &catalogpb.GetMyPlaylistsRequest{
		UserId: userID.String(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	playlists := make([]*PlaylistResponse, len(pbRes.GetPlaylists()))
	for i, p := range pbRes.GetPlaylists() {
		pl, err := toPlaylistResponse(p)
		if err != nil {
			return nil, err
		}
		playlists[i] = pl
	}

	return playlists, nil
}

func (c *PlaylistClient) AddTrackToPlaylist(ctx context.Context, req AddTrackToPlaylistRequest) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.AddTrackToPlaylist(ctx, &catalogpb.AddTrackToPlaylistRequest{
		UserId:     req.UserID.String(),
		PlaylistId: req.PlaylistID.String(),
		TrackId:    req.TrackID.String(),
		Position:   req.Position,
	})
	return err
}

func (c *PlaylistClient) RemoveTrackFromPlaylist(ctx context.Context, userID, playlistID, trackID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.RemoveTrackFromPlaylist(ctx, &catalogpb.RemoveTrackFromPlaylistRequest{
		UserId:     userID.String(),
		PlaylistId: playlistID.String(),
		TrackId:    trackID.String(),
	})
	return err
}

func (c *PlaylistClient) ChangeTrackPosition(ctx context.Context, req ChangePlaylistTrackPositionRequest) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.ChangePlaylistTrackPosition(ctx, &catalogpb.ChangePlaylistTrackPositionRequest{
		UserId:      req.UserID.String(),
		PlaylistId:  req.PlaylistID.String(),
		TrackId:     req.TrackID.String(),
		NewPosition: req.NewPosition,
	})
	return err
}

func (c *PlaylistClient) UpdatePlaylist(ctx context.Context, req UpdatePlaylistRequest) (*PlaylistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.UpdatePlaylist(ctx, &catalogpb.UpdatePlaylistRequest{
		UserId:      req.UserID.String(),
		PlaylistId:  req.PlaylistID.String(),
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
	})
	if err != nil {
		return nil, err
	}

	return toPlaylistResponse(pbRes.GetPlaylist())
}

func (c *PlaylistClient) DeletePlaylist(ctx context.Context, userID, playlistID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.DeletePlaylist(ctx, &catalogpb.DeletePlaylistRequest{
		UserId:     userID.String(),
		PlaylistId: playlistID.String(),
	})
	return err
}
