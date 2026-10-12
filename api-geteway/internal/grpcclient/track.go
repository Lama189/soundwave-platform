package grpcclient

import (
	"context"
	"fmt"
	"time"

	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type TrackClient struct {
	grpcClient catalogpb.TrackServiceClient
	timeout    time.Duration
}

func NewTrackClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *TrackClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &TrackClient{
		grpcClient: catalogpb.NewTrackServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *TrackClient) UploadTrack(ctx context.Context, req UploadTrackRequest) (*UploadTrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbReq := &catalogpb.UploadTrackRequest{
		UserId:   req.UserID.String(),
		Title:    req.Title,
		FileName: req.FileName,
	}

	if req.AlbumID != nil && *req.AlbumID != uuid.Nil {
		albumIDStr := req.AlbumID.String()
		pbReq.AlbumId = &albumIDStr
	}

	pbRes, err := c.grpcClient.UploadTrack(ctx, pbReq)
	if err != nil {
		return nil, err
	}

	trackRes, err := toTrackResponse(pbRes.GetTrack())
	if err != nil {
		return nil, fmt.Errorf("invalid track returned from gRPC: %w", err)
	}

	return &UploadTrackResponse{
		Track:     *trackRes,
		UploadURL: pbRes.GetUploadUrl(),
	}, nil
}

func toTrackResponse(pb *catalogpb.Track) (*TrackResponse, error) {
	if pb == nil {
		return nil, nil
	}

	id, err := uuid.Parse(pb.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid track id format: %w", err)
	}

	artistID, err := uuid.Parse(pb.GetArtistId())
	if err != nil {
		return nil, fmt.Errorf("invalid artist id format: %w", err)
	}

	var albumID *uuid.UUID
	if pb.AlbumId != nil && *pb.AlbumId != "" {
		parsedAlbumID, err := uuid.Parse(*pb.AlbumId)
		if err != nil {
			return nil, fmt.Errorf("invalid album id format: %w", err)
		}
		albumID = &parsedAlbumID
	}

	var createdAt, updatedAt time.Time
	if pb.GetCreatedAt() != nil {
		createdAt = pb.GetCreatedAt().AsTime()
	}
	if pb.GetUpdatedAt() != nil {
		updatedAt = pb.GetUpdatedAt().AsTime()
	}

	return &TrackResponse{
		ID:              id,
		ArtistID:        artistID,
		AlbumID:         albumID,
		Title:           pb.GetTitle(),
		DurationSeconds: pb.GetDurationSeconds(),
		CoverKey:        pb.GetCoverKey(),
		FileKey:         pb.GetFileKey(),
		PreviewKey:      pb.GetPreviewKey(),
		RawKey:          pb.GetRawKey(),
		Status:          pb.GetStatus(),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}
