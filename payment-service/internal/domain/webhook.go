package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type WebhookStatus string

const (
	WebhookStatusReceived  WebhookStatus = "received"
	WebhookStatusProcessed WebhookStatus = "processed"
	WebhookStatusFailed    WebhookStatus = "failed"
	WebhookStatusIgnored   WebhookStatus = "ignored"
)

func (s WebhookStatus) IsValid() bool {
	switch s {
	case WebhookStatusReceived, WebhookStatusProcessed, WebhookStatusFailed, WebhookStatusIgnored:
		return true
	default:
		return false
	}
}

type WebhookEvent struct {
	ID              uuid.UUID
	Provider        PaymentProvider
	ExternalEventID string
	EventType       string
	Payload         []byte
	Status          WebhookStatus
	ErrorMessage    *string
	CreatedAt       time.Time
	ProcessedAt     *time.Time
}

func NewWebhookEvent(provider PaymentProvider, externalEventID string, eventType string, payload []byte) (*WebhookEvent, error) {
	var payloadCopy []byte
	if payload != nil {
		payloadCopy = make([]byte, len(payload))
		copy(payloadCopy, payload)
	}

	event := &WebhookEvent{
		ID:              uuid.New(),
		Provider:        provider,
		ExternalEventID: strings.TrimSpace(externalEventID),
		EventType:       strings.TrimSpace(eventType),
		Payload:         payloadCopy,
		Status:          WebhookStatusReceived,
		CreatedAt:       time.Now().UTC(),
	}

	if err := event.Validate(); err != nil {
		return nil, err
	}

	return event, nil
}

func RestoreWebhookEvent(
	id uuid.UUID,
	provider PaymentProvider,
	externalEventID, eventType string,
	payload []byte,
	status WebhookStatus,
	errorMessage *string,
	createdAt time.Time,
	processedAt *time.Time,
) (*WebhookEvent, error) {
	var payloadCopy []byte
	if payload != nil {
		payloadCopy = make([]byte, len(payload))
		copy(payloadCopy, payload)
	}

	event := &WebhookEvent{
		ID:              id,
		Provider:        provider,
		ExternalEventID: externalEventID,
		EventType:       eventType,
		Payload:         payloadCopy,
		Status:          status,
		ErrorMessage:    errorMessage,
		CreatedAt:       createdAt,
		ProcessedAt:     processedAt,
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("failed to restore corrupted webhook event: %w", err)
	}

	return event, nil
}

func (e *WebhookEvent) Validate() error {
	if e == nil {
		return ErrInvalidInput
	}

	if e.ID == uuid.Nil {
		return fmt.Errorf("%w: webhook event id is required", ErrInvalidInput)
	}

	if !e.Provider.IsValid() {
		return ErrUnsupportedProvider
	}

	if strings.TrimSpace(e.ExternalEventID) == "" {
		return fmt.Errorf("%w: external event id is required", ErrInvalidInput)
	}

	if strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("%w: event type is required", ErrInvalidInput)
	}

	if len(e.Payload) == 0 {
		return ErrWebhookPayloadEmpty
	}

	if !e.Status.IsValid() {
		return fmt.Errorf("%w: invalid webhook status: %s", ErrInvalidInput, e.Status)
	}

	if e.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at timestamp is required", ErrInvalidInput)
	}

	if e.Status != WebhookStatusReceived && e.ProcessedAt == nil {
		return fmt.Errorf("%w: processed_at timestamp is required for terminal status", ErrInvalidInput)
	}

	return nil
}

func (w *WebhookEvent) IsTerminal() bool {
	if w == nil {
		return false
	}

	switch w.Status {
	case WebhookStatusProcessed, WebhookStatusFailed, WebhookStatusIgnored:
		return true
	default:
		return false
	}
}

func (w *WebhookEvent) MarkProcessed() error {
	if err := w.ensureTransitionable(); err != nil {
		return err
	}

	now := time.Now().UTC()
	w.Status = WebhookStatusProcessed
	w.ErrorMessage = nil
	w.ProcessedAt = &now

	return nil
}

func (w *WebhookEvent) MarkFailed(err error) error {
	if transitionErr := w.ensureTransitionable(); transitionErr != nil {
		return transitionErr
	}

	msg := "unknown webhook processing failure"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		msg = err.Error()
	}

	now := time.Now().UTC()
	w.Status = WebhookStatusFailed
	w.ErrorMessage = &msg
	w.ProcessedAt = &now

	return nil
}

func (w *WebhookEvent) MarkIgnored(reason string) error {
	if err := w.ensureTransitionable(); err != nil {
		return err
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "event ignored by handler"
	}

	now := time.Now().UTC()
	w.Status = WebhookStatusIgnored
	w.ErrorMessage = &reason
	w.ProcessedAt = &now

	return nil
}

func (w *WebhookEvent) ensureTransitionable() error {
	if w == nil {
		return ErrNotFound
	}

	if w.IsTerminal() {
		return ErrWebhookAlreadyProcessed
	}

	return nil
}
