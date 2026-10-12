package track

import "github.com/google/uuid"

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
