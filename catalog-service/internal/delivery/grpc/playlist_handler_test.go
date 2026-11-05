package v1_test

import (
	"context"
	"testing"
	"time"

	deliveryGrpc "github.com/Lama189/soundwave-platform/catalog-service/internal/delivery/grpc"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/playlist"
	catalogpb "github.com/Lama189/soundwave-platform/gen/go/catalog/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockPlaylistService struct {
	createFn              func(ctx context.Context, userID uuid.UUID, dto playlist.CreatePlaylistDTO) (*playlist.PlaylistOutputDTO, error)
	getByIDFn             func(ctx context.Context, currentUserID *uuid.UUID, playlistID uuid.UUID) (*playlist.PlaylistWithTracksDTO, error)
	getMyPlaylistsFn      func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*playlist.PlaylistOutputDTO, error)
	addTrackFn            func(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, position int) error
	removeTrackFn         func(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID) error
	changeTrackPositionFn func(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, newPosition int) error
	updateFn              func(ctx context.Context, currentUserID, playlistID uuid.UUID, dto playlist.UpdatePlaylistDTO) (*playlist.PlaylistOutputDTO, error)
	deleteFn              func(ctx context.Context, currentUserID, playlistID uuid.UUID) error
}

func (m *mockPlaylistService) Create(ctx context.Context, userID uuid.UUID, dto playlist.CreatePlaylistDTO) (*playlist.PlaylistOutputDTO, error) {
	if m.createFn != nil {
		return m.createFn(ctx, userID, dto)
	}
	return nil, nil
}
func (m *mockPlaylistService) GetByID(ctx context.Context, currentUserID *uuid.UUID, playlistID uuid.UUID) (*playlist.PlaylistWithTracksDTO, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, currentUserID, playlistID)
	}
	return nil, nil
}
func (m *mockPlaylistService) GetMyPlaylists(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*playlist.PlaylistOutputDTO, error) {
	if m.getMyPlaylistsFn != nil {
		return m.getMyPlaylistsFn(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockPlaylistService) AddTrack(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, position int) error {
	if m.addTrackFn != nil {
		return m.addTrackFn(ctx, currentUserID, playlistID, trackID, position)
	}
	return nil
}
func (m *mockPlaylistService) RemoveTrack(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID) error {
	if m.removeTrackFn != nil {
		return m.removeTrackFn(ctx, currentUserID, playlistID, trackID)
	}
	return nil
}
func (m *mockPlaylistService) ChangeTrackPosition(ctx context.Context, currentUserID, playlistID, trackID uuid.UUID, newPosition int) error {
	if m.changeTrackPositionFn != nil {
		return m.changeTrackPositionFn(ctx, currentUserID, playlistID, trackID, newPosition)
	}
	return nil
}
func (m *mockPlaylistService) Update(ctx context.Context, currentUserID, playlistID uuid.UUID, dto playlist.UpdatePlaylistDTO) (*playlist.PlaylistOutputDTO, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, currentUserID, playlistID, dto)
	}
	return nil, nil
}
func (m *mockPlaylistService) Delete(ctx context.Context, currentUserID, playlistID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, currentUserID, playlistID)
	}
	return nil
}

func TestPlaylistGRPCServer_CreateAndGet(t *testing.T) {
	userID := uuid.New()
	playlistID := uuid.New()
	now := time.Now()

	t.Run("create success", func(t *testing.T) {
		svc := &mockPlaylistService{
			createFn: func(ctx context.Context, uid uuid.UUID, dto playlist.CreatePlaylistDTO) (*playlist.PlaylistOutputDTO, error) {
				if uid != userID {
					t.Errorf("got userID %v, want %v", uid, userID)
				}
				return &playlist.PlaylistOutputDTO{
					ID:        playlistID,
					UserID:    userID,
					Title:     dto.Title,
					IsPrivate: dto.IsPrivate,
					CreatedAt: now,
				}, nil
			},
		}

		server := deliveryGrpc.NewPlaylistGRPCServer(svc)
		res, err := server.CreatePlaylist(context.Background(), &catalogpb.CreatePlaylistRequest{
			UserId:      userID.String(),
			Title:       "My Playlist",
			Description: "desc",
			IsPrivate:   true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.GetPlaylist().GetId() != playlistID.String() {
			t.Errorf("got %v, want %v", res.GetPlaylist().GetId(), playlistID)
		}
		if !res.GetPlaylist().GetIsPrivate() {
			t.Error("expected is_private true")
		}
	})

	t.Run("create invalid user id", func(t *testing.T) {
		server := deliveryGrpc.NewPlaylistGRPCServer(&mockPlaylistService{})
		_, err := server.CreatePlaylist(context.Background(), &catalogpb.CreatePlaylistRequest{
			UserId: "invalid-uuid",
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("got code %v, want InvalidArgument", st.Code())
		}
	})

	t.Run("get playlist private access error maps to PermissionDenied", func(t *testing.T) {
		svc := &mockPlaylistService{
			getByIDFn: func(ctx context.Context, uid *uuid.UUID, pid uuid.UUID) (*playlist.PlaylistWithTracksDTO, error) {
				return nil, domain.ErrPrivatePlaylistAccess
			},
		}

		server := deliveryGrpc.NewPlaylistGRPCServer(svc)
		_, err := server.GetPlaylist(context.Background(), &catalogpb.GetPlaylistRequest{
			PlaylistId: playlistID.String(),
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.PermissionDenied {
			t.Errorf("got code %v, want PermissionDenied", st.Code())
		}
	})
}

func TestPlaylistGRPCServer_TracksOperations(t *testing.T) {
	userID := uuid.New()
	playlistID := uuid.New()
	trackID := uuid.New()

	t.Run("add track conflict maps to AlreadyExists", func(t *testing.T) {
		svc := &mockPlaylistService{
			addTrackFn: func(ctx context.Context, uid, pid, tid uuid.UUID, pos int) error {
				return domain.ErrTrackAlreadyInPlaylist
			},
		}

		server := deliveryGrpc.NewPlaylistGRPCServer(svc)
		_, err := server.AddTrackToPlaylist(context.Background(), &catalogpb.AddTrackToPlaylistRequest{
			UserId:     userID.String(),
			PlaylistId: playlistID.String(),
			TrackId:    trackID.String(),
			Position:   1,
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.AlreadyExists {
			t.Errorf("got code %v, want AlreadyExists", st.Code())
		}
	})

	t.Run("remove track not found maps to NotFound", func(t *testing.T) {
		svc := &mockPlaylistService{
			removeTrackFn: func(ctx context.Context, uid, pid, tid uuid.UUID) error {
				return domain.ErrTrackNotInPlaylist
			},
		}

		server := deliveryGrpc.NewPlaylistGRPCServer(svc)
		_, err := server.RemoveTrackFromPlaylist(context.Background(), &catalogpb.RemoveTrackFromPlaylistRequest{
			UserId:     userID.String(),
			PlaylistId: playlistID.String(),
			TrackId:    trackID.String(),
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.NotFound {
			t.Errorf("got code %v, want NotFound", st.Code())
		}
	})

	t.Run("delete not owner maps to PermissionDenied", func(t *testing.T) {
		svc := &mockPlaylistService{
			deleteFn: func(ctx context.Context, uid, pid uuid.UUID) error {
				return domain.ErrNotPlaylistOwner
			},
		}

		server := deliveryGrpc.NewPlaylistGRPCServer(svc)
		_, err := server.DeletePlaylist(context.Background(), &catalogpb.DeletePlaylistRequest{
			UserId:     userID.String(),
			PlaylistId: playlistID.String(),
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.PermissionDenied {
			t.Errorf("got code %v, want PermissionDenied", st.Code())
		}
	})
}
