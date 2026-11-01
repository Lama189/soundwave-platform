package http

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func SendError(w http.ResponseWriter, statusCode int, message string) {
	SendJSON(w, statusCode, ErrorResponse{Error: message})
}

func HandleGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		SendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		SendError(w, http.StatusBadRequest, st.Message())
	case codes.NotFound:
		SendError(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists:
		SendError(w, http.StatusConflict, st.Message())
	case codes.Unauthenticated:
		SendError(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		SendError(w, http.StatusForbidden, st.Message())
	case codes.DeadlineExceeded:
		SendError(w, http.StatusGatewayTimeout, "upstream request timeout")
	case codes.Unavailable:
		SendError(w, http.StatusServiceUnavailable, "service temporarily unavailable")
	default:
		SendError(w, http.StatusInternalServerError, "internal server error")
	}
}
