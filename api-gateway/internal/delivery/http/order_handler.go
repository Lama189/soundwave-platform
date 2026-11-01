package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/usecase/order"
)

type OrderHandler struct {
	createOrder CreateOrderUseCase
}

func NewOrderHandler(createOrder CreateOrderUseCase) *OrderHandler {
	return &OrderHandler{
		createOrder: createOrder,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	buyerUserID, ok := UserIDFromContext(r.Context())
	if !ok {
		SendError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := req.ToInput(buyerUserID)

	output, err := h.createOrder.Execute(r.Context(), input)
	if err != nil {
		if errors.Is(err, order.ErrInvalidInput) {
			SendError(w, http.StatusBadRequest, err.Error())
			return
		}

		HandleGRPCError(w, err)
		return
	}

	SendJSON(w, http.StatusCreated, ToOrderResponse(output))
}
