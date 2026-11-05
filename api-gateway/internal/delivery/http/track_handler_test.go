package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	deliveryHttp "github.com/Lama189/soundwave-platform/api-gateway/internal/delivery/http"
	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	usecaseTrack "github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/track"
	"github.com/Lama189/soundwave-platform/api-gateway/pkg/contextutil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockGetTrackUseCase struct {
	executeFunc func(ctx context.Context, input usecaseTrack.GetTrackInput) (*grpcclient.TrackResponse, error)
}

func (m *mockGetTrackUseCase) Execute(ctx context.Context, input usecaseTrack.GetTrackInput) (*grpcclient.TrackResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return nil, nil
}

type mockLikeTrackUseCase struct {
	executeFunc func(ctx context.Context, input usecaseTrack.LikeTrackInput) error
}

func (m *mockLikeTrackUseCase) Execute(ctx context.Context, input usecaseTrack.LikeTrackInput) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return nil
}

type mockUnlikeTrackUseCase struct {
	executeFunc func(ctx context.Context, input usecaseTrack.UnlikeTrackInput) error
}

func (m *mockUnlikeTrackUseCase) Execute(ctx context.Context, input usecaseTrack.UnlikeTrackInput) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return nil
}

func TestTrackHandler_GetByID(t *testing.T) {
	trackID := uuid.New()
	artistID := uuid.New()

	t.Run("success", func(t *testing.T) {
		getTrackUC := &mockGetTrackUseCase{
			executeFunc: func(ctx context.Context, input usecaseTrack.GetTrackInput) (*grpcclient.TrackResponse, error) {
				return &grpcclient.TrackResponse{
					ID:              trackID,
					ArtistID:        artistID,
					Title:           "Track Name",
					DurationSeconds: 180,
					Status:          "ready",
					Price:           120,
					Currency:        "USD",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}, nil
			},
		}

		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, getTrackUC, nil, nil)

		r := chi.NewRouter()
		r.Get("/tracks/{id}", handler.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/tracks/"+trackID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, &mockGetTrackUseCase{}, nil, nil)

		r := chi.NewRouter()
		r.Get("/tracks/{id}", handler.GetByID)

		req := httptest.NewRequest(http.MethodGet, "/tracks/not-a-valid-uuid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}

func TestTrackHandler_LikeAndUnlike(t *testing.T) {
	trackID := uuid.New()
	userID := uuid.New()

	t.Run("like success", func(t *testing.T) {
		likeUC := &mockLikeTrackUseCase{
			executeFunc: func(ctx context.Context, input usecaseTrack.LikeTrackInput) error {
				if input.UserID != userID || input.TrackID != trackID {
					t.Errorf("got u=%v, tr=%v", input.UserID, input.TrackID)
				}
				return nil
			},
		}

		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, nil, likeUC, nil)

		r := chi.NewRouter()
		r.Post("/tracks/{id}/like", handler.LikeTrack)

		req := httptest.NewRequest(http.MethodPost, "/tracks/"+trackID.String()+"/like", nil)
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("like unauthorized", func(t *testing.T) {
		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, nil, &mockLikeTrackUseCase{}, nil)

		r := chi.NewRouter()
		r.Post("/tracks/{id}/like", handler.LikeTrack)

		req := httptest.NewRequest(http.MethodPost, "/tracks/"+trackID.String()+"/like", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("unlike success", func(t *testing.T) {
		unlikeUC := &mockUnlikeTrackUseCase{
			executeFunc: func(ctx context.Context, input usecaseTrack.UnlikeTrackInput) error {
				if input.UserID != userID || input.TrackID != trackID {
					t.Errorf("got u=%v, tr=%v", input.UserID, input.TrackID)
				}
				return nil
			},
		}

		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, nil, nil, unlikeUC)

		r := chi.NewRouter()
		r.Delete("/tracks/{id}/like", handler.UnlikeTrack)

		req := httptest.NewRequest(http.MethodDelete, "/tracks/"+trackID.String()+"/like", nil)
		ctx := contextutil.WithUserID(req.Context(), userID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
		}
	})
}
