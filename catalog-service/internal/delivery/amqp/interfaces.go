package amqp

import (
	"context"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/service/track"
	"github.com/google/uuid"
)

type TrackService interface {
	MarkProcessing(ctx context.Context, trackID uuid.UUID) error
	CompleteProcessing(ctx context.Context, trackID uuid.UUID, dto track.CompleteProcessingDTO) (*track.TrackOutputDTO, error)
	FailProcessing(ctx context.Context, trackID uuid.UUID) error
}
