package v1

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/playlist"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PlaylistGRPCServer struct {
	catalogpb.UnimplementedPlaylistServiceServer
	service PlaylistService
}

func NewPlaylistGRPCServer(service PlaylistService) *PlaylistGRPCServer {
	return &PlaylistGRPCServer{service: service}
}

func (s *PlaylistGRPCServer) CreatePlaylist(ctx context.Context, req *catalogpb.CreatePlaylistRequest) (*catalogpb.CreatePlaylistResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	dto := playlist.CreatePlaylistDTO{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		IsPrivate:   req.GetIsPrivate(),
	}

	p, err := s.service.Create(ctx, userID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.CreatePlaylistResponse{
		Playlist: toProtoPlaylist(p),
	}, nil
}

func (s *PlaylistGRPCServer) GetPlaylist(ctx context.Context, req *catalogpb.GetPlaylistRequest) (*catalogpb.GetPlaylistResponse, error) {
	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	var currentUserID *uuid.UUID
	if req.GetUserId() != "" {
		id, err := uuid.Parse(req.GetUserId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user id format")
		}
		currentUserID = &id
	}

	res, err := s.service.GetByID(ctx, currentUserID, playlistID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.GetPlaylistResponse{
		Playlist: toProtoPlaylist(&res.Playlist),
		Tracks:   toProtoPlaylistTracks(res.Tracks),
	}, nil
}

func (s *PlaylistGRPCServer) GetMyPlaylists(ctx context.Context, req *catalogpb.GetMyPlaylistsRequest) (*catalogpb.GetMyPlaylistsResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	limit := req.GetLimit()
	if limit <= 0 {
		limit = 20
	}

	playlists, err := s.service.GetMyPlaylists(ctx, userID, int(limit), int(req.GetOffset()))
	if err != nil {
		return nil, mapDomainError(err)
	}

	pbPlaylists := make([]*catalogpb.Playlist, len(playlists))
	for i, p := range playlists {
		pbPlaylists[i] = toProtoPlaylist(p)
	}

	return &catalogpb.GetMyPlaylistsResponse{
		Playlists: pbPlaylists,
	}, nil
}

func (s *PlaylistGRPCServer) AddTrackToPlaylist(ctx context.Context, req *catalogpb.AddTrackToPlaylistRequest) (*catalogpb.AddTrackToPlaylistResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	if err := s.service.AddTrack(ctx, userID, playlistID, trackID, int(req.GetPosition())); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.AddTrackToPlaylistResponse{}, nil
}

func (s *PlaylistGRPCServer) RemoveTrackFromPlaylist(ctx context.Context, req *catalogpb.RemoveTrackFromPlaylistRequest) (*catalogpb.RemoveTrackFromPlaylistResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	if err := s.service.RemoveTrack(ctx, userID, playlistID, trackID); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.RemoveTrackFromPlaylistResponse{}, nil
}

func (s *PlaylistGRPCServer) ChangePlaylistTrackPosition(ctx context.Context, req *catalogpb.ChangePlaylistTrackPositionRequest) (*catalogpb.ChangePlaylistTrackPositionResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	if err := s.service.ChangeTrackPosition(ctx, userID, playlistID, trackID, int(req.GetNewPosition())); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.ChangePlaylistTrackPositionResponse{}, nil
}

func (s *PlaylistGRPCServer) UpdatePlaylist(ctx context.Context, req *catalogpb.UpdatePlaylistRequest) (*catalogpb.UpdatePlaylistResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	dto := playlist.UpdatePlaylistDTO{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		IsPrivate:   req.GetIsPrivate(),
	}

	p, err := s.service.Update(ctx, userID, playlistID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.UpdatePlaylistResponse{
		Playlist: toProtoPlaylist(p),
	}, nil
}

func (s *PlaylistGRPCServer) DeletePlaylist(ctx context.Context, req *catalogpb.DeletePlaylistRequest) (*catalogpb.DeletePlaylistResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	playlistID, err := uuid.Parse(req.GetPlaylistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid playlist id format")
	}

	if err := s.service.Delete(ctx, userID, playlistID); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.DeletePlaylistResponse{}, nil
}

func toProtoPlaylist(p *playlist.PlaylistOutputDTO) *catalogpb.Playlist {
	if p == nil {
		return nil
	}

	return &catalogpb.Playlist{
		Id:          p.ID.String(),
		UserId:      p.UserID.String(),
		Title:       p.Title,
		Description: p.Description,
		IsPrivate:   p.IsPrivate,
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}
}

func toProtoPlaylistTracks(tracks []playlist.PlaylistTrackItemDTO) []*catalogpb.PlaylistTrack {
	if len(tracks) == 0 {
		return []*catalogpb.PlaylistTrack{}
	}

	res := make([]*catalogpb.PlaylistTrack, len(tracks))
	for i, t := range tracks {
		res[i] = &catalogpb.PlaylistTrack{
			TrackId:         t.TrackID.String(),
			Position:        int32(t.Position),
			AddedAt:         timestamppb.New(t.AddedAt),
			Title:           t.Title,
			ArtistId:        t.ArtistID.String(),
			DurationSeconds: int64(t.Duration.Seconds()),
			CoverKey:        t.CoverKey,
			Status:          t.Status,
		}
	}

	return res
}
