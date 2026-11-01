package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	usecaseArtist "github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/artist"
)

type ArtistHandler struct {
	client       ArtistClient
	becomeArtist BecomeArtistUseCase
}

func NewArtistHandler(client ArtistClient, becomeArtist BecomeArtistUseCase) *ArtistHandler {
	return &ArtistHandler{
		client:       client,
		becomeArtist: becomeArtist,
	}
}

func (h *ArtistHandler) BecomeArtist(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req BecomeArtistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		SendError(w, http.StatusBadRequest, "artist name is required")
		return
	}

	artist, err := h.becomeArtist.Execute(r.Context(), usecaseArtist.BecomeArtistInput{
		UserID: userID,
		Name:   req.Name,
		Bio:    req.Bio,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, artist)
}

func (h *ArtistHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	artistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid artist id")
		return
	}

	artist, err := h.client.GetArtist(r.Context(), artistID)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, artist)
}

func (h *ArtistHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	artist, err := h.client.GetArtistByUserId(r.Context(), userID)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, artist)
}
