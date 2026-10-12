package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	usecaseTrack "github.com/Lama189/soundwave-platform/api-geteway/internal/usecase/track"
)

type TrackHandler struct {
	uploadTrack        UploadTrackUseCase
	confirmUpload      ConfirmUploadUseCase
	completeProcessing CompleteProcessingUseCase
	failProcessing     FailProcessingUseCase
}

func NewTrackHandler(
	uploadTrack UploadTrackUseCase,
	confirmUpload ConfirmUploadUseCase,
	completeProcessing CompleteProcessingUseCase,
	failProcessing FailProcessingUseCase,
) *TrackHandler {
	return &TrackHandler{
		uploadTrack:        uploadTrack,
		confirmUpload:      confirmUpload,
		completeProcessing: completeProcessing,
		failProcessing:     failProcessing,
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

func (h *TrackHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	trackID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	res, err := h.confirmUpload.Execute(r.Context(), usecaseTrack.ConfirmUploadInput{
		TrackID: trackID,
		UserID:  userID,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, res)
}

func (h *TrackHandler) CompleteProcessing(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	var req CompleteProcessingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FileKey == "" {
		SendError(w, http.StatusBadRequest, "file key is required")
		return
	}

	res, err := h.completeProcessing.Execute(r.Context(), usecaseTrack.CompleteProcessingInput{
		TrackID:         trackID,
		DurationSeconds: req.DurationSeconds,
		FileKey:         req.FileKey,
		PreviewKey:      req.PreviewKey,
		CoverKey:        req.CoverKey,
	})
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, res)
}

func (h *TrackHandler) FailProcessing(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, err := uuid.Parse(idStr)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	if err := h.failProcessing.Execute(r.Context(), usecaseTrack.FailProcessingInput{
		TrackID: trackID,
	}); err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, map[string]string{"status": "failed"})
}
