package v1

import (
	"context"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TrackGRPCServer struct {
	catalogpb.UnimplementedTrackServiceServer
	service TrackService
}

func NewTrackGRPCServer(service TrackService) *TrackGRPCServer {
	return &TrackGRPCServer{
		service: service,
	}
}

func (s *TrackGRPCServer) UploadTrack(ctx context.Context, req *catalogpb.UploadTrackRequest) (*catalogpb.UploadTrackResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	var albumID *uuid.UUID
	if rawID := req.GetAlbumId(); rawID != "" {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid album id format")
		}
		albumID = &id
	}

	dto := track.UploadTrackDTO{
		AlbumID:  albumID,
		Title:    req.GetTitle(),
		FileName: req.GetFileName(),
	}

	res, err := s.service.UploadTrack(ctx, userID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.UploadTrackResponse{
		Track:     toProtoTrack(&res.Track),
		UploadUrl: res.UploadURL,
	}, nil
}

func (s *TrackGRPCServer) ConfirmUpload(ctx context.Context, req *catalogpb.ConfirmUploadRequest) (*catalogpb.ConfirmUploadResponse, error) {
	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	t, err := s.service.ConfirmUpload(ctx, userID, trackID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.ConfirmUploadResponse{
		Track: toProtoTrack(t),
	}, nil
}

func (s *TrackGRPCServer) CompleteProcessing(ctx context.Context, req *catalogpb.CompleteProcessingRequest) (*catalogpb.CompleteProcessingResponse, error) {
	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	dto := track.CompleteProcessingDTO{
		CoverKey:   req.GetCoverKey(),
		FileKey:    req.GetFileKey(),
		PreviewKey: req.GetPreviewKey(),
		Duration:   time.Duration(req.GetDurationSeconds()) * time.Second,
	}

	t, err := s.service.CompleteProcessing(ctx, trackID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.CompleteProcessingResponse{
		Track: toProtoTrack(t),
	}, nil
}

func (s *TrackGRPCServer) FailProcessing(ctx context.Context, req *catalogpb.FailProcessingRequest) (*catalogpb.FailProcessingResponse, error) {
	trackID, err := uuid.Parse(req.GetTrackId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid track id format")
	}

	if err := s.service.FailProcessing(ctx, trackID); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.FailProcessingResponse{}, nil
}

func toProtoTrack(t *track.TrackOutputDTO) *catalogpb.Track {
	if t == nil {
		return nil
	}

	pbTrack := &catalogpb.Track{
		Id:              t.ID.String(),
		ArtistId:        t.ArtistID.String(),
		Title:           t.Title,
		DurationSeconds: int64(t.Duration.Seconds()),
		CoverKey:        t.CoverKey,
		FileKey:         t.FileKey,
		PreviewKey:      t.PreviewKey,
		RawKey:          t.RawKey,
		Status:          t.Status,
		CreatedAt:       timestamppb.New(t.CreatedAt),
		UpdatedAt:       timestamppb.New(t.UpdatedAt),
	}

	if t.AlbumID != nil && *t.AlbumID != uuid.Nil {
		albumIDStr := t.AlbumID.String()
		pbTrack.AlbumId = &albumIDStr
	}

	return pbTrack
}
