package grpcclient

import (
	"context"
	"time"

	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type AlbumClient struct {
	grpcClient catalogpb.AlbumServiceClient
	timeout    time.Duration
}

func NewAlbumClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *AlbumClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &AlbumClient{
		grpcClient: catalogpb.NewAlbumServiceClient(conn),
		timeout:    defaultTimeout,
	}
}

func (c *AlbumClient) CreateAlbum(ctx context.Context, req CreateAlbumRequest) (*AlbumResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.CreateAlbum(ctx, &catalogpb.CreateAlbumRequest{
		ArtistId:    req.ArtistID.String(),
		Title:       req.Title,
		Description: req.Description,
		CoverKey:    req.CoverKey,
		Price:       req.Price,
		Currency:    req.Currency,
		UserId:      req.UserID.String(),
	})
	if err != nil {
		return nil, err
	}

	return toAlbumResponse(pbRes.GetAlbum())
}

func (c *AlbumClient) GetAlbum(ctx context.Context, albumID uuid.UUID) (*AlbumResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetAlbum(ctx, &catalogpb.GetAlbumRequest{
		AlbumId: albumID.String(),
	})
	if err != nil {
		return nil, err
	}

	return toAlbumResponse(pbRes.GetAlbum())
}

func (c *AlbumClient) GetAlbumsByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int32) ([]*AlbumResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.GetAlbumsByArtistId(ctx, &catalogpb.GetAlbumsByArtistIdRequest{
		ArtistId: artistID.String(),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	albums := make([]*AlbumResponse, len(pbRes.GetAlbums()))
	for i, a := range pbRes.GetAlbums() {
		alb, err := toAlbumResponse(a)
		if err != nil {
			return nil, err
		}
		albums[i] = alb
	}

	return albums, nil
}

func (c *AlbumClient) UpdateAlbum(ctx context.Context, req UpdateAlbumRequest) (*AlbumResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	pbRes, err := c.grpcClient.UpdateAlbum(ctx, &catalogpb.UpdateAlbumRequest{
		AlbumId:     req.AlbumID.String(),
		UserId:      req.UserID.String(),
		Title:       req.Title,
		Description: req.Description,
		CoverKey:    req.CoverKey,
		Price:       req.Price,
		Currency:    req.Currency,
	})
	if err != nil {
		return nil, err
	}

	return toAlbumResponse(pbRes.GetAlbum())
}

func (c *AlbumClient) DeleteAlbum(ctx context.Context, albumID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.grpcClient.DeleteAlbum(ctx, &catalogpb.DeleteAlbumRequest{
		AlbumId: albumID.String(),
		UserId:  userID.String(),
	})
	return err
}
