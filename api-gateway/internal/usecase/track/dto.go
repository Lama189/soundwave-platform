package track

import (
	"errors"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/api-gateway/internal/grpcclient"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid track input")

type UploadTrackInput struct {
	UserID   uuid.UUID
	AlbumID  *uuid.UUID
	Title    string
	FileName string
}

type ConfirmUploadInput struct {
	TrackID uuid.UUID
	UserID  uuid.UUID
}

type CompleteProcessingInput struct {
	TrackID         uuid.UUID
	DurationSeconds int64
	FileKey         string
	PreviewKey      string
	CoverKey        string
}

type FailProcessingInput struct {
	TrackID uuid.UUID
}

type GetTrackStreamInput struct {
	TrackID uuid.UUID
	UserID  *uuid.UUID
}

func (in *GetTrackStreamInput) Validate() error {
	if in.TrackID == uuid.Nil {
		return fmt.Errorf("%w: track id is required", ErrInvalidInput)
	}

	if in.UserID != nil && *in.UserID == uuid.Nil {
		return fmt.Errorf("%w: user id cannot be empty", ErrInvalidInput)
	}

	return nil
}

type GetTrackStreamOutput struct {
	StreamURL string
	IsPreview bool
	ExpiresAt time.Time
}

func toTrackStreamOutput(res *grpcclient.TrackStreamResponse) *GetTrackStreamOutput {
	return &GetTrackStreamOutput{
		StreamURL: res.StreamURL,
		IsPreview: res.IsPreview,
		ExpiresAt: res.ExpiresAt,
	}
}
