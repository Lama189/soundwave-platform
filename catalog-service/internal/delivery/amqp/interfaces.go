package amqp

import (
	"context"

	"github.com/google/uuid"
)

type TrackService interface {
	MarkProcessing(ctx context.Context, trackID uuid.UUID) error
}
