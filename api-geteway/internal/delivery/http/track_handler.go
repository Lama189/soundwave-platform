package http

import (
	"encoding/json"
	"net/http"

	usecaseTrack "github.com/Lama189/soundwave-platform/api-geteway/internal/usecase/track"
)

type TrackHandler struct {
	uploadTrack UploadTrackUseCase
}

func NewTrackHandler(uploadTrack UploadTrackUseCase) *TrackHandler {
	return &TrackHandler{
		uploadTrack: uploadTrack,
	}
}

func (h *TrackHandler) UploadTrack(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UploadTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		SendError(w, http.StatusBadRequest, "track title is required")
		return
	}
	if req.FileName == "" {
		SendError(w, http.StatusBadRequest, "file name is required")
		return
	}

	res, err := h.uploadTrack.Execute(r.Context(), usecaseTrack.UploadTrackInput{
		UserID:   userID,
		AlbumID:  req.AlbumID,
		Title:    req.Title,
		FileName: req.FileName,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, res)
}
