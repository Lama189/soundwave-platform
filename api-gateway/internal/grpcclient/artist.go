package grpcclient

import (
	"context"
	"fmt"
	"time"

	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type ArtistClient struct {
	grpcClient catalogpb.ArtistServiceClient
	timeout    time.Duration
}

func NewArtistClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *ArtistClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &ArtistClient{
		grpcClient: catalogpb.NewArtistServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *ArtistClient) Create(ctx context.Context, dto CreateArtistRequest) (*ArtistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.CreateArtist(ctx, &catalogpb.CreateArtistRequest{
		UserId: dto.UserID.String(),
		Name:   dto.Name,
		Bio:    dto.Bio,
	})
	if err != nil {
		return nil, err
	}

	return toArtistResponse(pbRes.GetArtist())
}

func (c *ArtistClient) GetArtist(ctx context.Context, artistID uuid.UUID) (*ArtistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetArtist(ctx, &catalogpb.GetArtistRequest{
		Id: artistID.String(),
	})
	if err != nil {
		return nil, err
	}

	return toArtistResponse(pbRes.GetArtist())
}

func (c *ArtistClient) GetArtistByUserId(ctx context.Context, userID uuid.UUID) (*ArtistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetArtistByUserId(ctx, &catalogpb.GetArtistByUserIdRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}

	return toArtistResponse(pbRes.GetArtist())
}

func (c *ArtistClient) UpdateArtist(ctx context.Context, req UpdateArtistRequest) (*ArtistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.UpdateArtist(ctx, &catalogpb.UpdateArtistRequest{
		Id:     req.ID.String(),
		UserId: req.UserID.String(),
		Name:   req.Name,
		Bio:    req.Bio,
	})
	if err != nil {
		return nil, err
	}

	return toArtistResponse(pbRes.GetArtist())
}

func (c *ArtistClient) Delete(ctx context.Context, artistID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.DeleteArtist(ctx, &catalogpb.DeleteArtistRequest{
		Id:     artistID.String(),
		UserId: userID.String(),
	})
	return err
}

func toArtistResponse(pb *catalogpb.Artist) (*ArtistResponse, error) {
	if pb == nil {
		return nil, nil
	}

	id, err := uuid.Parse(pb.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid artist id format: %w", err)
	}

	userID, err := uuid.Parse(pb.GetUserId())
	if err != nil {
		return nil, fmt.Errorf("invalid user id format: %w", err)
	}

	return &ArtistResponse{
		ID:        id,
		UserID:    userID,
		Name:      pb.GetName(),
		Bio:       pb.GetBio(),
		CreatedAt: pb.GetCreatedAt().AsTime(),
		UpdatedAt: pb.GetUpdatedAt().AsTime(),
	}, nil
}
