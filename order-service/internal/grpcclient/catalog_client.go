package grpcclient

import (
	"context"
	"fmt"
	"time"

	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/Lama189/soundwave-platform/order-service/internal/domain"
	"github.com/Lama189/soundwave-platform/order-service/internal/service/order"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type CatalogClient struct {
	tracks  catalogpb.TrackServiceClient
	albums  catalogpb.AlbumServiceClient
	timeout time.Duration
}

var _ order.CatalogClient = (*CatalogClient)(nil)

func NewCatalogClient(conn grpc.ClientConnInterface, defaultTimeout time.Duration) *CatalogClient {
	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &CatalogClient{
		tracks:  catalogpb.NewTrackServiceClient(conn),
		albums:  catalogpb.NewAlbumServiceClient(conn),
		timeout: defaultTimeout,
	}
}

func (c *CatalogClient) GetTracksByIDs(ctx context.Context, ids []uuid.UUID) ([]order.CatalogItemInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rawIDs := make([]string, len(ids))
	for i, id := range ids {
		rawIDs[i] = id.String()
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.tracks.GetTracks(ctx, &catalogpb.GetTracksRequest{TrackIds: rawIDs})
	if err != nil {
		return nil, fmt.Errorf("rpc GetTracks: %w", err)
	}

	items := make([]order.CatalogItemInfo, 0, len(res.GetTracks()))
	for _, t := range res.GetTracks() {
		trackID, err := uuid.Parse(t.GetId())
		if err != nil {
			return nil, fmt.Errorf("parse track id %q: %w", t.GetId(), err)
		}
		artistID, err := uuid.Parse(t.GetArtistId())
		if err != nil {
			return nil, fmt.Errorf("parse artist id %q: %w", t.GetArtistId(), err)
		}

		items = append(items, order.CatalogItemInfo{
			ID:       trackID,
			ArtistID: artistID,
			Price:    t.GetPrice(),
			Currency: domain.Currency(t.GetCurrency()),
			Status:   t.GetStatus(),
		})
	}

	return items, nil
}

func (c *CatalogClient) GetAlbumsByIDs(ctx context.Context, ids []uuid.UUID) ([]order.CatalogItemInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rawIDs := make([]string, len(ids))
	for i, id := range ids {
		rawIDs[i] = id.String()
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.albums.GetAlbums(ctx, &catalogpb.GetAlbumsRequest{AlbumIds: rawIDs})
	if err != nil {
		return nil, fmt.Errorf("rpc GetAlbums: %w", err)
	}

	items := make([]order.CatalogItemInfo, 0, len(res.GetAlbums()))
	for _, a := range res.GetAlbums() {
		albumID, err := uuid.Parse(a.GetId())
		if err != nil {
			return nil, fmt.Errorf("parse album id %q: %w", a.GetId(), err)
		}
		artistID, err := uuid.Parse(a.GetArtistId())
		if err != nil {
			return nil, fmt.Errorf("parse artist id %q: %w", a.GetArtistId(), err)
		}

		items = append(items, order.CatalogItemInfo{
			ID:       albumID,
			ArtistID: artistID,
			Price:    a.GetPrice(),
			Currency: domain.Currency(a.GetCurrency()),
			Status:   "ready",
		})
	}

	return items, nil
}
