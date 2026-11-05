package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PlaylistHandler struct {
	client PlaylistClient
}

func NewPlaylistHandler(client PlaylistClient) *PlaylistHandler {
	return &PlaylistHandler{client: client}
}

type CreatePlaylistRequestBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

func (h *PlaylistHandler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreatePlaylistRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		SendError(w, http.StatusBadRequest, "playlist title is required")
		return
	}

	p, err := h.client.CreatePlaylist(r.Context(), grpcclient.CreatePlaylistRequest{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, p)
}

func (h *PlaylistHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	var userID *uuid.UUID
	if id, ok := UserIDFromContext(r.Context()); ok {
		userID = &id
	}

	p, err := h.client.GetPlaylist(r.Context(), playlistID, userID)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, p)
}

func (h *PlaylistHandler) GetMyPlaylists(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
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

	playlists, err := h.client.GetMyPlaylists(r.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, playlists)
}

type AddTrackRequestBody struct {
	TrackID  uuid.UUID `json:"track_id"`
	Position int32     `json:"position"`
}

func (h *PlaylistHandler) AddTrack(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	var req AddTrackRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TrackID == uuid.Nil {
		SendError(w, http.StatusBadRequest, "track_id is required")
		return
	}

	err = h.client.AddTrackToPlaylist(r.Context(), grpcclient.AddTrackToPlaylistRequest{
		UserID:     userID,
		PlaylistID: playlistID,
		TrackID:    req.TrackID,
		Position:   req.Position,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "track added"})
}

func (h *PlaylistHandler) RemoveTrack(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	trackIDStr := chi.URLParam(r, "track_id")
	trackID, err := uuid.Parse(trackIDStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	if err := h.client.RemoveTrackFromPlaylist(r.Context(), userID, playlistID, trackID); err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "track removed"})
}

type ChangePositionRequestBody struct {
	NewPosition int32 `json:"new_position"`
}

func (h *PlaylistHandler) ChangeTrackPosition(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	trackIDStr := chi.URLParam(r, "track_id")
	trackID, err := uuid.Parse(trackIDStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	var req ChangePositionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.client.ChangeTrackPosition(r.Context(), grpcclient.ChangePlaylistTrackPositionRequest{
		UserID:      userID,
		PlaylistID:  playlistID,
		TrackID:     trackID,
		NewPosition: req.NewPosition,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "position updated"})
}

type UpdatePlaylistRequestBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

func (h *PlaylistHandler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	var req UpdatePlaylistRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		SendError(w, http.StatusBadRequest, "playlist title is required")
		return
	}

	p, err := h.client.UpdatePlaylist(r.Context(), grpcclient.UpdatePlaylistRequest{
		UserID:      userID,
		PlaylistID:  playlistID,
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, p)
}

func (h *PlaylistHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	playlistID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	if err := h.client.DeletePlaylist(r.Context(), userID, playlistID); err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
