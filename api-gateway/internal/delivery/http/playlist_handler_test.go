package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	deliveryHttp "github.com/Lama189/soundwave-platform/api-gateway/internal/delivery/http"
	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/Lama189/soundwave-platform/api-gateway/pkg/contextutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockPlaylistClient struct {
	createFn              func(ctx context.Context, req grpcclient.CreatePlaylistRequest) (*grpcclient.PlaylistResponse, error)
	getFn                 func(ctx context.Context, playlistID uuid.UUID, userID *uuid.UUID) (*grpcclient.PlaylistWithTracksResponse, error)
	getMyFn               func(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*grpcclient.PlaylistResponse, error)
	addTrackFn            func(ctx context.Context, req grpcclient.AddTrackToPlaylistRequest) error
	removeTrackFn         func(ctx context.Context, userID, playlistID, trackID uuid.UUID) error
	changeTrackPositionFn func(ctx context.Context, req grpcclient.ChangePlaylistTrackPositionRequest) error
	updateFn              func(ctx context.Context, req grpcclient.UpdatePlaylistRequest) (*grpcclient.PlaylistResponse, error)
	deleteFn              func(ctx context.Context, userID, playlistID uuid.UUID) error
}

func (m *mockPlaylistClient) CreatePlaylist(ctx context.Context, req grpcclient.CreatePlaylistRequest) (*grpcclient.PlaylistResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, nil
}
func (m *mockPlaylistClient) GetPlaylist(ctx context.Context, playlistID uuid.UUID, userID *uuid.UUID) (*grpcclient.PlaylistWithTracksResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, playlistID, userID)
	}
	return nil, nil
}
func (m *mockPlaylistClient) GetMyPlaylists(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*grpcclient.PlaylistResponse, error) {
	if m.getMyFn != nil {
		return m.getMyFn(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockPlaylistClient) AddTrackToPlaylist(ctx context.Context, req grpcclient.AddTrackToPlaylistRequest) error {
	if m.addTrackFn != nil {
		return m.addTrackFn(ctx, req)
	}
	return nil
}
func (m *mockPlaylistClient) RemoveTrackFromPlaylist(ctx context.Context, userID, playlistID, trackID uuid.UUID) error {
	if m.removeTrackFn != nil {
		return m.removeTrackFn(ctx, userID, playlistID, trackID)
	}
	return nil
}
func (m *mockPlaylistClient) ChangeTrackPosition(ctx context.Context, req grpcclient.ChangePlaylistTrackPositionRequest) error {
	if m.changeTrackPositionFn != nil {
		return m.changeTrackPositionFn(ctx, req)
	}
	return nil
}
func (m *mockPlaylistClient) UpdatePlaylist(ctx context.Context, req grpcclient.UpdatePlaylistRequest) (*grpcclient.PlaylistResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil, nil
}
func (m *mockPlaylistClient) DeletePlaylist(ctx context.Context, userID, playlistID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, playlistID)
	}
	return nil
}

func TestPlaylistHandler_CreateAndGet(t *testing.T) {
	userID := uuid.New()
	playlistID := uuid.New()

	t.Run("create success", func(t *testing.T) {
		client := &mockPlaylistClient{
			createFn: func(ctx context.Context, req grpcclient.CreatePlaylistRequest) (*grpcclient.PlaylistResponse, error) {
				return &grpcclient.PlaylistResponse{
					ID:        playlistID,
					UserID:    userID,
					Title:     req.Title,
					IsPrivate: req.IsPrivate,
					CreatedAt: time.Now(),
				}, nil
			},
		}

		handler := deliveryHttp.NewPlaylistHandler(client)
		reqBody, _ := json.Marshal(map[string]any{
			"title":      "Chill Vibes",
			"is_private": false,
		})

		req := httptest.NewRequest(http.MethodPost, "/playlists", bytes.NewReader(reqBody))
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreatePlaylist(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("create unauthorized", func(t *testing.T) {
		handler := deliveryHttp.NewPlaylistHandler(&mockPlaylistClient{})
		req := httptest.NewRequest(http.MethodPost, "/playlists", bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()

		handler.CreatePlaylist(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("get by id success", func(t *testing.T) {
		client := &mockPlaylistClient{
			getFn: func(ctx context.Context, pid uuid.UUID, uid *uuid.UUID) (*grpcclient.PlaylistWithTracksResponse, error) {
				return &grpcclient.PlaylistWithTracksResponse{
					Playlist: grpcclient.PlaylistResponse{
						ID:    playlistID,
						Title: "Chill Vibes",
					},
					Tracks: []grpcclient.PlaylistTrackResponse{},
				}, nil
			},
		}

		handler := deliveryHttp.NewPlaylistHandler(client)
		r := chi.NewRouter()
		r.Get("/playlists/{id}", handler.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/playlists/"+playlistID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}

func TestPlaylistHandler_TrackOperations(t *testing.T) {
	userID := uuid.New()
	playlistID := uuid.New()
	trackID := uuid.New()

	t.Run("add track success", func(t *testing.T) {
		client := &mockPlaylistClient{
			addTrackFn: func(ctx context.Context, req grpcclient.AddTrackToPlaylistRequest) error {
				if req.UserID != userID || req.PlaylistID != playlistID || req.TrackID != trackID {
					t.Errorf("got u=%v, p=%v, tr=%v", req.UserID, req.PlaylistID, req.TrackID)
				}
				return nil
			},
		}

		handler := deliveryHttp.NewPlaylistHandler(client)
		r := chi.NewRouter()
		r.Post("/playlists/{id}/tracks", handler.AddTrack)

		reqBody, _ := json.Marshal(map[string]any{
			"track_id": trackID.String(),
			"position": 1,
		})
		req := httptest.NewRequest(http.MethodPost, "/playlists/"+playlistID.String()+"/tracks", bytes.NewReader(reqBody))
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("remove track success", func(t *testing.T) {
		client := &mockPlaylistClient{
			removeTrackFn: func(ctx context.Context, uid, pid, tid uuid.UUID) error {
				if uid != userID || pid != playlistID || tid != trackID {
					t.Errorf("got u=%v, p=%v, tr=%v", uid, pid, tid)
				}
				return nil
			},
		}

		handler := deliveryHttp.NewPlaylistHandler(client)
		r := chi.NewRouter()
		r.Delete("/playlists/{id}/tracks/{track_id}", handler.RemoveTrack)

		req := httptest.NewRequest(http.MethodDelete, "/playlists/"+playlistID.String()+"/tracks/"+trackID.String(), nil)
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("delete playlist success", func(t *testing.T) {
		client := &mockPlaylistClient{
			deleteFn: func(ctx context.Context, uid, pid uuid.UUID) error {
				return nil
			},
		}

		handler := deliveryHttp.NewPlaylistHandler(client)
		r := chi.NewRouter()
		r.Delete("/playlists/{id}", handler.DeletePlaylist)

		req := httptest.NewRequest(http.MethodDelete, "/playlists/"+playlistID.String(), nil)
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}
