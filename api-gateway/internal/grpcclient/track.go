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

func (c *TrackClient) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) (*TrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.ConfirmUpload(ctx, &catalogpb.ConfirmUploadRequest{
		TrackId: req.TrackID.String(),
		UserId:  req.UserID.String(),
	})
	if err != nil {
		return nil, err
	}

	trackRes, err := toTrackResponse(pbRes.GetTrack())
	if err != nil {
		return nil, fmt.Errorf("invalid track returned from gRPC: %w", err)
	}

	return trackRes, nil
}

func (c *TrackClient) CompleteProcessing(ctx context.Context, req CompleteProcessingRequest) (*TrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.CompleteProcessing(ctx, &catalogpb.CompleteProcessingRequest{
		TrackId:         req.TrackID.String(),
		DurationSeconds: req.DurationSeconds,
		FileKey:         req.FileKey,
		PreviewKey:      req.PreviewKey,
		CoverKey:        req.CoverKey,
	})
	if err != nil {
		return nil, err
	}

	trackRes, err := toTrackResponse(pbRes.GetTrack())
	if err != nil {
		return nil, fmt.Errorf("invalid track returned from gRPC: %w", err)
	}

	return trackRes, nil
}

func (c *TrackClient) FailProcessing(ctx context.Context, req FailProcessingRequest) (*FailProcessingResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.FailProcessing(ctx, &catalogpb.FailProcessingRequest{
		TrackId: req.TrackID.String(),
	})
	if err != nil {
		return nil, err
	}

	return &FailProcessingResponse{}, nil
}

func (c *TrackClient) GetTrackStream(ctx context.Context, req GetTrackStreamRequest) (*TrackStreamResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var userID *string
	if req.UserID != nil {
		s := req.UserID.String()
		userID = &s
	}

	streamRes, err := c.grpcClient.GetTrackStream(ctx, &catalogpb.GetTrackStreamRequest{
		TrackId: req.TrackID.String(),
		UserId:  userID,
	})
	if err != nil {
		return nil, err
	}

	return toTrackStreamResponse(streamRes), nil
}

func (c *TrackClient) GetTrack(ctx context.Context, trackID uuid.UUID) (*TrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetTrack(ctx, &catalogpb.GetTrackRequest{
		TrackId: trackID.String(),
	})
	if err != nil {
		return nil, err
	}

	return toTrackResponse(pbRes.GetTrack())
}

func (c *TrackClient) GetTracks(ctx context.Context, trackIDs []uuid.UUID) ([]*TrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	strIDs := make([]string, len(trackIDs))
	for i, id := range trackIDs {
		strIDs[i] = id.String()
	}

	pbRes, err := c.grpcClient.GetTracks(ctx, &catalogpb.GetTracksRequest{
		TrackIds: strIDs,
	})
	if err != nil {
		return nil, err
	}

	res := make([]*TrackResponse, len(pbRes.GetTracks()))
	for i, t := range pbRes.GetTracks() {
		tr, err := toTrackResponse(t)
		if err != nil {
			return nil, err
		}
		res[i] = tr
	}

	return res, nil
}

func (c *TrackClient) LikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.LikeTrack(ctx, &catalogpb.LikeTrackRequest{
		TrackId: trackID.String(),
		UserId:  userID.String(),
	})
	return err
}

func (c *TrackClient) UnlikeTrack(ctx context.Context, userID, trackID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.UnlikeTrack(ctx, &catalogpb.UnlikeTrackRequest{
		TrackId: trackID.String(),
		UserId:  userID.String(),
	})
	return err
}

func (c *TrackClient) GetLikedTracks(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*TrackResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetLikedTracks(ctx, &catalogpb.GetLikedTracksRequest{
		UserId: userID.String(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	tracks := make([]*TrackResponse, len(pbRes.GetTracks()))
	for i, t := range pbRes.GetTracks() {
		tr, err := toTrackResponse(t)
		if err != nil {
			return nil, err
		}
		tracks[i] = tr
	}

	return tracks, nil
}
