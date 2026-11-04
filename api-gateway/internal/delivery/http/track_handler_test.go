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

		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, getTrackUC)

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
		handler := deliveryHttp.NewTrackHandler(nil, nil, nil, nil, nil, &mockGetTrackUseCase{})

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
