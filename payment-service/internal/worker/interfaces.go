package worker

import (
	"context"
	"time"

	"github.com/Lama189/soundwave-platform/payment-service/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository interface {
	ClaimUnprocessed(ctx context.Context, limit int, lockDuration time.Duration, workerID string) ([]*domain.Event, error)
	MarkProcessed(ctx context.Context, ids []uuid.UUID) error
	RecordFailure(ctx context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time, maxRetries int) error
	CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
}
