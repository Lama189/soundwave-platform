package worker

import (
	"context"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository interface {
	FetchUnprocessed(ctx context.Context, limit int) ([]*domain.Event, error)
	MarkProcessed(ctx context.Context, ids []uuid.UUID) error
}
