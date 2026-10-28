package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
)

type Response struct {
	Status string `json:"status"`
}

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

func MapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidWebhookSignature):
		SendError(w, http.StatusUnauthorized, "invalid webhook signature")

	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrWebhookPayloadEmpty):
		SendError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrUnsupportedProvider):
		SendError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrNotFound):
		SendError(w, http.StatusNotFound, "payment entity not found")

	case errors.Is(err, domain.ErrPaymentAlreadyCompleted), errors.Is(err, domain.ErrWebhookAlreadyProcessed):
		SendJSON(w, http.StatusOK, Response{Status: "already_processed"})

	default:
		SendError(w, http.StatusInternalServerError, "internal server error")
	}
}
