package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AlbumHandler struct {
	client AlbumClient
}

func NewAlbumHandler(client AlbumClient) *AlbumHandler {
	return &AlbumHandler{client: client}
}

type CreateAlbumRequestBody struct {
	ArtistID    uuid.UUID `json:"artist_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CoverKey    string    `json:"cover_key"`
	Price       int64     `json:"price"`
	Currency    string    `json:"currency"`
}

func (h *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAlbumRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ArtistID == uuid.Nil {
		SendError(w, http.StatusBadRequest, "artist id is required")
		return
	}
	if req.Title == "" {
		SendError(w, http.StatusBadRequest, "album title is required")
		return
	}
	if req.Price < 0 {
		SendError(w, http.StatusBadRequest, "price cannot be negative")
		return
	}

	album, err := h.client.CreateAlbum(r.Context(), grpcclient.CreateAlbumRequest{
		UserID:      userID,
		ArtistID:    req.ArtistID,
		Title:       req.Title,
		Description: req.Description,
		CoverKey:    req.CoverKey,
		Price:       req.Price,
		Currency:    req.Currency,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, album)
}

func (h *AlbumHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid album id")
		return
	}

	album, err := h.client.GetAlbum(r.Context(), albumID)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, album)
}

func (h *AlbumHandler) GetByArtistID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	artistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid artist id")
		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	albums, err := h.client.GetAlbumsByArtistID(r.Context(), artistID, int32(limit), int32(offset))
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, albums)
}

type UpdateAlbumRequestBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CoverKey    string `json:"cover_key"`
	Price       int64  `json:"price"`
	Currency    string `json:"currency"`
}

func (h *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid album id")
		return
	}

	var req UpdateAlbumRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		SendError(w, http.StatusBadRequest, "album title is required")
		return
	}
	if req.Price < 0 {
		SendError(w, http.StatusBadRequest, "price cannot be negative")
		return
	}

	album, err := h.client.UpdateAlbum(r.Context(), grpcclient.UpdateAlbumRequest{
		AlbumID:     albumID,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		CoverKey:    req.CoverKey,
		Price:       req.Price,
		Currency:    req.Currency,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, album)
}

func (h *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid album id")
		return
	}

	if err := h.client.DeleteAlbum(r.Context(), albumID, userID); err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
