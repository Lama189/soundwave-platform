package http

import (
	"net/http"

	"github.com/Lama189/soundwave-platform/api-geteway/internal/infrastructure/jwt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	userHandler *UserHandler,
	artistHandler *ArtistHandler,
	trackHandler *TrackHandler,
	tokenValidator jwt.TokenValidator,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		SendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
			r.Post("/refresh", userHandler.Refresh)
		})

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(tokenValidator))
			r.Get("/users/me", userHandler.GetMe)
		})

		r.Route("/artists", func(r chi.Router) {
			r.Get("/{id}", artistHandler.GetByID)

			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(tokenValidator))
				r.Post("/become", artistHandler.BecomeArtist)
				r.Get("/me", artistHandler.GetMe)
			})
		})

		r.Route("/tracks", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(AuthMiddleware(tokenValidator))
				r.Post("/upload", trackHandler.UploadTrack)
			})
		})
	})

	return r
}
