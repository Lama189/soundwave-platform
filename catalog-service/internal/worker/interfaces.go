package worker

import (
	"context"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
)

type OutboxRepository interface {
	FetchUnprocessed(ctx context.Context, limit int) ([]*domain.Event, error)
	MarkProcessed(ctx context.Context, ids []uuid.UUID) error
}
