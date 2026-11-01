package http

import (
	"encoding/json"
	"net/http"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
)

type UserHandler struct {
	client UserClient
}

func NewUserHandler(client UserClient) *UserHandler {
	return &UserHandler{client: client}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req grpcclient.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.client.Register(r.Context(), req)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, res)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req grpcclient.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.client.Login(r.Context(), req)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, res)
}

func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req grpcclient.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.client.Refresh(r.Context(), req)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, res)
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.client.GetMe(r.Context(), userID)
	if err != nil {
		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, res)
}
