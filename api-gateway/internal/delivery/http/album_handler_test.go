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

type mockAlbumClient struct {
	createAlbumFunc         func(ctx context.Context, req grpcclient.CreateAlbumRequest) (*grpcclient.AlbumResponse, error)
	getAlbumFunc            func(ctx context.Context, albumID uuid.UUID) (*grpcclient.AlbumResponse, error)
	getAlbumsByArtistIDFunc func(ctx context.Context, artistID uuid.UUID, limit, offset int32) ([]*grpcclient.AlbumResponse, error)
	updateAlbumFunc         func(ctx context.Context, req grpcclient.UpdateAlbumRequest) (*grpcclient.AlbumResponse, error)
	deleteAlbumFunc         func(ctx context.Context, albumID, userID uuid.UUID) error
}

func (m *mockAlbumClient) CreateAlbum(ctx context.Context, req grpcclient.CreateAlbumRequest) (*grpcclient.AlbumResponse, error) {
	if m.createAlbumFunc != nil {
		return m.createAlbumFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAlbumClient) GetAlbum(ctx context.Context, albumID uuid.UUID) (*grpcclient.AlbumResponse, error) {
	if m.getAlbumFunc != nil {
		return m.getAlbumFunc(ctx, albumID)
	}
	return nil, nil
}

func (m *mockAlbumClient) GetAlbumsByArtistID(ctx context.Context, artistID uuid.UUID, limit, offset int32) ([]*grpcclient.AlbumResponse, error) {
	if m.getAlbumsByArtistIDFunc != nil {
		return m.getAlbumsByArtistIDFunc(ctx, artistID, limit, offset)
	}
	return nil, nil
}

func (m *mockAlbumClient) UpdateAlbum(ctx context.Context, req grpcclient.UpdateAlbumRequest) (*grpcclient.AlbumResponse, error) {
	if m.updateAlbumFunc != nil {
		return m.updateAlbumFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAlbumClient) DeleteAlbum(ctx context.Context, albumID, userID uuid.UUID) error {
	if m.deleteAlbumFunc != nil {
		return m.deleteAlbumFunc(ctx, albumID, userID)
	}
	return nil
}

func TestAlbumHandler_CreateAlbum(t *testing.T) {
	userID := uuid.New()
	artistID := uuid.New()
	albumID := uuid.New()

	t.Run("success", func(t *testing.T) {
		client := &mockAlbumClient{
			createAlbumFunc: func(ctx context.Context, req grpcclient.CreateAlbumRequest) (*grpcclient.AlbumResponse, error) {
				return &grpcclient.AlbumResponse{
					ID:        albumID,
					ArtistID:  artistID,
					Title:     "New Album",
					Price:     500,
					Currency:  "USD",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		}

		handler := deliveryHttp.NewAlbumHandler(client)

		reqBody, _ := json.Marshal(map[string]any{
			"artist_id": artistID.String(),
			"title":     "New Album",
			"price":     500,
			"currency":  "USD",
		})

		req := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(reqBody))
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateAlbum(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		handler := deliveryHttp.NewAlbumHandler(&mockAlbumClient{})
		req := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()

		handler.CreateAlbum(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("missing title", func(t *testing.T) {
		handler := deliveryHttp.NewAlbumHandler(&mockAlbumClient{})

		reqBody, _ := json.Marshal(map[string]any{
			"artist_id": artistID.String(),
			"title":     "",
		})
		req := httptest.NewRequest(http.MethodPost, "/albums", bytes.NewReader(reqBody))
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateAlbum(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAlbumHandler_GetByID(t *testing.T) {
	albumID := uuid.New()
	artistID := uuid.New()

	t.Run("success", func(t *testing.T) {
		client := &mockAlbumClient{
			getAlbumFunc: func(ctx context.Context, id uuid.UUID) (*grpcclient.AlbumResponse, error) {
				return &grpcclient.AlbumResponse{
					ID:       albumID,
					ArtistID: artistID,
					Title:    "Album Title",
				}, nil
			},
		}

		handler := deliveryHttp.NewAlbumHandler(client)

		r := chi.NewRouter()
		r.Get("/albums/{id}", handler.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/albums/"+albumID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		handler := deliveryHttp.NewAlbumHandler(&mockAlbumClient{})

		r := chi.NewRouter()
		r.Get("/albums/{id}", handler.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/albums/not-a-uuid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}
