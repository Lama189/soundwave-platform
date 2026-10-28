package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/Lama189/soundwave-platform/payment-service/internal/service/payment"
	"github.com/go-chi/chi/v5"
)

type WebhookHandler struct {
	service            PaymentService
	maxWebhookBodySize int64
}

func NewWebhookHandler(service PaymentService, maxWebhookBodySize int64) *WebhookHandler {
	if maxWebhookBodySize <= 0 {
		maxWebhookBodySize = 1024 * 1024
	}

	return &WebhookHandler{
		service:            service,
		maxWebhookBodySize: maxWebhookBodySize,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	providerParam := chi.URLParam(r, "provider")
	provider := domain.PaymentProvider(providerParam)
	if !provider.IsValid() {
		SendError(w, http.StatusBadRequest, "unsupported payment provider")
		return
	}

	signature := h.extractSignature(r, provider)

	r.Body = http.MaxBytesReader(w, r.Body, h.maxWebhookBodySize)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			SendError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		SendError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	if len(bodyBytes) == 0 {
		SendError(w, http.StatusBadRequest, "empty request body")
		return
	}

	err = h.service.HandleWebhook(r.Context(), payment.HandleWebhookInput{
		Provider:  provider,
		Payload:   bodyBytes,
		Signature: signature,
	})
	if err != nil {
		MapDomainError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, Response{Status: "ok"})
}

func (h *WebhookHandler) extractSignature(r *http.Request, provider domain.PaymentProvider) string {
	switch provider {
	case domain.PaymentProviderStripe:
		return r.Header.Get("Stripe-Signature")
	case domain.PaymentProviderMock:
		if sig := r.Header.Get("X-Mock-Signature"); sig != "" {
			return sig
		}
		return r.Header.Get("Stripe-Signature")
	default:
		return ""
	}
}
