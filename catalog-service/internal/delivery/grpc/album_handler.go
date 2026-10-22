package v1

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/album"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AlbumGRPCServer struct {
	catalogpb.UnimplementedAlbumServiceServer
	service AlbumService
}

func NewAlbumGRPCServer(service AlbumService) *AlbumGRPCServer {
	return &AlbumGRPCServer{service: service}
}

func (s *AlbumGRPCServer) CreateAlbum(ctx context.Context, req *catalogpb.CreateAlbumRequest) (*catalogpb.CreateAlbumResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	artistID, err := uuid.Parse(req.GetArtistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid artist id format")
	}

	dto := album.CreateAlbumDTO{
		ArtistID:    artistID,
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		CoverKey:    req.GetCoverKey(),
		Price:       req.GetPrice(),
		Currency:    domain.Currency(req.GetCurrency()),
	}

	a, err := s.service.Create(ctx, userID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.CreateAlbumResponse{
		Album: toProtoAlbum(a),
	}, nil
}

func (s *AlbumGRPCServer) GetAlbum(ctx context.Context, req *catalogpb.GetAlbumRequest) (*catalogpb.GetAlbumResponse, error) {
	albumID, err := uuid.Parse(req.GetAlbumId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid album id format")
	}

	a, err := s.service.GetByID(ctx, albumID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.GetAlbumResponse{
		Album: toProtoAlbum(a),
	}, nil
}

func (s *AlbumGRPCServer) GetAlbums(ctx context.Context, req *catalogpb.GetAlbumsRequest) (*catalogpb.GetAlbumsResponse, error) {
	rawIDs := req.GetAlbumIds()
	if len(rawIDs) == 0 {
		return &catalogpb.GetAlbumsResponse{Albums: []*catalogpb.Album{}}, nil
	}

	albumIDs := make([]uuid.UUID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid album id format %q: %v", rawID, err)
		}
		albumIDs = append(albumIDs, id)
	}

	albums, err := s.service.GetByIDs(ctx, albumIDs)
	if err != nil {
		return nil, mapDomainError(err)
	}

	pbAlbums := make([]*catalogpb.Album, len(albums))
	for i, a := range albums {
		pbAlbums[i] = toProtoAlbum(a)
	}

	return &catalogpb.GetAlbumsResponse{
		Albums: pbAlbums,
	}, nil
}

func (s *AlbumGRPCServer) GetAlbumsByArtistId(ctx context.Context, req *catalogpb.GetAlbumsByArtistIdRequest) (*catalogpb.GetAlbumsByArtistIdResponse, error) {
	artistID, err := uuid.Parse(req.GetArtistId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid artist id format")
	}

	albums, err := s.service.GetByArtistID(ctx, artistID, int(req.GetLimit()), int(req.GetOffset()))
	if err != nil {
		return nil, mapDomainError(err)
	}

	pbAlbums := make([]*catalogpb.Album, len(albums))
	for i, a := range albums {
		pbAlbums[i] = toProtoAlbum(a)
	}

	return &catalogpb.GetAlbumsByArtistIdResponse{
		Albums: pbAlbums,
	}, nil
}

func (s *AlbumGRPCServer) UpdateAlbum(ctx context.Context, req *catalogpb.UpdateAlbumRequest) (*catalogpb.UpdateAlbumResponse, error) {
	albumID, err := uuid.Parse(req.GetAlbumId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid album id format")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	dto := album.UpdateAlbumDTO{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		CoverKey:    req.GetCoverKey(),
		Price:       req.GetPrice(),
		Currency:    domain.Currency(req.GetCurrency()),
	}

	a, err := s.service.Update(ctx, userID, albumID, dto)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.UpdateAlbumResponse{
		Album: toProtoAlbum(a),
	}, nil
}

func (s *AlbumGRPCServer) DeleteAlbum(ctx context.Context, req *catalogpb.DeleteAlbumRequest) (*catalogpb.DeleteAlbumResponse, error) {
	albumID, err := uuid.Parse(req.GetAlbumId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid album id format")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	if err := s.service.Delete(ctx, userID, albumID); err != nil {
		return nil, mapDomainError(err)
	}

	return &catalogpb.DeleteAlbumResponse{}, nil
}

func toProtoAlbum(a *album.AlbumOutputDTO) *catalogpb.Album {
	if a == nil {
		return nil
	}

	return &catalogpb.Album{
		Id:          a.ID.String(),
		ArtistId:    a.ArtistID.String(),
		Title:       a.Title,
		Description: a.Description,
		CoverKey:    a.CoverKey,
		Price:       a.Price,
		Currency:    string(a.Currency),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
}
