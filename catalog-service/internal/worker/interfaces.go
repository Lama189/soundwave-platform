package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/domain"
)

type OutboxRepository interface {
	ClaimUnprocessed(ctx context.Context, limit int, lockDuration time.Duration, workerID string) ([]*domain.Event, error)
	MarkProcessed(ctx context.Context, ids []uuid.UUID) error
	RecordFailure(ctx context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time, maxRetries int) error
	CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
}
