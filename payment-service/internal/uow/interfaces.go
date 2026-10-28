package uow

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

type UnitOfWork interface {
	Payments() PaymentRepository
	WebhookEvents() WebhookEventRepository
	Outbox() OutboxRepository
}

type Manager interface {
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error
}

type OutboxRepository interface {
	Record(ctx context.Context, event *domain.Event) error
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetByExternalID(ctx context.Context, externalID string) (*domain.Payment, error)
	GetByExternalIDForUpdate(ctx context.Context, externalID string) (*domain.Payment, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error)
	Update(ctx context.Context, payment *domain.Payment) error
}

type WebhookEventRepository interface {
	Create(ctx context.Context, event *domain.WebhookEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookEvent, error)
	GetByProviderAndExternalID(ctx context.Context, provider domain.PaymentProvider, externalEventID string) (*domain.WebhookEvent, error)
	GetByProviderAndExternalIDForUpdate(ctx context.Context, provider domain.PaymentProvider, externalEventID string) (*domain.WebhookEvent, error)
	Update(ctx context.Context, event *domain.WebhookEvent) error
}
